package repair

import (
	"math"

	"github.com/wnjoon/go-yfinance/pkg/models"
	"github.com/wnjoon/go-yfinance/pkg/stats"
)

// barVolumes extracts the volume series as floats.
func barVolumes(bars []models.Bar) []float64 {
	vol := make([]float64, len(bars))
	for i, bar := range bars {
		vol[i] = float64(bar.Volume)
	}
	return vol
}

func allVolumesZero(vol []float64) bool {
	for _, v := range vol {
		if v != 0 {
			return false
		}
	}
	return true
}

// denoiseVolume smooths a volume series for boundary comparison: zeros are
// backward- then forward-filled, then a sliding median with window <= 9 is
// taken. Port of upstream _fix_prices_sudden_change's denoise_volume.
func denoiseVolume(vol []float64) []float64 {
	n := len(vol)
	if n == 0 {
		return nil
	}

	filled := make([]float64, n)
	copy(filled, vol)
	for i := n - 2; i >= 0; i-- {
		if filled[i] == 0 {
			filled[i] = filled[i+1]
		}
	}
	for i := 1; i < n; i++ {
		if filled[i] == 0 {
			filled[i] = filled[i-1]
		}
	}

	w := n
	if w > 9 {
		w = 9
	}
	if w%2 == 0 {
		w--
	}
	pad := w / 2

	out := make([]float64, n)
	window := make([]float64, 0, w)
	for i := 0; i < n; i++ {
		lo := maxInt(0, i-pad)
		hi := minInt(n, i+pad+1)
		window = window[:0]
		for j := lo; j < hi; j++ {
			if filled[j] > 0 && !math.IsNaN(filled[j]) {
				window = append(window, filled[j])
			}
		}
		if len(window) == 0 {
			out[i] = 0
			continue
		}
		out[i] = stats.Median(window)
	}
	return out
}

// volumeChangeThreshold returns the ratio above which a one-bar volume change
// counts as abnormal, scaled to the candidate price-change size (upstream
// threshold_volUnitChg = 1 + (split_max - 1 + largest_volChg_pct) * 0.333).
func volumeChangeThreshold(vol []float64, changeMax float64, interval string) float64 {
	denoised := denoiseVolume(vol)

	changes := make([]float64, 0, len(denoised))
	for i := 1; i < len(denoised); i++ {
		if denoised[i-1] > 0 {
			changes = append(changes, denoised[i]/denoised[i-1])
		} else {
			changes = append(changes, 1.0)
		}
	}

	largestVolChgPct := 0.0
	if len(changes) >= 3 {
		q1, q3, iqr := stats.IQR(changes)
		if !math.IsNaN(iqr) {
			normal := boundedChanges(changes, q1-1.5*iqr, q3+1.5*iqr)
			if len(normal) > 0 {
				avg := stats.Mean(normal)
				sd := stats.Std(normal, 1)
				if !math.IsNaN(sd) && avg != 0 {
					largestVolChgPct = 5 * sd / avg
				}
			}
		}
	}

	// Coarser intervals aggregate more volume noise (upstream: x3 for interday
	// above 1d, x2 more for months).
	switch interval {
	case "1wk", "5d":
		largestVolChgPct *= 3
	case "1mo", "3mo":
		largestVolChgPct *= 6
	}

	return 1 + (changeMax-1+largestVolChgPct)*0.333
}

// boundaryVolumeChange returns the denoised volume just after a boundary
// divided by the denoised volume just before it, or NaN when either side has
// no usable volume.
func boundaryVolumeChange(vol []float64, boundary int) float64 {
	if boundary <= 0 || boundary >= len(vol) {
		return math.NaN()
	}
	before := positiveValues(denoiseVolume(vol[:boundary]))
	during := positiveValues(denoiseVolume(vol[boundary:]))
	if len(before) == 0 || len(during) == 0 {
		return math.NaN()
	}
	return during[0] / before[len(before)-1]
}

func positiveValues(values []float64) []float64 {
	out := make([]float64, 0, len(values))
	for _, v := range values {
		if v > 0 {
			out = append(out, v)
		}
	}
	return out
}

// unitSwitchVolumeOK cross-checks a candidate unit switch against volume: a
// genuine currency-unit switch leaves volume roughly unchanged, while a real
// split or reverse split moves volume sharply opposite to price (upstream
// #2943). Unusable volume (NaN change) does not veto.
func unitSwitchVolumeOK(vol []float64, boundary int, priceUp bool, volThreshold float64) bool {
	change := boundaryVolumeChange(vol, boundary)
	if math.IsNaN(change) {
		return true
	}
	if priceUp && change < 1/volThreshold {
		// Price up ~100x while volume collapsed: a real reverse split.
		return false
	}
	if !priceUp && change > volThreshold {
		// Price down ~100x while volume surged: a real forward split.
		return false
	}
	return true
}

// splitVolumeConfirms requires the mirror-image volume jump a genuinely
// unadjusted split produces at the split date (upstream #2943: stock-split
// repair REQUIRES a big volume change of matching sign).
func splitVolumeConfirms(bars []models.Bar, splitIdx int, splitRatio float64, interval string) bool {
	vol := barVolumes(bars)
	if allVolumesZero(vol) {
		return false
	}

	change := boundaryVolumeChange(vol, splitIdx)
	if math.IsNaN(change) {
		return false
	}

	changeMax := math.Max(splitRatio, 1/splitRatio)
	volThreshold := volumeChangeThreshold(vol, changeMax, interval)

	if splitRatio > 1 {
		// Unadjusted forward split: price drops, volume jumps.
		return change > volThreshold
	}
	// Unadjusted reverse split: price jumps, volume collapses.
	return change < 1/volThreshold
}
