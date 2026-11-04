package repair

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

// This file ports the candidate stages of Python yfinance 1.8.0
// _fix_prices_sudden_change, shared by split, unit-switch and
// unexplained-level-shift repair:
//
//   - _detect_price_shifts                         -> detectPriceShifts
//   - _filter_price_shifts_on_volume_spikes        -> filterPriceShiftsOnVolumeSpikes
//   - _filter_price_shifts_that_match_local_stdev  -> filterPriceShiftsThatMatchLocalStdev
//
// Rows are in the order the caller passes them; _fix_prices_sudden_change
// passes them newest first, so row i compares its prices with the newer row
// i-1 and row 0 is never a signal.

// priceShiftDetection ports upstream's _PriceShiftDetection namedtuple.
type priceShiftDetection struct {
	// up and down are indexed [column][row]. There is one column, or four
	// (Open, High, Low, Close) when columns are corrected individually.
	up, down [][]bool
	// change1D is the workings "1D %" column: the denoised day-to-day change
	// rounded to 3 decimals, which the local-volatility filter reads.
	change1D []float64
	// flag is the workings "f" column (up or down), single-column only.
	flag []bool
	// threshold is the detection threshold (min_change without a change).
	threshold float64
}

func (d *priceShiftDetection) perColumn() bool { return len(d.up) > 1 }

// rowSignal reports whether any column of row i is an up (or down) signal.
func rowSignal(m [][]bool, i int) bool {
	for _, col := range m {
		if col[i] {
			return true
		}
	}
	return false
}

// anySignal reports whether any mask cell is set.
func anySignal(masks ...[][]bool) bool {
	for _, m := range masks {
		for _, col := range m {
			for _, v := range col {
				if v {
					return true
				}
			}
		}
	}
	return false
}

func cloneMask(m [][]bool) [][]bool {
	out := make([][]bool, len(m))
	for c := range m {
		out[c] = append([]bool(nil), m[c]...)
	}
	return out
}

// isInterday is upstream's interday: 1d, 1wk, 1mo and 3mo (not 5d).
func isInterday(interval string) bool {
	switch interval {
	case "1d", "1wk", "1mo", "3mo":
		return true
	}
	return false
}

// scaleLargestChange applies upstream's largest_change_pct multipliers:
// x3 for interday intervals above 1d, and x2 more for 1mo/3mo.
func scaleLargestChange(pct float64, interval string) float64 {
	if isInterday(interval) && interval != "1d" {
		pct *= 3
		if interval == "1mo" || interval == "3mo" {
			pct *= 2
		}
	}
	return pct
}

func ohlcValue(b models.Bar, j int) float64 {
	switch j {
	case 0:
		return b.Open
	case 1:
		return b.High
	case 2:
		return b.Low
	default:
		return b.Close
	}
}

// adjustmentFactor is the dividend-adjustment factor Adj Close / Close used
// on detector prices. As upstream, it is 1 when Close is 0, and a NaN Adj
// Close yields NaN (so that row's changes denoise to "no change"). Go has no
// separate "missing" value for a float field, so an AdjClose of exactly 0 --
// the zero value of bars built without Adj Close, e.g. by direct callers of
// [Repairer.Repair] -- means "unadjusted" (factor 1) instead of scaling every
// price to 0, which would disable detection.
func adjustmentFactor(b models.Bar) float64 {
	if b.Close == 0 || b.AdjClose == 0 {
		return 1
	}
	return b.AdjClose / b.Close
}

// detectPriceShifts ports _detect_price_shifts. change == 0 means upstream's
// change=None: signals are day-to-day changes beyond minChange. Otherwise the
// threshold depends on change and the global price volatility. It returns
// nil when there is no candidate (upstream None). Rows must have no missing
// OHLC.
func detectPriceShifts(rows []models.Bar, interval string, change float64, perColumn bool, minChange float64) *priceShiftDetection {
	n := len(rows)
	if n == 0 {
		return nil
	}
	hasChange := change != 0
	if !hasChange && minChange <= 1 {
		return nil // upstream raises ValueError
	}
	split, splitRcp, splitMax := change, 1.0, minChange
	if hasChange {
		splitRcp = 1 / split
		splitMax = pyMax(split, splitRcp)
	}
	multiday := isInterday(interval) && interval != "1d"

	// Daily change of each price column, on dividend-adjusted prices. For
	// multi-day intervals High and Low are too volatile, so only Open and
	// Close are used (except for 100x / 0.001x changes).
	cols := []int{0, 1, 2, 3}
	if multiday && !(hasChange && (split == 100 || split == 0.001)) {
		cols = []int{0, 3}
	}
	prices := make([][]float64, n)
	for i, b := range rows {
		prices[i] = make([]float64, len(cols))
		adj := adjustmentFactor(b)
		for k, j := range cols {
			v := ohlcValue(b, j)
			if v == 0 {
				v = 1
			}
			prices[i][k] = v * adj
		}
	}
	changes := make([][]float64, n)
	for i := range changes {
		changes[i] = make([]float64, len(cols))
		for k := range cols {
			if i == 0 || rows[i].Close == 0 || rows[(i-1+n)%n].Close == 0 {
				// Upstream np.roll wraps, so row 0 also checks the last row.
				changes[i][k] = 1
			} else {
				changes[i][k] = prices[i][k] / prices[i-1][k]
			}
		}
	}
	denoised := make([]float64, n)
	for i, row := range changes {
		if multiday {
			denoised[i] = npSum(row) / float64(len(row)) // np.average
		} else {
			denoised[i] = npMedian(row)
		}
		if math.IsNaN(denoised[i]) {
			denoised[i] = 1
		}
	}

	threshold := minChange
	if hasChange {
		// Exit if every change is closer to 1 than to the split.
		lim := (splitMax-1)*0.5 + 1
		lo, hi := denoised[0], denoised[0]
		for _, v := range denoised {
			lo, hi = math.Min(lo, v), math.Max(hi, v)
		}
		if hi < lim && lo > 1/lim {
			return nil
		}
		// True price volatility, ignoring changes outside the IQR.
		q1, q3 := npPercentile(denoised, 25), npPercentile(denoised, 75)
		iqr := q3 - q1
		lb, ub := q1-1.5*iqr, q3+1.5*iqr
		inliers := make([]float64, 0, n)
		for _, v := range denoised {
			if v >= lb && v <= ub {
				inliers = append(inliers, v)
			}
		}
		sdPct := npStd(inliers, 0) / npMean(inliers)
		largest := scaleLargestChange(5*sdPct, interval)
		if pyMax(split, splitRcp) < 1+largest {
			return nil // split ratio too close to normal volatility
		}
		threshold = 1 + (splitMax-1+largest)*0.6
	}

	var x [][]float64 // [column][row]
	if perColumn {
		// Raw per-column changes, zeros replaced by 1, not dividend-adjusted.
		x = make([][]float64, 4)
		for j := range x {
			x[j] = make([]float64, n)
			x[j][0] = 1
			for i := 1; i < n; i++ {
				x[j][i] = nonZero(ohlcValue(rows[i], j)) / nonZero(ohlcValue(rows[i-1], j))
			}
		}
	} else {
		x = [][]float64{denoised}
	}
	thresholdRcp := 1 / threshold
	up, down := make([][]bool, len(x)), make([][]bool, len(x))
	for c := range x {
		up[c], down[c] = make([]bool, n), make([]bool, n)
		for i, v := range x[c] {
			down[c][i] = v < thresholdRcp
			up[c][i] = v > threshold
		}
	}
	if !anySignal(up, down) {
		return nil
	}
	det := &priceShiftDetection{up: up, down: down, threshold: threshold, change1D: make([]float64, n)}
	for i, v := range denoised {
		det.change1D[i] = npRound3(v)
	}
	if !perColumn {
		det.flag = signalFlags(up[0], down[0])
	}
	return det
}

func nonZero(v float64) float64 {
	if v == 0 {
		return 1
	}
	return v
}

func signalFlags(up, down []bool) []bool {
	f := make([]bool, len(up))
	for i := range f {
		f[i] = up[i] || down[i]
	}
	return f
}

// filterPriceShiftsOnVolumeSpikes ports _filter_price_shifts_on_volume_spikes.
// A price drop on abnormally high volume may be a genuine market event, so
// its up signal is cleared. For each up signal at row idx (rows newest
// first), the drop happened at row i = idx-1; i's volume and the volume of
// the first row after it are z-scored against the non-up rows dated after i
// (and before the next down signal), within a 31-row window of non-up rows
// around i. It returns nil when volume is all zero or no signal survives.
func filterPriceShiftsOnVolumeSpikes(rows []models.Bar, det *priceShiftDetection) *priceShiftDetection {
	n := len(rows)
	vol := make([]float64, n)
	allZero := true
	for i, b := range rows {
		vol[i] = float64(b.Volume)
		allZero = allZero && vol[i] == 0
	}
	if allZero {
		return nil
	}
	up, down := cloneMask(det.up), cloneMask(det.down)

	upShift := make([]bool, n)
	var flat []int
	for i := range rows {
		upShift[i] = rowSignal(det.up, i)
		if !upShift[i] {
			flat = append(flat, i)
		}
	}
	var downDates []time.Time
	for i := range rows {
		if rowSignal(det.down, i) {
			downDates = append(downDates, rows[i].Date)
		}
	}
	const day = 24 * time.Hour
	for idx := range rows {
		if !upShift[idx] {
			continue
		}
		i := idx - 1
		dt := rows[i].Date
		pos := 0
		for k := 0; k < i; k++ {
			if !upShift[k] {
				pos++
			}
		}
		start := maxInt(0, pos-15)
		end := minInt(len(flat), start+31)
		block := flat[start:end]

		var next time.Time
		hasNext := false
		for _, d := range downDates {
			if !d.Before(dt) && (!hasNext || d.Before(next)) {
				next, hasNext = d, true
			}
		}
		if hasNext && next.Equal(dt) {
			// Only this row has a price drop: definitely a data error.
			continue
		}
		// block.loc[dt + 1 day : next_down - 1 day], sorted by date.
		lo := dt.Add(day)
		var after []int
		for _, k := range block {
			d := rows[k].Date
			if d.Before(lo) || (hasNext && d.After(next.Add(-day))) {
				continue
			}
			after = append(after, k)
		}
		if len(after) == 0 {
			continue
		}
		sort.SliceStable(after, func(a, b int) bool { return rows[after[a]].Date.Before(rows[after[b]].Date) })
		sample := make([]float64, len(after))
		for k, r := range after {
			sample[k] = vol[r]
		}
		if pyMax(volumeZScore(vol[i], sample), volumeZScore(sample[0], sample)) > 2 {
			for c := range up {
				up[c][idx] = false
			}
		}
	}
	if !anySignal(up, down) {
		return nil
	}
	out := &priceShiftDetection{up: up, down: down, change1D: det.change1D, threshold: det.threshold}
	if !out.perColumn() {
		out.flag = signalFlags(up[0], down[0])
	}
	return out
}

// volumeZScore is upstream's _calc_volume_zscore (sample standard deviation).
func volumeZScore(v float64, sample []float64) float64 {
	allZero := true
	for _, s := range sample {
		allZero = allZero && s == 0
	}
	if len(sample) <= 1 || allZero {
		return 0
	}
	sd := npStd(sample, 1)
	if sd == 0 {
		return 0
	}
	return (v - npMean(sample)) / sd
}

// filterPriceShiftsThatMatchLocalStdev ports
// _filter_price_shifts_that_match_local_stdev: a signal whose change is
// within a threshold derived from the local (non-signal) volatility is a
// false positive. Signals are visited in row order and a cleared signal
// stops being excluded from later windows. As upstream, per-column
// detections pass through unchanged (the column branch only reads values).
// It returns nil when no signal survives.
func filterPriceShiftsThatMatchLocalStdev(n int, interval string, change float64, det *priceShiftDetection) *priceShiftDetection {
	if det.perColumn() {
		return det
	}
	splitMax := pyMax(change, 1/change)
	up, down := cloneMask(det.up), cloneMask(det.down)
	flag := append([]bool(nil), det.flag...)
	lookback := 3
	if strings.HasSuffix(interval, "d") {
		lookback = 10
	} else if strings.HasSuffix(interval, "m") {
		lookback = 100
	}
	signals := signalFlags(up[0], down[0])
	for idx, s := range signals {
		if !s {
			continue
		}
		idxEnd := minInt(n-1, idx+2)
		idxStart := maxInt(0, idx-lookback)
		var clean []float64
		for k := idxStart; k < idxEnd; k++ {
			if !flag[k] {
				clean = append(clean, det.change1D[k])
			}
		}
		sdPct := npStd(clean, 0) / npMean(clean)
		largest := scaleLargestChange(5*sdPct, interval)
		threshold := 1 + (splitMax-1+largest)*0.5
		big := det.change1D[idx]
		if big < threshold && big > 1/threshold {
			flag[idx] = false
		}
	}
	for i := range flag {
		up[0][i] = up[0][i] && flag[i]
		down[0][i] = down[0][i] && flag[i]
	}
	if !anySignal(up, down) {
		return nil
	}
	return &priceShiftDetection{up: up, down: down, change1D: det.change1D, flag: flag, threshold: det.threshold}
}
