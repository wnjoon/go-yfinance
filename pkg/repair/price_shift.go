package repair

import (
	"math"
)

// This file holds the range-based price-shift helpers shared by split repair
// and unit-switch repair. They port the Python yfinance 1.8.0 helpers that
// _fix_prices_sudden_change is built from:
//
//   - map_signals_to_ranges            -> mapSignalsToRanges
//   - _estimate_volume_shift_threshold -> estimateVolumeShiftThreshold
//   - _reconcile_price_shifts_with_volume -> reconcilePriceShiftsWithVolume
//
// Like upstream, every slice here is in REVERSE date order (newest first):
// position k compares row k (older) with row k-1 (newer), so position 0 is
// never a signal. Callers map positions back to their oldest-first bars.

// priceShiftRange is a half-open [start, end) range of newest-first row
// positions whose prices must be multiplied by multiplier, and volume by
// volumeMultiplier, to undo a level shift. Upstream stores this as the tuple
// (start, end, 'split' | '1.0/split') and derives m / m_rcp from it; both are
// kept so volume rounding sees the same operands as upstream.
type priceShiftRange struct {
	start, end       int
	multiplier       float64
	volumeMultiplier float64
}

// rangeVolumeSampleTarget is upstream's "> 10 positive values on each side"
// pooling goal in _reconcile_price_shifts_with_volume.
const rangeVolumeSampleTarget = 10

// rangeVolumeStdDevFallback is the stock-split fallback added in 1.8.0
// (5b63983): a range whose volume mean moved by more than this many standard
// deviations is kept even if the ratio misses the threshold.
const rangeVolumeStdDevFallback = 15

// mapSignalsToRanges ports upstream's map_signals_to_ranges. Signals pair up
// into alternating [signal, next signal) ranges; an odd final signal extends
// to the oldest row. A down signal means the older side is too LOW, so prices
// are multiplied by max(change, 1/change); an up signal by the reciprocal.
// Position 0 is ignored, as upstream forces it false.
func mapSignalsToRanges(up, down []bool, change float64) []priceShiftRange {
	indices := make([]int, 0)
	for i := 1; i < len(up); i++ {
		if up[i] || down[i] {
			indices = append(indices, i)
		}
	}
	split, splitRcp := change, 1/change
	ranges := make([]priceShiftRange, 0, (len(indices)+1)/2)
	for pair := 0; pair < len(indices); pair += 2 {
		start, end := indices[pair], len(up)
		if pair+1 < len(indices) {
			end = indices[pair+1]
		}
		// Upstream: split > 1 -> 'split' if down else '1.0/split';
		//           split <= 1 -> '1.0/split' if down else 'split'.
		r := priceShiftRange{start: start, end: end, multiplier: splitRcp, volumeMultiplier: split}
		if (split > 1) == down[start] {
			r.multiplier, r.volumeMultiplier = split, splitRcp
		}
		ranges = append(ranges, r)
	}
	return ranges
}

// estimateVolumeShiftThreshold ports _estimate_volume_shift_threshold. vol and
// the ORIGINAL (pre-screening) candidate masks are newest-first. Volume is
// denoised separately within each block delimited by candidates; upstream
// stores the denoised values in an integer array, so they are truncated
// toward zero. The second result is false when there are no candidates or no
// volume, matching upstream's (None, None).
func estimateVolumeShiftThreshold(vol []float64, up, down []bool, change float64, interval string) (float64, bool) {
	n := len(vol)
	indices := make([]int, 0)
	for i := 0; i < n && i < len(up); i++ {
		if up[i] || down[i] {
			indices = append(indices, i)
		}
	}
	if len(indices) == 0 || allVolumesZero(vol) {
		return math.NaN(), false
	}

	denoised := make([]float64, n)
	fill := func(lo, hi int) {
		for j, v := range denoiseVolume(vol[lo:hi]) {
			denoised[lo+j] = math.Trunc(v)
		}
	}
	fill(0, indices[0])
	for i, lo := range indices {
		hi := n
		if i+1 < len(indices) {
			hi = indices[i+1]
		}
		fill(lo, hi)
	}

	ratios := make([]float64, n)
	ratios[0] = 1
	for k := 1; k < n; k++ {
		if denoised[k-1] == 0 {
			ratios[k] = 1
		} else {
			ratios[k] = denoised[k] / denoised[k-1]
		}
	}

	q1, q3 := npPercentile(ratios, 25), npPercentile(ratios, 75)
	iqr := q3 - q1
	normal := make([]float64, 0, n)
	for _, v := range ratios {
		if v >= q1-1.5*iqr && v <= q3+1.5*iqr {
			normal = append(normal, v)
		}
	}
	sdPct := npStd(normal, 0) / npMean(normal)
	largestVolChgPct := scaleLargestChange(5*sdPct, interval)
	changeMax := pyMax(change, 1/change)
	return 1 + (changeMax-1+largestVolChgPct)*0.2, true
}

// reconcilePriceShiftsWithVolume ports _reconcile_price_shifts_with_volume
// (Python 1.8.0, upstream commits aec8332 and 5b63983). vol and the screened
// candidate masks are newest-first; it returns the volume-confirmed ranges
// without mutating its input.
//
// Ranges are visited from the last (oldest) to the first (newest), i.e.
// len-1 -> 0, and a rejected range is deleted immediately, so ranges visited
// afterwards no longer pool samples from it (see rangeVolumeSamples). Both
// samples are denoised and the boundary change is mean(during)/mean(outside).
//
// Split (unitSwitch=false): keep when volume moves inversely to price
// (change < 1/threshold with an up signal, or > threshold with a down
// signal), or when both samples have >= 4 values and the mean difference
// exceeds 15 standard deviations of either sample; otherwise drop.
// Unit switch: drop when volume moves inversely to price, otherwise keep.
// Ranges without usable volume on either side are always kept.
func reconcilePriceShiftsWithVolume(vol []float64, ranges []priceShiftRange, up, down []bool, threshold float64, unitSwitch bool) []priceShiftRange {
	ranges = append([]priceShiftRange(nil), ranges...)
	for i := len(ranges) - 1; i >= 0; i-- {
		during, outside := rangeVolumeSamples(vol, ranges, i)
		duringDenoised := denoiseVolume(during)
		outsideDenoised := denoiseVolume(outside)
		if len(duringDenoised) == 0 || len(outsideDenoised) == 0 {
			// No volume to check, but this should be incredibly rare.
			continue
		}
		r := ranges[i]
		if !rangeVolumeConfirms(duringDenoised, outsideDenoised, up[r.start], down[r.start], threshold, unitSwitch) {
			ranges = append(ranges[:i], ranges[i+1:]...)
		}
	}
	return ranges
}

// rangeVolumeSamples builds the positive "during" and "outside" volume samples
// for ranges[i], exactly as upstream's reconcile loop does. "during" starts as
// the range itself; "outside" starts as up to 10 rows beyond the range when it
// is the first (newer side) or last (older side) range. While either side has
// at most 10 positive values, step outwards: prepend the volume of the ranges
// step positions before and after i to "during", and the gaps that separate
// them from their neighbour towards i to "outside". Non-positive volumes are
// dropped at the end.
func rangeVolumeSamples(vol []float64, ranges []priceShiftRange, i int) (during, outside []float64) {
	r := ranges[i]
	during = volumeSegment(vol, r.start, r.end)
	outside = []float64{}
	if i == 0 && r.start > 0 {
		outside = volumeSegment(vol, maxInt(0, r.start-rangeVolumeSampleTarget), r.start)
	} else if i == len(ranges)-1 && r.end < len(vol) {
		outside = volumeSegment(vol, r.end, minInt(r.end+rangeVolumeSampleTarget, len(vol)))
	}
	for step := 1; step < len(ranges); step++ {
		if countPositive(outside) > rangeVolumeSampleTarget && countPositive(during) > rangeVolumeSampleTarget {
			break
		}
		// Upstream stops at "> 10" but extends only at "< 10": a side with
		// exactly 10 positive values is neither extended nor complete.
		if i2 := i - step; i2 >= 0 {
			r2 := ranges[i2]
			if countPositive(during) < rangeVolumeSampleTarget {
				during = append(volumeSegment(vol, r2.start, r2.end), during...)
			}
			if countPositive(outside) < rangeVolumeSampleTarget {
				outside = append(volumeSegment(vol, r2.end, ranges[i2+1].start), outside...)
			}
		}
		if i3 := i + step; i3 < len(ranges) {
			r3 := ranges[i3]
			if countPositive(during) < rangeVolumeSampleTarget {
				during = append(volumeSegment(vol, r3.start, r3.end), during...)
			}
			if countPositive(outside) < rangeVolumeSampleTarget {
				outside = append(volumeSegment(vol, ranges[i3-1].end, r3.start), outside...)
			}
		}
	}
	return positiveValues(during), positiveValues(outside)
}

// rangeVolumeConfirms is the per-range decision of
// reconcilePriceShiftsWithVolume, given non-empty denoised samples.
func rangeVolumeConfirms(during, outside []float64, up, down bool, threshold float64, unitSwitch bool) bool {
	duringMean, outsideMean := npMean(during), npMean(outside)
	change := duringMean / outsideMean
	inverseMove := (change < 1/threshold && up) || (change > threshold && down)
	if unitSwitch {
		// A currency-unit switch leaves volume unchanged; an inverse volume
		// move means a genuine corporate action.
		return !inverseMove
	}
	if inverseMove {
		return true
	}
	return volumeStdDevMultiple(during, outside, duringMean, outsideMean) > rangeVolumeStdDevFallback
}

// volumeStdDevMultiple returns upstream's volChg_StdDevMultiple: 1 unless both
// samples have at least 4 values, else max(|Δmean|/sd_during,
// |Δmean|/sd_outside) with population standard deviations. Float division by
// a zero deviation yields +Inf (or NaN for 0/0) exactly like NumPy, and the
// two-argument max keeps Python's builtin semantics: a NaN first operand wins,
// a NaN second operand is ignored.
func volumeStdDevMultiple(during, outside []float64, duringMean, outsideMean float64) float64 {
	if len(during) < 4 || len(outside) < 4 {
		return 1
	}
	diff := math.Abs(duringMean - outsideMean)
	return pyMax(diff/npStd(during, 0), diff/npStd(outside, 0))
}

// volumeSegment copies vol[lo:hi], returning an empty slice when the bounds
// are empty or inverted (NumPy slicing semantics).
func volumeSegment(vol []float64, lo, hi int) []float64 {
	lo, hi = maxInt(lo, 0), minInt(hi, len(vol))
	if lo >= hi {
		return []float64{}
	}
	return append([]float64(nil), vol[lo:hi]...)
}

func countPositive(values []float64) int {
	n := 0
	for _, v := range values {
		if v > 0 {
			n++
		}
	}
	return n
}
