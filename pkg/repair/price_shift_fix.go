package repair

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

// priceShiftFix holds the flags of upstream _fix_prices_sudden_change.
type priceShiftFix struct {
	unitSwitch      bool
	correctVolume   bool
	correctDividend bool
}

// priceShiftTrace records the stages of one fixPricesSuddenChange call. It
// exists for the upstream differential tests (Repairer.traceShift); a nil
// mask means upstream returned None at that stage.
type priceShiftTrace struct {
	change         float64
	rows           int
	perColumn      bool
	detectCalled   bool
	threshold      float64
	detected       *priceShiftDetection
	volumeCalled   bool
	volThreshold   float64
	volThresholdOK bool
	afterVolume    *priceShiftDetection
	afterLocal     *priceShiftDetection
	reconciled     bool
	rangesIn       []priceShiftRange
	rangesOut      []priceShiftRange
}

// fixPricesSuddenChange ports Python yfinance 1.8.0 _fix_prices_sudden_change,
// the repair shared by split repair (change = split ratio), unit-switch
// repair (change = 100 or 1000) and unexplained-level-shift repair. bars are
// sorted oldest first and are not modified; the repaired copy is returned.
//
// As upstream: rows with any missing OHLC are set aside; the rest are
// processed newest first. Candidates come from detectPriceShifts, are
// screened for volume spikes and local volatility, then paired into ranges.
// When the newest row has no volume (a suspended stock), signals before and
// after the latest active row are mapped separately. Unless repairing a unit
// switch, ranges whose newest row is older than one year before the oldest
// split are pruned (start_min). Unit-switch repair corrects columns
// individually when a row's high/low spread exceeds half the change, and
// needs at least two columns to agree; otherwise ranges must pass the volume
// reconcile. A 100x/0.01x repair is abandoned when a signal is within 30
// days after a split.
//
// Go adaptations: Volume is an integer, so rescaled volume is rounded half
// to even exactly as upstream's final round().astype(int); bars are
// repaired in place rather than through a reindexed DataFrame. Upstream's
// per-column Volume heuristic is unreachable (unit-switch repair never
// corrects volume) and is not ported.
func (r *Repairer) fixPricesSuddenChange(bars []models.Bar, change float64, opt priceShiftFix) []models.Bar {
	if len(bars) == 0 {
		return bars
	}
	tr := &priceShiftTrace{change: change}
	if r.traceShift != nil {
		defer r.traceShift(tr)
	}
	interval := r.opts.Interval
	loc := r.location()
	split, splitRcp := change, 1/change
	splitMax := pyMax(split, splitRcp)

	var startMin time.Time
	hasStartMin := false
	if !opt.unitSwitch {
		// start_min = 1 year before the oldest split (NaN counts as a split,
		// as upstream's != 0 test does).
		for _, b := range bars {
			if b.Splits != 0 && (!hasStartMin || b.Date.Before(startMin)) {
				startMin, hasStartMin = b.Date, true
			}
		}
		if hasStartMin {
			startMin = subtractYearClamped(startMin.In(loc))
		}
	}

	perColumn := false
	if opt.unitSwitch {
		for _, b := range bars {
			if rowSpreadExceeds(b, 0.5*splitMax) {
				perColumn = true
				break
			}
		}
	}

	if 0.8 < split && split < 1.25 {
		return bars
	}

	// Newest-first rows without missing OHLC; idx maps them back to bars.
	idx := make([]int, 0, len(bars))
	for i := len(bars) - 1; i >= 0; i-- {
		if ohlcComplete(bars[i]) {
			idx = append(idx, i)
		}
	}
	n := len(idx)
	work := make([]models.Bar, n)
	for k, i := range idx {
		work[k] = bars[i]
	}

	// A suspended stock: no volume on the newest row. Use the latest active
	// row (preferably with an active row before it) as the baseline.
	active := make([]bool, n)
	for k := range work {
		active[k] = work[k].Volume != 0
	}
	appearsSuspended := n > 0 && !active[0]
	latestActive := -1
	for k := range active {
		if active[k] && active[(k-1+n)%n] {
			latestActive = k
			break
		}
	}
	if latestActive < 0 {
		for k := range active {
			if active[k] {
				latestActive = k
				break
			}
		}
	}

	tr.detectCalled, tr.rows, tr.perColumn = true, n, perColumn
	det := detectPriceShifts(work, interval, change, perColumn, 2)
	tr.detected = det
	if det != nil {
		tr.threshold = det.threshold
	}
	if det == nil {
		return bars
	}
	vol := make([]float64, n)
	for k := range work {
		vol[k] = float64(work[k].Volume)
	}
	tr.volumeCalled = true
	volThreshold, volThresholdOK := math.NaN(), false
	if !det.perColumn() {
		volThreshold, volThresholdOK = estimateVolumeShiftThreshold(vol, det.up[0], det.down[0], change, interval)
	}
	tr.volThreshold, tr.volThresholdOK = volThreshold, volThresholdOK
	det = filterPriceShiftsOnVolumeSpikes(work, det)
	tr.afterVolume = det
	if det == nil {
		return bars
	}
	det = filterPriceShiftsThatMatchLocalStdev(n, interval, change, det)
	tr.afterLocal = det
	if det == nil {
		return bars
	}
	if (change == 100 || change == 0.01) && r.unitSwitchTooSoonAfterSplit(work, det, interval) {
		return bars
	}

	result := append([]models.Bar(nil), bars...)
	repaired := make([]bool, n)
	anyMBelow1 := false
	firstSignal := -1
	for k := 0; k < n && firstSignal < 0; k++ {
		if rowSignal(det.up, k) || rowSignal(det.down, k) {
			firstSignal = k
		}
	}
	suspendedSplit := appearsSuspended && latestActive >= 0 && latestActive >= firstSignal
	ranges := func(up, down []bool) []priceShiftRange {
		var out []priceShiftRange
		if suspendedSplit {
			out = mapSuspendedRanges(up, down, change, latestActive)
		} else {
			out = mapSignalsToRanges(up, down, change)
		}
		if !hasStartMin {
			return out
		}
		kept := out[:0]
		for _, rg := range out {
			if !localDate(work[rg.start].Date.In(loc)).Before(startMin) {
				kept = append(kept, rg)
			}
		}
		return kept
	}

	if det.perColumn() {
		var colRanges [4][]priceShiftRange
		count := 0
		for j := 0; j < 4; j++ {
			colRanges[j] = ranges(det.up[j], det.down[j])
			if len(colRanges[j]) > 0 {
				count++
			}
		}
		// A shift in a single column is treated as a false positive.
		if count >= 2 {
			for j := 0; j < 4; j++ {
				for _, rg := range colRanges[j] {
					anyMBelow1 = anyMBelow1 || rg.multiplier < 0.99
					for k := rg.start; k < rg.end; k++ {
						scaleColumn(&work[k], j, rg.multiplier)
						if j == 3 {
							work[k].AdjClose *= rg.multiplier
						}
						repaired[k] = true
					}
				}
			}
			// Yahoo derived Low/High from the bad prices, so recalculate.
			for j := 0; j < 4; j++ {
				for _, rg := range colRanges[j] {
					for k := rg.start; k < rg.end; k++ {
						if rg.multiplier > 1 {
							work[k].Low = math.Min(work[k].Open, work[k].Close)
						} else {
							work[k].High = math.Max(work[k].Open, work[k].Close)
						}
					}
				}
			}
		}
	} else {
		in := ranges(det.up[0], det.down[0])
		tr.reconciled = true
		tr.rangesIn = append([]priceShiftRange(nil), in...)
		out := reconcilePriceShiftsWithVolume(vol, in, det.up[0], det.down[0], volThreshold, opt.unitSwitch)
		tr.rangesOut = out
		for _, rg := range out {
			anyMBelow1 = anyMBelow1 || rg.multiplier < 0.99
			for k := rg.start; k < rg.end; k++ {
				scaleBar(&work[k], rg.multiplier, rg.volumeMultiplier, opt)
				repaired[k] = true
			}
		}
	}

	if opt.unitSwitch && anyMBelow1 && r.currencyRepaired {
		// The switch was repaired towards the major currency, which
		// standardiseCurrency had already done: revert this repair.
		for k := range work {
			scaleBar(&work[k], change, 1/change, opt)
			repaired[k] = !repaired[k]
		}
	}
	for k, i := range idx {
		work[k].Repaired = work[k].Repaired || repaired[k]
		result[i] = work[k]
	}
	return result
}

// rowSpreadExceeds reports |max(OHLC)/min(OHLC) - 1| > limit, skipping NaN
// prices as pandas does.
func rowSpreadExceeds(b models.Bar, limit float64) bool {
	hi, lo := math.NaN(), math.NaN()
	for j := 0; j < 4; j++ {
		v := ohlcValue(b, j)
		if math.IsNaN(v) {
			continue
		}
		if math.IsNaN(hi) || v > hi {
			hi = v
		}
		if math.IsNaN(lo) || v < lo {
			lo = v
		}
	}
	return math.Abs(hi/lo-1) > limit
}

func scaleColumn(b *models.Bar, j int, m float64) {
	switch j {
	case 0:
		b.Open *= m
	case 1:
		b.High *= m
	case 2:
		b.Low *= m
	default:
		b.Close *= m
	}
}

// scaleBar multiplies prices (and dividends when correcting them) by m, and
// volume by mRcp with NumPy's half-to-even rounding when correcting volume.
func scaleBar(b *models.Bar, m, mRcp float64, opt priceShiftFix) {
	b.Open *= m
	b.High *= m
	b.Low *= m
	b.Close *= m
	b.AdjClose *= m
	if opt.correctDividend {
		b.Dividends *= m
	}
	if opt.correctVolume {
		b.Volume = int64(math.RoundToEven(float64(b.Volume) * mRcp))
	}
}

// mapSuspendedRanges ports upstream's suspended-midway branch: rows from
// latestActive onwards are mapped newest first as usual; rows before it are
// mapped in reverse (oldest-first) order with reversed signals, and the
// resulting ranges flipped back to newest-first positions.
func mapSuspendedRanges(up, down []bool, change float64, latestActive int) []priceShiftRange {
	n := len(up)
	before := mapSignalsToRanges(up[latestActive:], down[latestActive:], change)
	for i := range before {
		before[i].start += latestActive
		before[i].end += latestActive
	}
	// f_rev_down = flip(roll(f_up, -1)), f_rev_up = flip(roll(f_down, -1)).
	revUp, revDown := make([]bool, n), make([]bool, n)
	for i := 0; i < n; i++ {
		revDown[i] = up[(n-i)%n]
		revUp[i] = down[(n-i)%n]
	}
	revLatest := n - 1 - latestActive
	after := mapSignalsToRanges(revUp[revLatest:], revDown[revLatest:], change)
	for i := range after {
		s, e := after[i].start+revLatest, after[i].end+revLatest
		after[i].start, after[i].end = n-e, n-s
	}
	return append(before, after...)
}

// unitSwitchTooSoonAfterSplit ports upstream's abort of a 100x repair when a
// signal is within 30 days after a split, as it could be a split error.
// Signal rows are listed once per signalling column, as np.where(f)[0] does.
func (r *Repairer) unitSwitchTooSoonAfterSplit(work []models.Bar, det *priceShiftDetection, interval string) bool {
	var splits, signals []int
	for k := range work {
		if work[k].Splits != 0 {
			splits = append(splits, k)
		}
		for c := range det.up {
			if det.up[c][k] || det.down[c][k] {
				signals = append(signals, k)
			}
		}
	}
	if len(splits) == 0 || len(signals) == 0 {
		return false
	}
	gapMin, gapPos := 0, -1
	for b, s := range signals {
		for _, a := range splits {
			// Rows are newest first, so a positive gap is a signal after a split.
			if g := a - s; g > 0 && (gapPos < 0 || g < gapMin) {
				gapMin, gapPos = g, b
			}
		}
	}
	if gapPos < 0 {
		return false
	}
	// Upstream indexes the rows with the position in the signal list.
	dt := work[minInt(gapPos, len(work)-1)].Date.In(r.location())
	return intervalGapWithin30Days(interval, gapMin, dt)
}

// intervalGapWithin30Days ports `_interval_to_timedelta(interval) * gap <
// 30 days`. Day and week multiples compare as absolute days; month and year
// multiples are calendar offsets (dateutil relativedelta, clamping the day)
// compared at dt with dt + 30*24h; minutes and hours are absolute.
func intervalGapWithin30Days(interval string, gap int, dt time.Time) bool {
	num := func(s string) int {
		v, err := strconv.Atoi(s)
		if err != nil {
			return 1
		}
		return v
	}
	const thirty = 30 * 24 * time.Hour
	switch {
	case strings.HasSuffix(interval, "d"):
		return num(strings.TrimSuffix(interval, "d"))*gap < 30
	case strings.HasSuffix(interval, "wk"):
		return 7*num(strings.TrimSuffix(interval, "wk"))*gap < 30
	case strings.HasSuffix(interval, "mo"):
		return addMonthsClamped(dt, num(strings.TrimSuffix(interval, "mo"))*gap).Before(dt.Add(thirty))
	case strings.HasSuffix(interval, "y"):
		return addMonthsClamped(dt, 12*num(strings.TrimSuffix(interval, "y"))*gap).Before(dt.Add(thirty))
	case strings.HasSuffix(interval, "m"):
		return time.Duration(num(strings.TrimSuffix(interval, "m"))*gap)*time.Minute < thirty
	case strings.HasSuffix(interval, "h"):
		return time.Duration(num(strings.TrimSuffix(interval, "h"))*gap)*time.Hour < thirty
	}
	return false
}

// addMonthsClamped adds months on the wall clock, clamping the day to the
// end of the target month (dateutil relativedelta).
func addMonthsClamped(t time.Time, months int) time.Time {
	y, m, d := t.Date()
	total := int(m) - 1 + months
	ty, tm := y+total/12, time.Month(total%12+1)
	if last := time.Date(ty, tm+1, 0, 0, 0, 0, 0, time.UTC).Day(); d > last {
		d = last
	}
	return time.Date(ty, tm, d, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}
