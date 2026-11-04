package repair

import (
	"math"
	"sort"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/client"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

// HourlyFetcher fetches raw hourly bars between start (inclusive) and end
// (exclusive) for the ticker being repaired.
//
// The unexplained-level-shift repair calls it at most once per
// [Repairer.Repair], with start and end at midnight in the exchange timezone.
// Implementations must return unadjusted bars (no auto-adjust), without
// pre/post-market rows, and must not run price repair themselves. Returning
// an error for which [client.IsRateLimitError] is true makes Repair fail with
// that error; any other error makes the repair skip the level-shift step.
type HourlyFetcher func(start, end time.Time) ([]models.Bar, error)

// Level-shift repair constants, from upstream _fix_unexplained_level_shifts.
const (
	// levelShiftMinChange ignores day-to-day price changes smaller than 2x.
	levelShiftMinChange = 2.0
	// levelShiftSideRows is the number of rows analysed on each side of a
	// shift, and the minimum distance to a recorded split.
	levelShiftSideRows = 5
	// levelShiftMinSide is how many rows a shift must persist for, and the
	// minimum distance from either end of the data.
	levelShiftMinSide = 3
	// levelShiftHourlyDays is how far back Yahoo serves 1h data (730 days,
	// minus one for safety).
	levelShiftHourlyDays = 729
	// levelShiftRatioTolerance is the allowed dispersion of 1d/1h ratios
	// around their side median (5%).
	levelShiftRatioTolerance = 0.05
	// levelShiftCleanTolerance is how close the measured multiple must be to
	// an integer (or integer reciprocal) multiple (3%).
	levelShiftCleanTolerance = 0.03
)

// levelShiftCandidate is a possible shift at active row idx, with analysis
// window [i0, i1) of active rows.
type levelShiftCandidate struct {
	idx, i0, i1 int
}

// hourlyDay is 1h data aggregated to one exchange-timezone day.
type hourlyDay struct {
	open, high, low, close float64
}

// repairLevelShifts ports upstream _fix_unexplained_level_shifts (Python
// yfinance 1.8.0, #2990: commits 905d82f, a3241e6, 7c66783).
//
// Yahoo sometimes applies a split-like adjustment to all daily data before a
// date although no split exists near it: prices are N times too large and
// volume N times too small, and N need not be a split ratio. Daily data
// cannot measure N because the genuine price move across the gap is unknown,
// but Yahoo's intraday data is not affected, so N is measured as the change
// in the 1d/1h price ratio across the shift. The shift is then repaired with
// the split-repair logic, which also requires volume to confirm it. A genuine
// crash appears in 1h data too, so it is left alone.
//
// bars must be sorted oldest first. The hourly fetch happens only when a
// candidate shift survives the persistence check, and at most once.
func (r *Repairer) repairLevelShifts(bars []models.Bar) ([]models.Bar, error) {
	if len(bars) == 0 || r.opts.Interval != "1d" || r.opts.FetchHourly == nil {
		return bars, nil
	}

	active := levelShiftActiveRows(bars)
	if len(active) < 2*levelShiftMinSide {
		return bars, nil
	}
	prices := levelShiftPrices(bars, active)
	activeBars := make([]models.Bar, len(active))
	for k, i := range active {
		activeBars[k] = bars[i]
	}
	det := detectPriceShifts(activeBars, r.opts.Interval, 0, false, levelShiftMinChange)
	if det == nil {
		return bars, nil
	}
	up, down := det.up[0], det.down[0]

	now := r.now()
	minDate := now.Add(-levelShiftHourlyDays * 24 * time.Hour)
	candidates := levelShiftCandidates(bars, active, up, down, minDate)
	anyShift := false
	for _, c := range candidates {
		if levelShiftPersists(prices, c) {
			anyShift = true
			break
		}
	}
	if !anyShift {
		return bars, nil
	}

	// Fetch the usable two-year 1h window once for every candidate.
	loc := r.location()
	start := localMidnight(minDate, loc)
	end := localMidnight(now, loc).AddDate(0, 0, 1)
	hourly, err := r.opts.FetchHourly(start, end)
	if err != nil {
		if client.IsRateLimitError(err) {
			return nil, err
		}
		return bars, nil
	}
	days := aggregateHourlyToDays(hourly, loc)
	if len(days) == 0 {
		return bars, nil
	}

	result := bars
	for _, c := range candidates {
		// Re-evaluate persistence after previous repairs updated prices.
		if !levelShiftPersists(prices, c) {
			continue
		}
		dtShift := result[active[c.idx]].Date
		mult, ok := levelShiftMultiple(result, active, prices, days, c, loc)
		if !ok {
			continue
		}

		// Repair with the same logic as a bad split, which also confirms the
		// change with volume. Dividends are not touched: the dividend repair
		// that runs next corrects the dividend adjustment.
		nBefore := countRepairedBefore(result, dtShift)
		repaired := r.fixPricesSuddenChange(result, mult, priceShiftFix{correctVolume: true})
		if countRepairedBefore(repaired, dtShift) <= nBefore {
			continue
		}
		result = repaired
		prices = levelShiftPrices(result, active)
	}
	return result, nil
}

// now returns the configured clock's time, or time.Now.
func (r *Repairer) now() time.Time {
	if r.opts.Now != nil {
		return r.opts.Now()
	}
	return time.Now()
}

// location returns the exchange timezone, falling back to UTC when it is
// unset or unknown.
func (r *Repairer) location() *time.Location {
	if r.opts.Timezone == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(r.opts.Timezone)
	if err != nil {
		return time.UTC
	}
	return loc
}

// levelShiftActiveRows returns the indices of bars with all OHLC > 0 and
// volume > 0 (NaN prices are inactive).
func levelShiftActiveRows(bars []models.Bar) []int {
	active := make([]int, 0, len(bars))
	for i, b := range bars {
		if b.Open > 0 && b.High > 0 && b.Low > 0 && b.Close > 0 && b.Volume > 0 {
			active = append(active, i)
		}
	}
	return active
}

// levelShiftPrices copies the OHLC of the active rows.
func levelShiftPrices(bars []models.Bar, active []int) [][4]float64 {
	prices := make([][4]float64, len(active))
	for k, i := range active {
		b := bars[i]
		prices[k] = [4]float64{b.Open, b.High, b.Low, b.Close}
	}
	return prices
}

// levelShiftCandidates filters the detected shifts, newest first: not within
// levelShiftMinSide rows of either end, not within levelShiftSideRows rows of
// a recorded split (split repair handles those), and with the whole
// before-window not older than minDate (no 1h data there).
func levelShiftCandidates(bars []models.Bar, active []int, up, down []bool, minDate time.Time) []levelShiftCandidate {
	n := len(active)
	// Split rows can be inactive, so locate them among active rows by date
	// (pandas searchsorted, side='left').
	var splitPos []int
	for _, b := range bars {
		if b.Splits != 0 {
			splitPos = append(splitPos, sort.Search(n, func(k int) bool {
				return !bars[active[k]].Date.Before(b.Date)
			}))
		}
	}
	var out []levelShiftCandidate
	for idx := n - 1; idx >= 0; idx-- {
		if !up[idx] && !down[idx] {
			continue
		}
		if idx < levelShiftMinSide || n-idx < levelShiftMinSide {
			continue
		}
		nearSplit := false
		for _, p := range splitPos {
			if absInt(p-idx) <= levelShiftSideRows {
				nearSplit = true
				break
			}
		}
		if nearSplit {
			continue
		}
		i0 := maxInt(0, idx-levelShiftSideRows)
		if bars[active[i0]].Date.Before(minDate) {
			continue
		}
		out = append(out, levelShiftCandidate{idx: idx, i0: i0, i1: minInt(n, idx+levelShiftSideRows)})
	}
	return out
}

// levelShiftPersists reports whether the median Close after the shift
// differs from the median Close before it by at least levelShiftMinChange.
func levelShiftPersists(prices [][4]float64, c levelShiftCandidate) bool {
	closes := func(lo, hi int) []float64 {
		out := make([]float64, 0, hi-lo)
		for k := lo; k < hi; k++ {
			out = append(out, prices[k][3])
		}
		return out
	}
	shift := npMedian(closes(c.idx, c.i1)) / npMedian(closes(c.i0, c.idx))
	return !(1/levelShiftMinChange < shift && shift < levelShiftMinChange)
}

// aggregateHourlyToDays groups hourly bars with all OHLC > 0 by their date
// in loc: Open first, High max, Low min, Close last.
func aggregateHourlyToDays(hourly []models.Bar, loc *time.Location) map[string]hourlyDay {
	sorted := append([]models.Bar(nil), hourly...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Date.Before(sorted[j].Date) })
	days := make(map[string]hourlyDay)
	for _, b := range sorted {
		if !(b.Open > 0 && b.High > 0 && b.Low > 0 && b.Close > 0) {
			continue
		}
		key := b.Date.In(loc).Format(time.DateOnly)
		d, ok := days[key]
		if !ok {
			days[key] = hourlyDay{open: b.Open, high: b.High, low: b.Low, close: b.Close}
			continue
		}
		d.high = math.Max(d.high, b.High)
		d.low = math.Min(d.low, b.Low)
		d.close = b.Close
		days[key] = d
	}
	return days
}

// levelShiftMultiple measures the multiple that undoes candidate c: for each
// active row in its window, the median of 1d/1h OHLC ratios. It needs at
// least two ratios on each side of the shift, every ratio within 5% of its
// side median, a multiple outside (0.8, 1.25) (inside means the move is
// genuine, as 1h data shows it too), and a multiple within 3% of an integer
// or integer reciprocal, which it returns.
func levelShiftMultiple(bars []models.Bar, active []int, prices [][4]float64, days map[string]hourlyDay, c levelShiftCandidate, loc *time.Location) (float64, bool) {
	var before, after []float64
	for k := c.i0; k < c.i1; k++ {
		d, ok := days[bars[active[k]].Date.In(loc).Format(time.DateOnly)]
		if !ok {
			continue
		}
		hourly := [4]float64{d.open, d.high, d.low, d.close}
		var ratios [4]float64
		for j := range ratios {
			ratios[j] = prices[k][j] / hourly[j]
		}
		ratio := npMedian(ratios[:])
		if k < c.idx {
			before = append(before, ratio)
		} else {
			after = append(after, ratio)
		}
	}
	return levelShiftMultipleFromRatios(before, after)
}

// levelShiftMultipleFromRatios applies the ratio-consistency, genuine-move
// and clean-multiple checks of levelShiftMultiple to the per-row 1d/1h
// ratios before and after the shift.
func levelShiftMultipleFromRatios(before, after []float64) (float64, bool) {
	if len(before) < 2 || len(after) < 2 {
		return 0, false
	}
	rBefore, rAfter := npMedian(before), npMedian(after)
	tol := math.Log(1 + levelShiftRatioTolerance)
	for _, side := range []struct {
		ratios []float64
		median float64
	}{{before, rBefore}, {after, rAfter}} {
		for _, v := range side.ratios {
			if math.Abs(math.Log(v/side.median)) > tol {
				return 0, false
			}
		}
	}
	m := rBefore / rAfter
	if 0.8 < m && m < 1.25 {
		return 0, false
	}
	mInt := cleanLevelShiftMultiple(m)
	if math.Abs(m/mInt-1) > levelShiftCleanTolerance {
		return 0, false
	}
	return mInt, true
}

// cleanLevelShiftMultiple rounds m to the nearest integer, or for m < 1 to
// the nearest integer reciprocal, with Python's half-to-even round.
func cleanLevelShiftMultiple(m float64) float64 {
	if m > 1 {
		return math.RoundToEven(m)
	}
	return 1 / math.RoundToEven(1/m)
}

// countRepairedBefore counts repaired bars dated before t.
func countRepairedBefore(bars []models.Bar, t time.Time) int {
	n := 0
	for _, b := range bars {
		if b.Repaired && b.Date.Before(t) {
			n++
		}
	}
	return n
}

// subtractYearClamped is dateutil's relativedelta(years=1) subtraction,
// returning a date: 29 February maps to 28 February.
func subtractYearClamped(t time.Time) time.Time {
	y, m, d := t.Date()
	if m == time.February && d == 29 {
		d = 28
	}
	return time.Date(y-1, m, d, 0, 0, 0, 0, time.UTC)
}

// localDate truncates t to its calendar date (as a UTC midnight, for
// comparisons only).
func localDate(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// localMidnight returns midnight in loc of t's date in loc.
func localMidnight(t time.Time, loc *time.Location) time.Time {
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
