package repair

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

// The differential corpus in testdata/oracle was produced by running the
// upstream Python yfinance 1.8.0 repair functions (see
// testdata/oracle/README.md). These tests replay every case through the Go
// port and require identical results: prices within rtol 1e-9 (NaN equal to
// NaN), and exact Volume and Repaired flags.

const oracleRTol = 1e-9

// nanFloats decodes a JSON array of numbers in which null means NaN.
type nanFloats []float64

func (f *nanFloats) UnmarshalJSON(b []byte) error {
	var raw []*float64
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	out := make([]float64, len(raw))
	for i, v := range raw {
		if v == nil {
			out[i] = math.NaN()
		} else {
			out[i] = *v
		}
	}
	*f = out
	return nil
}

// oracleFrame is a column-oriented frame: t is Unix seconds, o/h/l/c/a are
// prices and adjusted close, v volume, d dividends, s splits and rep the
// indices of rows flagged Repaired.
type oracleFrame struct {
	T   []int64   `json:"t"`
	O   nanFloats `json:"o"`
	H   nanFloats `json:"h"`
	L   nanFloats `json:"l"`
	C   nanFloats `json:"c"`
	A   nanFloats `json:"a"`
	V   []int64   `json:"v"`
	D   nanFloats `json:"d"`
	S   nanFloats `json:"s"`
	Rep []int     `json:"rep"`
}

type oracleCase struct {
	Name             string          `json:"name"`
	Kind             string          `json:"kind"`
	Interval         string          `json:"interval"`
	TZ               string          `json:"tz"`
	Currency         string          `json:"currency"`
	CurrencyRepaired bool            `json:"currencyRepaired"`
	Now              int64           `json:"now"`
	Change           *float64        `json:"change"`
	PerColumn        bool            `json:"perColumn"`
	MinChange        float64         `json:"minChange"`
	Bars             oracleFrame     `json:"bars"`
	Hourly           *oracleFrame    `json:"hourly"`
	Fetches          int             `json:"fetches"`
	Want             json.RawMessage `json:"want"`
	Calls            []oracleCall    `json:"calls"`
}

type oracleCall struct {
	Change       float64         `json:"change"`
	Rows         *int            `json:"rows"`
	PerColumn    *bool           `json:"perColumn"`
	Threshold    *float64        `json:"threshold"`
	Detected     json.RawMessage `json:"detected"`
	VolThreshold json.RawMessage `json:"volThreshold"`
	AfterVolume  json.RawMessage `json:"afterVolume"`
	AfterLocal   json.RawMessage `json:"afterLocal"`
	RangesIn     json.RawMessage `json:"rangesIn"`
	RangesOut    json.RawMessage `json:"rangesOut"`
}

func loadOracleCorpus(t *testing.T, group string) []oracleCase {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "oracle", group+".json"))
	if err != nil {
		t.Fatalf("read corpus %s: %v", group, err)
	}
	var doc struct {
		Cases []oracleCase `json:"cases"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("decode corpus %s: %v", group, err)
	}
	if len(doc.Cases) == 0 {
		t.Fatalf("corpus %s is empty", group)
	}
	return doc.Cases
}

func oracleLocation(t *testing.T, tz string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(tz)
	if err != nil {
		t.Fatalf("load location %s: %v", tz, err)
	}
	return loc
}

func (f oracleFrame) bars(loc *time.Location) []models.Bar {
	col := func(c nanFloats, i int) float64 {
		if c == nil {
			return 0
		}
		return c[i]
	}
	bars := make([]models.Bar, len(f.T))
	for i := range bars {
		bars[i] = models.Bar{
			Date:      time.Unix(f.T[i], 0).In(loc),
			Open:      col(f.O, i),
			High:      col(f.H, i),
			Low:       col(f.L, i),
			Close:     col(f.C, i),
			AdjClose:  col(f.A, i),
			Dividends: col(f.D, i),
			Splits:    col(f.S, i),
		}
		if f.V != nil {
			bars[i].Volume = f.V[i]
		}
	}
	for _, i := range f.Rep {
		bars[i].Repaired = true
	}
	return bars
}

// wantBars applies the expected changed columns to the input frame.
func (c oracleCase) wantBars(t *testing.T, loc *time.Location) []models.Bar {
	t.Helper()
	var w oracleFrame
	if err := json.Unmarshal(c.Want, &w); err != nil {
		t.Fatalf("%s: decode want: %v", c.Name, err)
	}
	f := c.Bars
	for _, p := range []struct{ dst, src *nanFloats }{{&f.O, &w.O}, {&f.H, &w.H}, {&f.L, &w.L}, {&f.C, &w.C}, {&f.A, &w.A}, {&f.D, &w.D}, {&f.S, &w.S}} {
		if *p.src != nil {
			*p.dst = *p.src
		}
	}
	if w.V != nil {
		f.V = w.V
	}
	f.Rep = w.Rep
	return f.bars(loc)
}

// runOracleCase runs one end-to-end case through the Go repair entry point
// matching the upstream function that produced it.
func runOracleCase(t *testing.T, c oracleCase, trace func(*priceShiftTrace)) []models.Bar {
	t.Helper()
	loc := oracleLocation(t, c.TZ)
	bars := c.Bars.bars(loc)
	opts := DefaultOptions()
	opts.Interval = c.Interval
	opts.Timezone = c.TZ
	opts.Currency = c.Currency
	r := New(opts)
	r.traceShift = trace
	switch c.Kind {
	case "split":
		return r.repairStockSplits(bars)
	case "unit_switch":
		r.currencyRepaired = c.CurrencyRepaired
		return r.repairUnitSwitch(bars)
	case "level_shift":
		hourly := c.Hourly.bars(loc)
		fetches := 0
		r.opts.Now = func() time.Time { return time.Unix(c.Now, 0) }
		r.opts.FetchHourly = func(start, end time.Time) ([]models.Bar, error) {
			fetches++
			var out []models.Bar
			for _, b := range hourly {
				if !b.Date.Before(start) && b.Date.Before(end) {
					out = append(out, b)
				}
			}
			return out, nil
		}
		got, err := r.repairLevelShifts(bars)
		if err != nil {
			t.Fatalf("%s: repairLevelShifts: %v", c.Name, err)
		}
		if fetches != c.Fetches {
			t.Errorf("%s: %d hourly fetches, upstream %d", c.Name, fetches, c.Fetches)
		}
		return got
	}
	t.Fatalf("%s: unknown kind %q", c.Name, c.Kind)
	return nil
}

func oracleClose(got, want float64) bool {
	if math.IsNaN(got) || math.IsNaN(want) {
		return math.IsNaN(got) && math.IsNaN(want)
	}
	if got == want {
		return true
	}
	return math.Abs(got-want) <= oracleRTol*math.Abs(want)
}

// diffBars describes the differences between got and want, or "".
func diffBars(got, want []models.Bar) string {
	if len(got) != len(want) {
		return fmt.Sprintf("%d rows, want %d", len(got), len(want))
	}
	var sb strings.Builder
	n := 0
	for i := range want {
		g, w := got[i], want[i]
		for _, f := range []struct {
			name   string
			gv, wv float64
		}{{"Open", g.Open, w.Open}, {"High", g.High, w.High}, {"Low", g.Low, w.Low}, {"Close", g.Close, w.Close},
			{"AdjClose", g.AdjClose, w.AdjClose}, {"Dividends", g.Dividends, w.Dividends}} {
			if !oracleClose(f.gv, f.wv) {
				n++
				if n <= 5 {
					fmt.Fprintf(&sb, "\n  row %d %s: got %v, want %v", i, f.name, f.gv, f.wv)
				}
			}
		}
		if g.Volume != w.Volume {
			n++
			if n <= 5 {
				fmt.Fprintf(&sb, "\n  row %d Volume: got %d, want %d", i, g.Volume, w.Volume)
			}
		}
		if g.Repaired != w.Repaired {
			n++
			if n <= 5 {
				fmt.Fprintf(&sb, "\n  row %d Repaired: got %v, want %v", i, g.Repaired, w.Repaired)
			}
		}
	}
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("%d differences:%s", n, sb.String())
}

// TestOracleCorpusRepair replays every end-to-end case of the upstream
// differential corpus and compares the repaired frames.
func TestOracleCorpusRepair(t *testing.T) {
	for _, group := range []string{"split", "unit_switch", "level_shift", "fixtures", "mined"} {
		group := group
		t.Run(group, func(t *testing.T) {
			cases := loadOracleCorpus(t, group)
			failed := 0
			for _, c := range cases {
				got := runOracleCase(t, c, nil)
				if d := diffBars(got, c.wantBars(t, oracleLocation(t, c.TZ))); d != "" {
					failed++
					t.Errorf("%s: %s", c.Name, d)
				}
			}
			t.Logf("%s: %d/%d cases match upstream", group, len(cases)-failed, len(cases))
		})
	}
}

// oracleMaskCells normalises an upstream mask (row list, or [row, col]
// list) to sorted "row" / "row:col" strings.
func oracleMaskCells(t *testing.T, raw json.RawMessage) []string {
	t.Helper()
	var rows []int
	if err := json.Unmarshal(raw, &rows); err == nil {
		out := make([]string, len(rows))
		for i, r := range rows {
			out[i] = fmt.Sprint(r)
		}
		return out
	}
	var cells [][2]int
	if err := json.Unmarshal(raw, &cells); err != nil {
		t.Fatalf("decode mask %s: %v", raw, err)
	}
	out := make([]string, len(cells))
	for i, c := range cells {
		out[i] = fmt.Sprintf("%d:%d", c[0], c[1])
	}
	return out
}

func goMaskCells(m [][]bool) []string {
	var out []string
	if len(m) == 1 {
		for i, v := range m[0] {
			if v {
				out = append(out, fmt.Sprint(i))
			}
		}
		return out
	}
	for i := range m[0] {
		for c := range m {
			if m[c][i] {
				out = append(out, fmt.Sprintf("%d:%d", i, c))
			}
		}
	}
	return out
}

// compareMasks compares a Go detection with an upstream masks object
// ({"up": ..., "down": ...} or null).
func compareMasks(t *testing.T, raw json.RawMessage, det *priceShiftDetection) string {
	t.Helper()
	if len(raw) == 0 || string(raw) == "null" {
		if det != nil {
			return fmt.Sprintf("got up=%v down=%v, want None", goMaskCells(det.up), goMaskCells(det.down))
		}
		return ""
	}
	var m struct {
		Up   json.RawMessage `json:"up"`
		Down json.RawMessage `json:"down"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("decode masks: %v", err)
	}
	wantUp, wantDown := oracleMaskCells(t, m.Up), oracleMaskCells(t, m.Down)
	if det == nil {
		return fmt.Sprintf("got None, want up=%v down=%v", wantUp, wantDown)
	}
	gotUp, gotDown := goMaskCells(det.up), goMaskCells(det.down)
	if strings.Join(gotUp, ",") != strings.Join(wantUp, ",") || strings.Join(gotDown, ",") != strings.Join(wantDown, ",") {
		return fmt.Sprintf("got up=%v down=%v, want up=%v down=%v", gotUp, gotDown, wantUp, wantDown)
	}
	return ""
}

func encodeRanges(ranges []priceShiftRange, change float64) string {
	parts := make([]string, len(ranges))
	for i, r := range ranges {
		kind := 0
		if r.multiplier == change {
			kind = 1
		}
		parts[i] = fmt.Sprintf("[%d,%d,%d]", r.start, r.end, kind)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func compactJSON(raw json.RawMessage) string {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// compareTrace compares one Go fixPricesSuddenChange trace with upstream's
// record of the same call.
func compareTrace(t *testing.T, want oracleCall, got *priceShiftTrace) []string {
	t.Helper()
	var errs []string
	add := func(format string, args ...any) { errs = append(errs, fmt.Sprintf(format, args...)) }
	if !oracleClose(got.change, want.Change) {
		add("change %v, want %v", got.change, want.Change)
	}
	if (want.Rows != nil) != got.detectCalled {
		add("detector called=%v, upstream %v", got.detectCalled, want.Rows != nil)
		return errs
	}
	if want.Rows == nil {
		return errs
	}
	if *want.Rows != got.rows || *want.PerColumn != got.perColumn {
		add("rows=%d perColumn=%v, want %d %v", got.rows, got.perColumn, *want.Rows, *want.PerColumn)
	}
	if d := compareMasks(t, want.Detected, got.detected); d != "" {
		add("detected: %s", d)
	}
	if got.detected != nil && want.Threshold != nil && !oracleClose(got.threshold, *want.Threshold) {
		add("threshold %v, want %v", got.threshold, *want.Threshold)
	}
	if (len(want.VolThreshold) > 0) != got.volumeCalled {
		add("volume threshold estimated=%v, upstream %v", got.volumeCalled, len(want.VolThreshold) > 0)
	} else if got.volumeCalled {
		var wv *float64
		if err := json.Unmarshal(want.VolThreshold, &wv); err != nil {
			t.Fatalf("decode volThreshold: %v", err)
		}
		if (wv != nil) != got.volThresholdOK || (wv != nil && !oracleClose(got.volThreshold, *wv)) {
			add("volume threshold %v (ok=%v), want %s", got.volThreshold, got.volThresholdOK, want.VolThreshold)
		}
	}
	if len(want.AfterVolume) > 0 {
		if d := compareMasks(t, want.AfterVolume, got.afterVolume); d != "" {
			add("after volume spikes: %s", d)
		}
	}
	if len(want.AfterLocal) > 0 {
		if d := compareMasks(t, want.AfterLocal, got.afterLocal); d != "" {
			add("after local stdev: %s", d)
		}
	}
	if (len(want.RangesIn) > 0) != got.reconciled {
		add("reconciled=%v, upstream %v", got.reconciled, len(want.RangesIn) > 0)
	} else if got.reconciled {
		if g, w := encodeRanges(got.rangesIn, got.change), compactJSON(want.RangesIn); g != w {
			add("ranges %s, want %s", g, w)
		}
		if g, w := encodeRanges(got.rangesOut, got.change), compactJSON(want.RangesOut); g != w {
			add("reconciled ranges %s, want %s", g, w)
		}
	}
	return errs
}

// TestOracleCorpusStages compares every stage of every
// fixPricesSuddenChange call (detector masks and threshold, volume
// threshold, volume-spike and local-volatility screening, ranges before and
// after the volume reconcile) with upstream.
func TestOracleCorpusStages(t *testing.T) {
	for _, group := range []string{"split", "unit_switch", "level_shift", "fixtures", "mined"} {
		group := group
		t.Run(group, func(t *testing.T) {
			calls := 0
			for _, c := range loadOracleCorpus(t, group) {
				var traces []*priceShiftTrace
				runOracleCase(t, c, func(tr *priceShiftTrace) { traces = append(traces, tr) })
				if len(traces) != len(c.Calls) {
					t.Errorf("%s: %d fixPricesSuddenChange calls, upstream %d", c.Name, len(traces), len(c.Calls))
					continue
				}
				for i := range traces {
					calls++
					for _, e := range compareTrace(t, c.Calls[i], traces[i]) {
						t.Errorf("%s call %d: %s", c.Name, i, e)
					}
				}
			}
			t.Logf("%s: %d calls compared", group, calls)
		})
	}
}

// TestOracleCorpusDetector runs detectPriceShifts on frames passed directly
// to upstream _detect_price_shifts (newest first), with and without a
// change, per column or denoised.
func TestOracleCorpusDetector(t *testing.T) {
	cases := loadOracleCorpus(t, "detect")
	for _, c := range cases {
		bars := c.Bars.bars(oracleLocation(t, c.TZ))
		change := 0.0
		if c.Change != nil {
			change = *c.Change
		}
		det := detectPriceShifts(bars, c.Interval, change, c.PerColumn, c.MinChange)
		var want struct {
			Detected  json.RawMessage `json:"detected"`
			Threshold *float64        `json:"threshold"`
		}
		if err := json.Unmarshal(c.Want, &want); err != nil {
			t.Fatalf("%s: decode want: %v", c.Name, err)
		}
		if d := compareMasks(t, want.Detected, det); d != "" {
			t.Errorf("%s: %s", c.Name, d)
		}
		if det != nil && want.Threshold != nil && !oracleClose(det.threshold, *want.Threshold) {
			t.Errorf("%s: threshold %v, want %v", c.Name, det.threshold, *want.Threshold)
		}
	}
	t.Logf("%d detector cases", len(cases))
}

// TestOracleCorpusCoverage guards the corpus against silently shrinking.
func TestOracleCorpusCoverage(t *testing.T) {
	kinds := map[string]int{}
	for _, group := range []string{"split", "unit_switch", "level_shift", "detect", "fixtures", "mined"} {
		for _, c := range loadOracleCorpus(t, group) {
			kinds[c.Kind]++
		}
	}
	keys := make([]string, 0, len(kinds))
	for k := range kinds {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if kinds[k] < 50 {
			t.Errorf("only %d %s cases", kinds[k], k)
		}
	}
}

// TestOracleNumPyReductions pins the NumPy reductions in numpy.go bit for
// bit against values computed by NumPy.
func TestOracleNumPyReductions(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "oracle", "numpy.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Cases []struct {
			X      []float64 `json:"x"`
			Sum    float64   `json:"sum"`
			Mean   float64   `json:"mean"`
			Std0   float64   `json:"std0"`
			Std1   *float64  `json:"std1"`
			P25    float64   `json:"p25"`
			P75    float64   `json:"p75"`
			Median float64   `json:"median"`
			Round3 []float64 `json:"round3"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	for i, c := range doc.Cases {
		check := func(name string, got, want float64) {
			if got != want {
				t.Errorf("case %d (n=%d) %s: got %v, want %v", i, len(c.X), name, got, want)
			}
		}
		check("sum", npSum(c.X), c.Sum)
		check("mean", npMean(c.X), c.Mean)
		check("std", npStd(c.X, 0), c.Std0)
		if c.Std1 != nil {
			check("std ddof=1", npStd(c.X, 1), *c.Std1)
		}
		check("p25", npPercentile(c.X, 25), c.P25)
		check("p75", npPercentile(c.X, 75), c.P75)
		check("median", npMedian(c.X), c.Median)
		for j, v := range c.X {
			check("round3", npRound3(v), c.Round3[j])
		}
	}
}

// TestIntervalGapWithin30Days pins the 100x-abort window: day and week
// multiples are absolute, months are calendar offsets clamped to the month
// end (dateutil relativedelta), compared with dt + 30 days.
func TestIntervalGapWithin30Days(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	feb := time.Date(2021, 2, 1, 0, 0, 0, 0, ny)
	apr := time.Date(2021, 4, 1, 0, 0, 0, 0, ny)
	jan31 := time.Date(2021, 1, 31, 0, 0, 0, 0, ny)
	for _, tc := range []struct {
		interval string
		gap      int
		dt       time.Time
		want     bool
	}{
		{"1d", 29, feb, true}, {"1d", 30, feb, false},
		{"1wk", 4, feb, true}, {"1wk", 5, feb, false},
		{"1mo", 1, feb, true},   // 28 days
		{"1mo", 1, apr, false},  // 30 days: equal, not less
		{"1mo", 1, jan31, true}, // clamped to 28 February
		{"3mo", 1, feb, false},
		{"1h", 719, feb, true}, {"1h", 720, feb, false},
		{"5m", 8639, feb, true}, {"5m", 8640, feb, false},
	} {
		if got := intervalGapWithin30Days(tc.interval, tc.gap, tc.dt); got != tc.want {
			t.Errorf("%s x%d at %s: got %v, want %v", tc.interval, tc.gap, tc.dt.Format(time.DateOnly), got, tc.want)
		}
	}
	if got := addMonthsClamped(time.Date(2020, 1, 31, 9, 30, 0, 0, ny), 13); !got.Equal(time.Date(2021, 2, 28, 9, 30, 0, 0, ny)) {
		t.Errorf("addMonthsClamped = %v", got)
	}
}
