package repair

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/client"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

func soxsFixtures(t *testing.T) (bad, fixed []models.Bar, hourly *fakeHourly) {
	t.Helper()
	bad = loadBarsCSV(t, "SOXS-1d-bad-level-shift.csv")
	fixed = loadBarsCSV(t, "SOXS-1d-bad-level-shift-fixed.csv")
	hourly = &fakeHourly{bars: loadBarsCSV(t, "SOXS-1h-bad-level-shift.csv")}
	return bad, fixed, hourly
}

func assertUnchanged(t *testing.T, got, want []models.Bar) {
	t.Helper()
	assertLevelShiftGolden(t, "unchanged", got, want, 0)
	for i := range got {
		if got[i].Repaired != want[i].Repaired {
			t.Fatalf("row %d Repaired=%v, want %v", i, got[i].Repaired, want[i].Repaired)
		}
	}
}

// TestRepairLevelShiftSOXSFullPipeline runs every repair, as History does.
// The level-shift repair runs first, so the later repairs see the fixed
// prices and leave them alone.
func TestRepairLevelShiftSOXSFullPipeline(t *testing.T) {
	bad, fixed, hourly := soxsFixtures(t)
	opts := DefaultOptions()
	opts.Ticker = "SOXS"
	opts.Timezone = "America/New_York"
	opts.Currency = "USD"
	opts.QuoteType = QuoteTypeETF
	opts.FetchHourly = hourly.fetch
	now := fixtureNow(bad)
	opts.Now = func() time.Time { return now }

	got, err := New(opts).Repair(bad)
	if err != nil {
		t.Fatalf("Repair: %v", err)
	}
	if len(hourly.calls) != 1 {
		t.Fatalf("hourly fetches = %d, want 1", len(hourly.calls))
	}
	assertLevelShiftGolden(t, "SOXS", got, fixed, 1e-6)
}

func TestDefaultOptionsEnableLevelShifts(t *testing.T) {
	opts := DefaultOptions()
	if !opts.FixLevelShifts {
		t.Fatal("DefaultOptions should enable FixLevelShifts")
	}
	if opts.FetchHourly != nil || opts.Now != nil {
		t.Fatal("DefaultOptions must not set a fetcher or clock")
	}
}

// TestRepairLevelShiftSkips covers every case in which the hourly data must
// not be fetched and the bars must not change.
func TestRepairLevelShiftSkips(t *testing.T) {
	bad, _, _ := soxsFixtures(t)
	now := fixtureNow(bad)
	tests := []struct {
		name   string
		mutate func(*Options)
		bars   func() []models.Bar
	}{
		{name: "weekly interval", mutate: func(o *Options) { o.Interval = "1wk" }},
		{name: "hourly interval", mutate: func(o *Options) { o.Interval = "1h" }},
		{name: "FX ticker", mutate: func(o *Options) { o.Ticker = "EURUSD=X" }},
		{name: "disabled", mutate: func(o *Options) { o.FixLevelShifts = false }},
		{name: "older than 729 days", mutate: func(o *Options) {
			later := now.AddDate(3, 0, 0)
			o.Now = func() time.Time { return later }
		}},
		{name: "too few active rows", bars: func() []models.Bar {
			short := append([]models.Bar(nil), bad[7:12]...)
			return short
		}},
		{name: "inactive volume hides shift", bars: func() []models.Bar {
			out := append([]models.Bar(nil), bad...)
			for i := range out {
				out[i].Volume = 0
			}
			return out
		}},
		{name: "one-day spike does not persist", bars: func() []models.Bar {
			_, fixed, _ := soxsFixtures(t)
			out := append([]models.Bar(nil), fixed...)
			for i := range out {
				out[i].Repaired = false
			}
			out[9].Open *= 15
			out[9].High *= 15
			out[9].Low *= 15
			out[9].Close *= 15
			out[9].AdjClose *= 15
			return out
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bars := bad
			if tc.bars != nil {
				bars = tc.bars()
			}
			hourly := &fakeHourly{bars: loadBarsCSV(t, "SOXS-1h-bad-level-shift.csv")}
			opts := levelShiftOptions("SOXS", hourly, now)
			if tc.mutate != nil {
				tc.mutate(&opts)
			}
			got, err := New(opts).Repair(bars)
			if err != nil {
				t.Fatalf("Repair: %v", err)
			}
			if len(hourly.calls) != 0 {
				t.Fatalf("hourly fetches = %d, want 0", len(hourly.calls))
			}
			assertUnchanged(t, got, bars)
		})
	}
}

func TestRepairLevelShiftNilFetcher(t *testing.T) {
	bad, _, _ := soxsFixtures(t)
	opts := DefaultOptions()
	opts.Ticker = "SOXS"
	opts.Timezone = "America/New_York"
	opts.FixDividends, opts.FixSplits, opts.FixUnitMixups, opts.FixZeroes = false, false, false, false
	got, err := New(opts).Repair(bad)
	if err != nil {
		t.Fatalf("Repair: %v", err)
	}
	assertUnchanged(t, got, bad)
}

func TestRepairLevelShiftFetchErrors(t *testing.T) {
	bad, _, _ := soxsFixtures(t)
	now := fixtureNow(bad)

	t.Run("generic error skips the step", func(t *testing.T) {
		hourly := &fakeHourly{err: errors.New("boom")}
		got, err := New(levelShiftOptions("SOXS", hourly, now)).Repair(bad)
		if err != nil {
			t.Fatalf("Repair: %v", err)
		}
		if len(hourly.calls) != 1 {
			t.Fatalf("hourly fetches = %d, want 1", len(hourly.calls))
		}
		assertUnchanged(t, got, bad)
	})

	t.Run("no hourly data skips the step", func(t *testing.T) {
		hourly := &fakeHourly{}
		got, err := New(levelShiftOptions("SOXS", hourly, now)).Repair(bad)
		if err != nil {
			t.Fatalf("Repair: %v", err)
		}
		if len(hourly.calls) != 1 {
			t.Fatalf("hourly fetches = %d, want 1", len(hourly.calls))
		}
		assertUnchanged(t, got, bad)
	})

	t.Run("rate limit is returned", func(t *testing.T) {
		hourly := &fakeHourly{err: client.WrapRateLimitError()}
		opts := levelShiftOptions("SOXS", hourly, now)
		opts.FixDividends = true
		r := New(opts)
		got, err := r.Repair(bad)
		if !client.IsRateLimitError(err) {
			t.Fatalf("err = %v, want rate limit", err)
		}
		if got != nil {
			t.Fatalf("bars = %d rows, want nil", len(got))
		}
		if CountRepaired(bad) != 0 || bad[0].Close != 1230.0 {
			t.Fatal("Repair mutated its input")
		}
		if r.opts.Currency != "USD" {
			t.Fatalf("currency not restored: %q", r.opts.Currency)
		}
	})

	t.Run("wrapped rate limit is returned", func(t *testing.T) {
		wrapped := &fakeHourly{err: errors.Join(errors.New("chart"), client.ErrRateLimit)}
		_, err := New(levelShiftOptions("SOXS", wrapped, now)).Repair(bad)
		if !client.IsRateLimitError(err) {
			t.Fatalf("err = %v, want rate limit", err)
		}
	})
}

// TestRepairLevelShiftFetchWindow checks the single 1h request spans the
// last 729 days, from midnight to midnight in the exchange timezone, as
// upstream's history(start=min_dt.date(), end=now.date()+1) does. The dates
// were checked with the upstream oracle.
func TestRepairLevelShiftFetchWindow(t *testing.T) {
	bad, _, _ := soxsFixtures(t)
	ny := nyLocation()
	tests := []struct {
		name       string
		now        time.Time
		start, end time.Time
	}{
		{
			name:  "midday UTC",
			now:   time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC),
			start: time.Date(2024, 6, 13, 0, 0, 0, 0, ny),
			end:   time.Date(2026, 6, 13, 0, 0, 0, 0, ny),
		},
		{
			// 02:00 UTC is still the previous evening in New York.
			name:  "UTC date ahead of exchange date",
			now:   time.Date(2026, 6, 13, 2, 0, 0, 0, time.UTC),
			start: time.Date(2024, 6, 13, 0, 0, 0, 0, ny),
			end:   time.Date(2026, 6, 13, 0, 0, 0, 0, ny),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hourly := &fakeHourly{bars: loadBarsCSV(t, "SOXS-1h-bad-level-shift.csv")}
			if _, err := New(levelShiftOptions("SOXS", hourly, tc.now)).Repair(bad); err != nil {
				t.Fatalf("Repair: %v", err)
			}
			if len(hourly.calls) != 1 {
				t.Fatalf("hourly fetches = %d, want 1", len(hourly.calls))
			}
			c := hourly.calls[0]
			if !c.start.Equal(tc.start) || !c.end.Equal(tc.end) {
				t.Fatalf("window = %v .. %v, want %v .. %v", c.start, c.end, tc.start, tc.end)
			}
			if c.start.Location().String() != "America/New_York" {
				t.Fatalf("start location = %v", c.start.Location())
			}
		})
	}
}

// TestLevelShiftCandidatesTooOld pins the 729-day limit on the oldest row of
// the analysis window (five active rows before the shift).
func TestLevelShiftCandidatesTooOld(t *testing.T) {
	daily, _ := synthShift{badFactor: 15}.build()
	active := levelShiftActiveRows(daily)
	up, down := detectLevelShiftSignals(daily, active, levelShiftMinChange)
	windowStart := daily[18-levelShiftSideRows].Date

	got := levelShiftCandidates(daily, active, up, down, windowStart)
	if len(got) != 1 || got[0] != (levelShiftCandidate{idx: 18, i0: 13, i1: 23}) {
		t.Fatalf("candidates at the limit = %+v", got)
	}
	if got := levelShiftCandidates(daily, active, up, down, windowStart.Add(time.Second)); len(got) != 0 {
		t.Fatalf("candidates past the limit = %+v", got)
	}
}

// TestLevelShiftCandidatesSplitDistance checks the five-row exclusion around
// a split, including a split on an inactive row located by date.
func TestLevelShiftCandidatesSplitDistance(t *testing.T) {
	for _, tc := range []struct {
		splitRow int
		inactive bool
		want     int
	}{
		{splitRow: 12, want: 1}, {splitRow: 13, want: 0}, {splitRow: 23, want: 0}, {splitRow: 24, want: 1},
		{splitRow: 23, inactive: true, want: 0}, {splitRow: 24, inactive: true, want: 1},
	} {
		daily, _ := synthShift{badFactor: 15, splitRow: tc.splitRow}.build()
		if tc.inactive {
			daily[tc.splitRow].Volume = 0
		}
		active := levelShiftActiveRows(daily)
		up, down := detectLevelShiftSignals(daily, active, levelShiftMinChange)
		got := levelShiftCandidates(daily, active, up, down, time.Time{})
		if len(got) != tc.want {
			t.Errorf("split row %d inactive=%v: %d candidates, want %d", tc.splitRow, tc.inactive, len(got), tc.want)
		}
	}
}

func TestDetectPriceShiftsUnknownChange(t *testing.T) {
	mk := func(closes ...float64) []models.Bar {
		bars := make([]models.Bar, len(closes))
		for i, c := range closes {
			bars[i] = models.Bar{Open: c, High: c, Low: c, Close: c, AdjClose: c, Volume: 1}
		}
		return bars
	}
	bars := mk(10, 10, 21, 20, 9.9, 10)
	active := levelShiftActiveRows(bars)
	up, down := detectLevelShiftSignals(bars, active, 2)
	wantUp := []bool{false, false, true, false, false, false}
	wantDown := []bool{false, false, false, false, true, false}
	for i := range up {
		if up[i] != wantUp[i] || down[i] != wantDown[i] {
			t.Fatalf("row %d up=%v down=%v", i, up[i], down[i])
		}
	}
	// Exactly 2x is not a candidate (strict comparison).
	if up, _ := detectLevelShiftSignals(mk(10, 20, 20), []int{0, 1, 2}, 2); up != nil {
		t.Fatal("exact 2x should not be a candidate")
	}
	// Dividend-adjusted prices are compared: a big drop in Close that the
	// adjustment explains is no candidate.
	adj := mk(10, 4, 4)
	adj[0].AdjClose = 4
	if up, _ := detectLevelShiftSignals(adj, []int{0, 1, 2}, 2); up != nil {
		t.Fatal("adjusted change should not be a candidate")
	}
	// The median of the four column changes is used: one wild column is not
	// enough.
	wild := mk(10, 10, 10)
	wild[1].High = 100
	if up, _ := detectLevelShiftSignals(wild, []int{0, 1, 2}, 2); up != nil {
		t.Fatal("single-column spike should not be a candidate")
	}
	// A NaN change counts as no change.
	nan := mk(10, 30, 30)
	nan[1].AdjClose = math.NaN()
	if up, _ := detectLevelShiftSignals(nan, []int{0, 1, 2}, 2); up != nil {
		t.Fatal("NaN change should not be a candidate")
	}
}

func TestLevelShiftPersists(t *testing.T) {
	prices := make([][4]float64, 10)
	set := func(closes ...float64) {
		for i, c := range closes {
			prices[i] = [4]float64{c, c, c, c}
		}
	}
	c := levelShiftCandidate{idx: 5, i0: 0, i1: 10}
	set(30, 30, 30, 30, 30, 10, 10, 10, 10, 10)
	if !levelShiftPersists(prices, c) {
		t.Error("3x drop should persist")
	}
	set(20, 20, 20, 20, 20, 10, 10, 10, 10, 10)
	if !levelShiftPersists(prices, c) {
		t.Error("exactly 2x should count as a shift")
	}
	set(19, 19, 19, 19, 19, 10, 10, 10, 10, 10)
	if levelShiftPersists(prices, c) {
		t.Error("1.9x should not count")
	}
	set(30, 30, 30, 30, 30, 10, 30, 30, 30, 30)
	if levelShiftPersists(prices, c) {
		t.Error("one-day drop should not persist")
	}
}

// TestAggregateHourlyToDays checks grouping by exchange-timezone date: Open
// first, High max, Low min, Close last, rows with a non-positive price
// dropped and input order ignored.
func TestAggregateHourlyToDays(t *testing.T) {
	ny := nyLocation()
	bar := func(t time.Time, o, h, l, c float64) models.Bar {
		return models.Bar{Date: t, Open: o, High: h, Low: l, Close: c}
	}
	hourly := []models.Bar{
		bar(time.Date(2026, 6, 12, 10, 30, 0, 0, ny), 2, 5, 1.5, 3),
		bar(time.Date(2026, 6, 12, 9, 30, 0, 0, ny), 1, 4, 0.5, 2),
		bar(time.Date(2026, 6, 12, 11, 30, 0, 0, ny), 0, 9, 0.1, 9), // dropped
		// 23:30 New York is 03:30 UTC the next day.
		bar(time.Date(2026, 6, 12, 23, 30, 0, 0, ny), 3, 3.5, 2.5, 3.2),
		bar(time.Date(2026, 6, 13, 9, 30, 0, 0, ny), 7, 8, 6, 7.5),
	}
	days := aggregateHourlyToDays(hourly, ny)
	want := map[string]hourlyDay{
		"2026-06-12": {open: 1, high: 5, low: 0.5, close: 3.2},
		"2026-06-13": {open: 7, high: 8, low: 6, close: 7.5},
	}
	if len(days) != len(want) {
		t.Fatalf("days = %+v", days)
	}
	for k, w := range want {
		if days[k] != w {
			t.Errorf("%s = %+v, want %+v", k, days[k], w)
		}
	}

	utc := aggregateHourlyToDays(hourly, time.UTC)
	if d := utc["2026-06-13"]; d.open != 3 || d.close != 7.5 {
		t.Errorf("UTC grouping = %+v", utc)
	}
}

// TestLevelShiftMultipleFromRatios covers the ratio checks: at least two
// ratios per side, 5% dispersion (log scale), the genuine (0.8, 1.25) band,
// half-to-even rounding and the 3% clean-multiple tolerance, including the
// reciprocal branch.
func TestLevelShiftMultipleFromRatios(t *testing.T) {
	tests := []struct {
		name          string
		before, after []float64
		want          float64 // 0 = rejected
	}{
		{"15x", []float64{15, 15.1, 14.9}, []float64{1, 1, 1.01}, 15},
		{"one ratio before", []float64{15}, []float64{1, 1, 1}, 0},
		{"one ratio after", []float64{15, 15}, []float64{1}, 0},
		{"two ratios each side", []float64{15, 15}, []float64{1, 1}, 15},
		{"dispersion 4.9% ok", []float64{15, 15, 15 * 1.049}, []float64{1, 1, 1}, 15},
		{"dispersion 5.1% before", []float64{15, 15, 15 * 1.051}, []float64{1, 1, 1}, 0},
		{"dispersion 5.1% after", []float64{15, 15, 15}, []float64{1, 1, 1 / 1.051}, 0},
		{"genuine 1.24", []float64{1.24, 1.24}, []float64{1, 1}, 0},
		{"genuine 0.81", []float64{0.81, 0.81}, []float64{1, 1}, 0},
		{"1.25 not genuine but unclean", []float64{1.25, 1.25}, []float64{1, 1}, 0},
		{"2x", []float64{2.05, 2.05}, []float64{1, 1}, 2},
		{"2.5 rounds half-even to 2", []float64{2.5, 2.5}, []float64{1, 1}, 0},
		{"3.5 rounds half-even to 4, 12.5% off", []float64{3.5, 3.5}, []float64{1, 1}, 0},
		{"15.44 within 3%", []float64{15.44, 15.44}, []float64{1, 1}, 15},
		{"15.47 beyond 3%", []float64{15.47, 15.47}, []float64{1, 1}, 0},
		{"reciprocal 0.25", []float64{0.25, 0.25}, []float64{1, 1}, 0.25},
		{"reciprocal 1/6.1", []float64{1 / 6.1, 1 / 6.1}, []float64{1, 1}, 1.0 / 6},
		{"reciprocal 1/6.5", []float64{1 / 6.5, 1 / 6.5}, []float64{1, 1}, 0},
		{"ratio of medians", []float64{30, 30, 30}, []float64{2, 2, 2}, 15},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := levelShiftMultipleFromRatios(tc.before, tc.after)
			if tc.want == 0 {
				if ok {
					t.Fatalf("accepted %v, want rejection", got)
				}
				return
			}
			if !ok || !closeTo(got, tc.want, 1e-12) {
				t.Fatalf("got %v ok=%v, want %v", got, ok, tc.want)
			}
		})
	}
}

// TestRepairLevelShiftKALAIsGenuine pins why KALA is left alone: the 1d/1h
// ratio barely moves across its crash.
func TestRepairLevelShiftKALAIsGenuine(t *testing.T) {
	daily := loadBarsCSV(t, "KALA-1d-no-bad-level-shift.csv")
	hourly := loadBarsCSV(t, "KALA-1h-no-bad-level-shift.csv")
	ny := nyLocation()
	active := levelShiftActiveRows(daily)
	up, down := detectLevelShiftSignals(daily, active, levelShiftMinChange)
	candidates := levelShiftCandidates(daily, active, up, down, time.Time{})
	if len(candidates) != 1 {
		t.Fatalf("candidates = %+v, want 1", candidates)
	}
	c := candidates[0]
	if got := daily[active[c.idx]].Date.In(ny).Format(time.DateOnly); got != "2025-09-29" {
		t.Fatalf("candidate date = %s", got)
	}
	if !levelShiftPersists(levelShiftPrices(daily, active), c) {
		t.Fatal("KALA crash should persist")
	}
	days := aggregateHourlyToDays(hourly, ny)
	prices := levelShiftPrices(daily, active)
	var before, after []float64
	for k := c.i0; k < c.i1; k++ {
		d := days[daily[active[k]].Date.In(ny).Format(time.DateOnly)]
		h := [4]float64{d.open, d.high, d.low, d.close}
		r := make([]float64, 4)
		for j := range r {
			r[j] = prices[k][j] / h[j]
		}
		if k < c.idx {
			before = append(before, npMedian(r))
		} else {
			after = append(after, npMedian(r))
		}
	}
	m := npMedian(before) / npMedian(after)
	if !(0.8 < m && m < 1.25) {
		t.Fatalf("m = %v, want genuine", m)
	}
	if _, ok := levelShiftMultiple(daily, active, prices, days, c, ny); ok {
		t.Fatal("levelShiftMultiple accepted a genuine move")
	}
}

func TestSubtractYearClamped(t *testing.T) {
	ny := nyLocation()
	// relativedelta(years=1) clamps 29 February to 28 February.
	if got := subtractYearClamped(time.Date(2024, 2, 29, 0, 0, 0, 0, ny)); !got.Equal(time.Date(2023, 2, 28, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("got %v", got)
	}
	if got := subtractYearClamped(time.Date(2025, 3, 3, 0, 0, 0, 0, ny)); !got.Equal(time.Date(2024, 3, 3, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("got %v", got)
	}
}

func TestCleanLevelShiftMultiple(t *testing.T) {
	for _, tc := range []struct{ m, want float64 }{
		{15.4, 15}, {15.5, 16}, {16.5, 16}, {2.5, 2}, {0.25, 0.25}, {1 / 6.5, 1.0 / 6}, {1 / 7.5, 1.0 / 8},
	} {
		if got := cleanLevelShiftMultiple(tc.m); !closeTo(got, tc.want, 1e-12) {
			t.Errorf("clean(%v) = %v, want %v", tc.m, got, tc.want)
		}
	}
}

// TestFixPricesSuddenChangeWithoutSplit checks the shared repair runs on a
// frame without a split, as the level-shift repair calls it, and leaves
// dividends alone unless asked.
func TestFixPricesSuddenChangeWithoutSplit(t *testing.T) {
	daily, _ := synthShift{badFactor: 15, dividendRow: 10}.build()
	opts := DefaultOptions()
	opts.Timezone = "America/New_York"
	r := New(opts)
	got := r.fixPricesSuddenChange(daily, 15, priceShiftFix{correctVolume: true})
	for i := range got {
		repaired := i < 18
		if got[i].Repaired != repaired {
			t.Fatalf("row %d Repaired=%v", i, got[i].Repaired)
		}
		want := daily[i].Close
		if repaired {
			want /= 15
		}
		if !closeTo(got[i].Close, want, 1e-12) {
			t.Fatalf("row %d close %v, want %v", i, got[i].Close, want)
		}
	}
	if got[10].Dividends != daily[10].Dividends {
		t.Fatal("dividend changed without correctDividend")
	}
	withDiv := r.fixPricesSuddenChange(daily, 15, priceShiftFix{correctVolume: true, correctDividend: true})
	if !closeTo(withDiv[10].Dividends, daily[10].Dividends/15, 1e-12) {
		t.Fatal("dividend not rescaled with correctDividend")
	}
	if CountRepaired(daily) != 0 {
		t.Fatal("fixPricesSuddenChange mutated its input")
	}
}

// detectLevelShiftSignals runs the shared detector as repairLevelShifts does
// (change=None) on the active rows, returning nil masks without candidates.
func detectLevelShiftSignals(bars []models.Bar, active []int, minChange float64) (up, down []bool) {
	rows := make([]models.Bar, len(active))
	for k, i := range active {
		rows[k] = bars[i]
	}
	det := detectPriceShifts(rows, "1d", 0, false, minChange)
	if det == nil {
		return nil, nil
	}
	return det.up[0], det.down[0]
}
