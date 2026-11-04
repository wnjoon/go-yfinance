package ticker

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

// The History pipeline fixture (testdata/history_repair) is one raw Yahoo
// chart payload with a missing 2:1 split adjustment (rows 0-44), a dividend
// correctly reflected in Adj Close (row 25), a fully null row (15), a row
// with a null Open (60) and a 100x unit-switch block (85-90). expected.json
// holds Python yfinance 1.8.0's history() output for every combination of
// repair, actions, auto_adjust and keepna (see gen_history_oracle.py).

const (
	pipelineNullRow     = 15
	pipelineOpenNullRow = 60
)

type pipelineWant struct {
	T       []int64    `json:"t"`
	O       []*float64 `json:"o"`
	H       []*float64 `json:"h"`
	L       []*float64 `json:"l"`
	C       []*float64 `json:"c"`
	A       []*float64 `json:"a"`
	V       []int64    `json:"v"`
	D       []*float64 `json:"d"`
	S       []*float64 `json:"s"`
	Rep     []int      `json:"rep"`
	Columns []string   `json:"columns"`
}

type pipelineCase struct {
	Repair     bool         `json:"repair"`
	Actions    bool         `json:"actions"`
	AutoAdjust bool         `json:"autoAdjust"`
	KeepNA     bool         `json:"keepna"`
	Want       pipelineWant `json:"want"`
}

func (c pipelineCase) name() string {
	return fmt.Sprintf("repair=%v/actions=%v/autoAdjust=%v/keepNA=%v", c.Repair, c.Actions, c.AutoAdjust, c.KeepNA)
}

func (c pipelineCase) params() models.HistoryParams {
	return models.HistoryParams{Period: "1y", Interval: "1d", Repair: c.Repair, Actions: c.Actions, AutoAdjust: c.AutoAdjust, KeepNA: c.KeepNA}
}

func readPipelineFile(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "history_repair", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func loadPipelineCases(t *testing.T) []pipelineCase {
	t.Helper()
	cases := loadPipelineFile(t, "expected.json")
	if len(cases) != 16 {
		t.Fatalf("expected.json: %d cases", len(cases))
	}
	return cases
}

func loadPipelineFile(t *testing.T, name string) []pipelineCase {
	t.Helper()
	var doc struct {
		Upstream string         `json:"upstream"`
		Cases    []pipelineCase `json:"cases"`
	}
	if err := json.Unmarshal(readPipelineFile(t, name), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Upstream != "1.8.0" {
		t.Fatalf("%s: upstream %q", name, doc.Upstream)
	}
	return doc.Cases
}

// findPipelineCase returns the upstream case with the given flags.
func findPipelineCase(t *testing.T, cases []pipelineCase, repair, actions, autoAdjust, keepNA bool) pipelineCase {
	t.Helper()
	for _, c := range cases {
		if c.Repair == repair && c.Actions == actions && c.AutoAdjust == autoAdjust && c.KeepNA == keepNA {
			return c
		}
	}
	t.Fatalf("no upstream case repair=%v actions=%v autoAdjust=%v keepNA=%v", repair, actions, autoAdjust, keepNA)
	return pipelineCase{}
}

// pipelineTicker serves the fixture for 1d requests; any other request (the
// level-shift 1h fetch) fails, as no level-shift candidate is recent enough.
func pipelineTicker(t *testing.T, body string) (*Ticker, *int) {
	t.Helper()
	tkr, err := New("MTRX")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(tkr.Close)
	calls := 0
	tkr.chartBodyFetcher = func(p url.Values) (string, error) {
		calls++
		if p.Get("interval") != "1d" {
			return "", errors.New("unexpected request " + p.Encode())
		}
		return body, nil
	}
	now := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	tkr.clock = func() time.Time { return now }
	return tkr, &calls
}

func pipelineHistory(t *testing.T, c pipelineCase) []models.Bar {
	t.Helper()
	return pipelineHistoryFrom(t, "chart.json", c.params())
}

func pipelineHistoryFrom(t *testing.T, chart string, params models.HistoryParams) []models.Bar {
	t.Helper()
	c := pipelineCase{Repair: params.Repair, Actions: params.Actions, AutoAdjust: params.AutoAdjust, KeepNA: params.KeepNA}
	tkr, calls := pipelineTicker(t, string(readPipelineFile(t, chart)))
	bars, err := tkr.History(params)
	if err != nil {
		t.Fatalf("%s: History: %v", c.name(), err)
	}
	if *calls != 1 {
		t.Fatalf("%s: %d chart requests, want 1", c.name(), *calls)
	}
	return bars
}

func pipelineClose(got float64, want *float64) bool {
	if want == nil {
		// History reports a missing price as 0.
		return got == 0
	}
	if got == *want {
		return true
	}
	return math.Abs(got-*want) <= 1e-9*math.Abs(*want)
}

func hasColumn(cols []string, name string) bool {
	for _, c := range cols {
		if c == name {
			return true
		}
	}
	return false
}

// diffPipeline compares History bars with upstream's frame.
func diffPipeline(got []models.Bar, w pipelineWant) string {
	if len(got) != len(w.T) {
		return fmt.Sprintf("%d rows, want %d", len(got), len(w.T))
	}
	rep := make(map[int]bool, len(w.Rep))
	for _, i := range w.Rep {
		rep[i] = true
	}
	actions := hasColumn(w.Columns, "Dividends")
	var sb strings.Builder
	n := 0
	add := func(format string, args ...any) {
		n++
		if n <= 8 {
			fmt.Fprintf(&sb, "\n  "+format, args...)
		}
	}
	for i, b := range got {
		if b.Date.Unix() != w.T[i] {
			add("row %d date %v, want %v", i, b.Date.Unix(), w.T[i])
			continue
		}
		for _, f := range []struct {
			name string
			g    float64
			w    []*float64
		}{{"Open", b.Open, w.O}, {"High", b.High, w.H}, {"Low", b.Low, w.L}, {"Close", b.Close, w.C}} {
			if !pipelineClose(f.g, f.w[i]) {
				add("row %d %s: got %v, want %v", i, f.name, f.g, deref(f.w[i]))
			}
		}
		if w.A != nil && !pipelineClose(b.AdjClose, w.A[i]) {
			add("row %d AdjClose: got %v, want %v", i, b.AdjClose, deref(w.A[i]))
		}
		if b.Volume != w.V[i] {
			add("row %d Volume: got %d, want %d", i, b.Volume, w.V[i])
		}
		if actions {
			if !pipelineClose(b.Dividends, w.D[i]) {
				add("row %d Dividends: got %v, want %v", i, b.Dividends, deref(w.D[i]))
			}
			if !pipelineClose(b.Splits, w.S[i]) {
				add("row %d Splits: got %v, want %v", i, b.Splits, deref(w.S[i]))
			}
		} else if b.Dividends != 0 || b.Splits != 0 || b.CapitalGains != 0 || b.DividendCurrency != "" {
			add("row %d has actions without Actions: %+v", i, b)
		}
		if b.Repaired != rep[i] {
			add("row %d Repaired: got %v, want %v", i, b.Repaired, rep[i])
		}
	}
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("%d differences:%s", n, sb.String())
}

func deref(p *float64) any {
	if p == nil {
		return "NaN"
	}
	return *p
}

// TestHistoryWithoutRepairUnchanged pins History output without Repair to
// what main produced before the repair pipeline was reordered
// (no_repair_main.json), byte for byte.
func TestHistoryWithoutRepairUnchanged(t *testing.T) {
	var golden []struct {
		Actions, AutoAdjust, KeepNA bool
		Bars                        json.RawMessage
	}
	if err := json.Unmarshal(readPipelineFile(t, "no_repair_main.json"), &golden); err != nil {
		t.Fatal(err)
	}
	if len(golden) != 8 {
		t.Fatalf("golden has %d cases", len(golden))
	}
	for _, g := range golden {
		c := pipelineCase{Actions: g.Actions, AutoAdjust: g.AutoAdjust, KeepNA: g.KeepNA}
		got, err := json.Marshal(pipelineHistory(t, c))
		if err != nil {
			t.Fatal(err)
		}
		var want []models.Bar
		if err := json.Unmarshal(g.Bars, &want); err != nil {
			t.Fatal(err)
		}
		wantJSON, _ := json.Marshal(want)
		if string(got) != string(wantJSON) {
			t.Errorf("%s: output differs from main", c.name())
		}
	}
}

// indexOfTime returns the index of the bar at Unix time ts, or -1.
func indexOfTime(bars []models.Bar, ts int64) int {
	for i, b := range bars {
		if b.Date.Unix() == ts {
			return i
		}
	}
	return -1
}

// pipelineTimestamps returns the fixture's chart timestamps.
func pipelineTimestamps(t *testing.T, chart string) []int64 {
	t.Helper()
	var resp models.ChartResponse
	if err := json.Unmarshal(readPipelineFile(t, chart), &resp); err != nil {
		t.Fatal(err)
	}
	return resp.Chart.Result[0].Timestamp
}

// TestHistoryRepairKeepsNullPricesMissing pins issue #2: Yahoo's null
// prices are NaN while repairing, as upstream sees them. As 0, the null
// Open made unit-switch repair correct columns individually (a 0..x
// spread), which left the 100x block unrepaired. With FixZeroes off (Go's
// interpolation has no upstream equivalent) the whole frame matches
// upstream, and the null prices come back as 0, never NaN.
func TestHistoryRepairKeepsNullPricesMissing(t *testing.T) {
	ts := pipelineTimestamps(t, "chart.json")
	want := findPipelineCase(t, loadPipelineCases(t), true, true, false, true).Want
	opts := models.DefaultRepairOptions()
	opts.FixZeroes = false
	params := models.HistoryParams{Period: "1y", Interval: "1d", Repair: true, Actions: true, KeepNA: true, RepairOptions: &opts}
	got := pipelineHistoryFrom(t, "chart.json", params)
	if d := diffPipeline(got, want); d != "" {
		t.Errorf("%s", d)
	}
	for _, row := range []int{85, 90} {
		b := got[indexOfTime(got, ts[row])]
		if !b.Repaired || b.Close > 1000 {
			t.Errorf("100x row %d not repaired: %+v", row, b)
		}
	}
	null := got[indexOfTime(got, ts[pipelineNullRow])]
	if null.Open != 0 || null.High != 0 || null.Low != 0 || null.Close != 0 || null.AdjClose != 0 || null.Repaired {
		t.Errorf("null row = %+v, want zero prices, not repaired", null)
	}
	openNull := got[indexOfTime(got, ts[pipelineOpenNullRow])]
	if openNull.Open != 0 || openNull.Low <= 0 || openNull.Repaired {
		t.Errorf("null-Open row = %+v, want Open 0 and the rest untouched", openNull)
	}
	if _, err := json.Marshal(got); err != nil {
		t.Errorf("bars do not marshal (NaN left?): %v", err)
	}
}

// TestHistoryRepairWithoutActionsUsesEvents pins issue #3: Repair needs the
// split and dividend events even when Actions is false; they are dropped
// from the result afterwards, as upstream drops its action columns.
func TestHistoryRepairWithoutActionsUsesEvents(t *testing.T) {
	ts := pipelineTimestamps(t, "chart.json")
	params := models.HistoryParams{Period: "1y", Interval: "1d", Repair: true}
	got := pipelineHistoryFrom(t, "chart.json", params)
	withActions := params
	withActions.Actions = true
	ref := pipelineHistoryFrom(t, "chart.json", withActions)
	if len(got) != len(ref) {
		t.Fatalf("%d bars, %d with Actions", len(got), len(ref))
	}
	for i := range got {
		r := ref[i]
		r.Dividends, r.Splits, r.CapitalGains, r.DividendCurrency = 0, 0, 0, ""
		if got[i] != r {
			t.Fatalf("row %d: %+v, with Actions (events cleared) %+v", i, got[i], r)
		}
	}
	// The missing 2:1 adjustment before the split was repaired.
	if b := got[0]; !b.Repaired || b.Close > 150 {
		t.Errorf("row 0 not split-repaired: %+v", b)
	}
	if i := indexOfTime(ref, ts[45]); ref[i].Splits != 2 {
		t.Errorf("split event missing with Actions: %+v", ref[i])
	}
}
