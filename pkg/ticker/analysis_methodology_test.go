package ticker

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/internal/endpoints"
	"github.com/wnjoon/go-yfinance/pkg/client"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

// fakeQuoteSummary serves canned quoteSummary responses keyed by the
// "modules" query parameter and records every request.
type fakeQuoteSummary struct {
	mu        sync.Mutex
	responses map[string]*client.Response
	requests  []string
	urls      []string
	params    []url.Values
	// block, when set for a module, is received from before responding.
	block map[string]chan struct{}
}

func newFakeQuoteSummary() *fakeQuoteSummary {
	return &fakeQuoteSummary{
		responses: make(map[string]*client.Response),
		block:     make(map[string]chan struct{}),
	}
}

func (f *fakeQuoteSummary) set(modules string, status int, body string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.responses[modules] = &client.Response{StatusCode: status, Body: body}
}

func (f *fakeQuoteSummary) setFixture(t *testing.T, modules, file string) {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", "analysis", file))
	if err != nil {
		t.Fatalf("read fixture %s: %v", file, err)
	}
	f.set(modules, 200, string(body))
}

func (f *fakeQuoteSummary) get(rawURL string, params url.Values) (*client.Response, error) {
	modules := params.Get("modules")
	f.mu.Lock()
	f.requests = append(f.requests, modules)
	f.urls = append(f.urls, rawURL)
	f.params = append(f.params, params)
	resp, ok := f.responses[modules]
	block := f.block[modules]
	f.mu.Unlock()

	if block != nil {
		<-block
	}
	if !ok {
		return nil, fmt.Errorf("unexpected quoteSummary modules %q", modules)
	}
	clone := *resp
	return &clone, nil
}

func (f *fakeQuoteSummary) requested() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.requests...)
}

func (f *fakeQuoteSummary) count(modules string) int {
	n := 0
	for _, m := range f.requested() {
		if m == modules {
			n++
		}
	}
	return n
}

// newMSFTAnalysisTicker returns a ticker whose quoteSummary requests are
// served from the MSFT fixtures captured from Yahoo (formatted=false).
func newMSFTAnalysisTicker(t *testing.T) (*Ticker, *fakeQuoteSummary) {
	t.Helper()
	fake := newFakeQuoteSummary()
	for _, m := range []string{
		"earningsTrend", "earningsTrendGaap", "earningsTrendNonGaap",
		"earningsHistory", "earningsHistoryGaap", "earningsHistoryNonGaap",
	} {
		fake.setFixture(t, m, "msft_"+m+".json")
	}
	fake.setFixture(t, "industryTrend,sectorTrend,indexTrend", "msft_peerTrends.json")
	return newFakeAnalysisTicker(t, fake), fake
}

func newFakeAnalysisTicker(t *testing.T, fake *fakeQuoteSummary) *Ticker {
	t.Helper()
	tkr, err := New("MSFT")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(tkr.Close)
	tkr.quoteSummaryGetter = fake.get
	return tkr
}

// quoteSummaryBody wraps a single module payload in a quoteSummary envelope.
func quoteSummaryBody(module, payload string) string {
	return fmt.Sprintf(`{"quoteSummary":{"result":[{%q:%s}],"error":null}}`, module, payload)
}

func assertFloat(t *testing.T, name string, got, want float64) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
}

func TestEarningsEstimateDefaultRegression(t *testing.T) {
	tkr, fake := newMSFTAnalysisTicker(t)

	est, err := tkr.EarningsEstimate()
	if err != nil {
		t.Fatalf("EarningsEstimate: %v", err)
	}
	if len(est) != 4 {
		t.Fatalf("rows = %d, want 4", len(est))
	}
	want := models.EarningsEstimate{
		Period:           "0q",
		EndDate:          "2026-12-31",
		Currency:         "USD",
		NumberOfAnalysts: 25,
		Avg:              4.80272,
		Low:              3.53,
		High:             4.97073,
		YearAgoEPS:       4.14,
		Growth:           0.1601,
		Methodology:      "normalized", // defaultMethodology "nongaap"
	}
	if est[0] != want {
		t.Fatalf("row 0 = %+v, want %+v", est[0], want)
	}
	periods := []string{est[0].Period, est[1].Period, est[2].Period, est[3].Period}
	if !reflect.DeepEqual(periods, []string{"0q", "+1q", "0y", "+1y"}) {
		t.Fatalf("periods = %v", periods)
	}

	if got := fake.requested(); !reflect.DeepEqual(got, []string{"earningsTrend"}) {
		t.Fatalf("requests = %v, want [earningsTrend]", got)
	}
	p := fake.params[0]
	if p.Get("formatted") != "false" || p.Get("corsDomain") != "finance.yahoo.com" {
		t.Fatalf("unexpected params %v", p)
	}
	if fake.urls[0] != endpoints.QuoteSummaryURL+"/MSFT" {
		t.Fatalf("url = %s", fake.urls[0])
	}

	// Deprecated alias and explicit default share the same cache entry.
	alias, err := tkr.EarningsEstimates()
	if err != nil {
		t.Fatal(err)
	}
	explicit, err := tkr.EarningsEstimateWithMethodology(models.EPSMethodologyDefault)
	if err != nil {
		t.Fatal(err)
	}
	if &alias[0] != &est[0] || &explicit[0] != &est[0] {
		t.Fatal("default estimate not served from cache")
	}
	if n := len(fake.requested()); n != 1 {
		t.Fatalf("requests = %d, want 1", n)
	}
}

func TestEPSTablesDefaultRegression(t *testing.T) {
	tkr, fake := newMSFTAnalysisTicker(t)

	trend, err := tkr.EPSTrend()
	if err != nil {
		t.Fatal(err)
	}
	wantTrend := models.EPSTrend{
		Period: "0q", Currency: "USD", Current: 4.80272, SevenDays: 4.81524,
		ThirtyDays: trend[0].ThirtyDays, SixtyDays: trend[0].SixtyDays, NinetyDays: trend[0].NinetyDays,
		Methodology: "normalized",
	}
	if len(trend) != 4 || trend[0] != wantTrend {
		t.Fatalf("EPSTrend[0] = %+v (len %d)", trend[0], len(trend))
	}

	rev, err := tkr.EPSRevisions()
	if err != nil {
		t.Fatal(err)
	}
	if len(rev) != 4 || rev[0].Period != "0q" || rev[0].UpLast30Days != 8 || rev[0].Currency != "USD" || rev[0].Methodology != "normalized" {
		t.Fatalf("EPSRevisions[0] = %+v (len %d)", rev[0], len(rev))
	}

	hist, err := tkr.EarningsHistory()
	if err != nil {
		t.Fatal(err)
	}
	if len(hist.History) != 4 {
		t.Fatalf("history rows = %d", len(hist.History))
	}
	h := hist.History[0]
	if h.Period != "-4q" || !h.Quarter.Equal(time.Unix(1767139200, 0)) || h.EPSActual != 4.14 ||
		h.EPSEstimate != 3.91702 || h.EPSDifference != 0.22 || h.SurprisePercent != 0.056900002 || h.Methodology != "normalized" {
		t.Fatalf("history[0] = %+v", h)
	}

	// Trend tables share one earningsTrend request; history is separate.
	if got := fake.requested(); !reflect.DeepEqual(got, []string{"earningsTrend", "earningsHistory"}) {
		t.Fatalf("requests = %v", got)
	}
}

func TestEPSMethodologyRequestsSuffixedModules(t *testing.T) {
	type call func(*Ticker, models.EPSMethodology) (first float64, label string, err error)
	calls := map[string]struct {
		base string
		fn   call
		// expected first-row value per methodology
		want map[models.EPSMethodology]float64
	}{
		"EarningsEstimate": {"earningsTrend", func(tk *Ticker, m models.EPSMethodology) (float64, string, error) {
			r, err := tk.EarningsEstimateWithMethodology(m)
			if err != nil {
				return 0, "", err
			}
			return r[0].Avg, r[0].Methodology, nil
		}, map[models.EPSMethodology]float64{"": 4.80272, "gaap": 4.86377, "normalized": 4.80272}},
		"EPSTrend": {"earningsTrend", func(tk *Ticker, m models.EPSMethodology) (float64, string, error) {
			r, err := tk.EPSTrendWithMethodology(m)
			if err != nil {
				return 0, "", err
			}
			return r[0].SevenDays, r[0].Methodology, nil
		}, map[models.EPSMethodology]float64{"": 4.81524, "gaap": 4.84591, "normalized": 4.81524}},
		"EPSRevisions": {"earningsTrend", func(tk *Ticker, m models.EPSMethodology) (float64, string, error) {
			r, err := tk.EPSRevisionsWithMethodology(m)
			if err != nil {
				return 0, "", err
			}
			return float64(r[0].UpLast30Days), r[0].Methodology, nil
		}, map[models.EPSMethodology]float64{"": 8, "gaap": 5, "normalized": 8}},
		"EarningsHistory": {"earningsHistory", func(tk *Ticker, m models.EPSMethodology) (float64, string, error) {
			r, err := tk.EarningsHistoryWithMethodology(m)
			if err != nil {
				return 0, "", err
			}
			return r.History[0].EPSActual, r.History[0].Methodology, nil
		}, map[models.EPSMethodology]float64{"": 4.14, "gaap": 5.16, "normalized": 4.14}},
	}
	suffix := map[models.EPSMethodology]string{"": "", "gaap": "Gaap", "normalized": "NonGaap"}
	// Yahoo labels every variant "nongaap"; the requested methodology wins.
	label := map[models.EPSMethodology]string{"": "normalized", "gaap": "gaap", "normalized": "normalized"}

	for name, c := range calls {
		for _, m := range []models.EPSMethodology{models.EPSMethodologyDefault, models.EPSMethodologyGAAP, models.EPSMethodologyNormalized} {
			t.Run(fmt.Sprintf("%s/%q", name, m), func(t *testing.T) {
				tkr, fake := newMSFTAnalysisTicker(t)
				got, lbl, err := c.fn(tkr, m)
				if err != nil {
					t.Fatal(err)
				}
				wantModule := c.base + suffix[m]
				if r := fake.requested(); !reflect.DeepEqual(r, []string{wantModule}) {
					t.Fatalf("requests = %v, want [%s]", r, wantModule)
				}
				assertFloat(t, "first value", got, c.want[m])
				if lbl != label[m] {
					t.Fatalf("Methodology = %q, want %q", lbl, label[m])
				}
			})
		}
	}
}

// Port of Python yfinance test_eps_methodology_modules.
func TestEPSMethodologyModulesAndCaching(t *testing.T) {
	fake := newFakeQuoteSummary()
	for module, eps := range map[string]float64{"earningsTrend": 1, "earningsTrendGaap": 2, "earningsTrendNonGaap": 3} {
		fake.set(module, 200, quoteSummaryBody(module,
			fmt.Sprintf(`{"trend":[{"period":"0q","earningsEstimate":{"avg":{"raw":%v}}}],"defaultMethodology":"nongaap"}`, eps)))
	}
	fake.set("earningsHistory", 200, quoteSummaryBody("earningsHistory",
		`{"history":[{"quarter":{"raw":1782777600,"fmt":"2026-06-30"},"epsActual":{"raw":1.0}}],"defaultMethodology":"nongaap"}`))
	tkr := newFakeAnalysisTicker(t, fake)

	dflt, err := tkr.EarningsEstimate()
	if err != nil {
		t.Fatal(err)
	}
	gaap, err := tkr.EarningsEstimateWithMethodology("gaap")
	if err != nil {
		t.Fatal(err)
	}
	normalized, err := tkr.EarningsEstimateWithMethodology("normalized")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tkr.EarningsEstimateWithMethodology("normalized"); err != nil {
		t.Fatal(err)
	}
	hist, err := tkr.EarningsHistory()
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"earningsTrend", "earningsTrendGaap", "earningsTrendNonGaap", "earningsHistory"}
	if got := fake.requested(); !reflect.DeepEqual(got, want) {
		t.Fatalf("requests = %v, want %v", got, want)
	}
	assertFloat(t, "default avg", dflt[0].Avg, 1)
	assertFloat(t, "gaap avg", gaap[0].Avg, 2)
	assertFloat(t, "normalized avg", normalized[0].Avg, 3)
	if dflt[0].Methodology != "normalized" || gaap[0].Methodology != "gaap" ||
		normalized[0].Methodology != "normalized" || hist.History[0].Methodology != "normalized" {
		t.Fatalf("labels = %q %q %q %q", dflt[0].Methodology, gaap[0].Methodology,
			normalized[0].Methodology, hist.History[0].Methodology)
	}
}

func TestEPSMethodologyCaseInsensitiveSharesCache(t *testing.T) {
	tkr, fake := newMSFTAnalysisTicker(t)

	first, err := tkr.EPSTrendWithMethodology("gaap")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []models.EPSMethodology{"GAAP", "Gaap", models.EPSMethodologyGAAP} {
		again, err := tkr.EPSTrendWithMethodology(m)
		if err != nil {
			t.Fatalf("%q: %v", m, err)
		}
		if &again[0] != &first[0] {
			t.Fatalf("%q: not served from the gaap cache", m)
		}
		if again[0].Methodology != "gaap" {
			t.Fatalf("%q: Methodology = %q, want normalized value gaap", m, again[0].Methodology)
		}
	}

	h1, err := tkr.EarningsHistoryWithMethodology("Normalized")
	if err != nil {
		t.Fatal(err)
	}
	h2, err := tkr.EarningsHistoryWithMethodology("NORMALIZED")
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 || h1.History[0].Methodology != "normalized" {
		t.Fatalf("history not cached case-insensitively: %p %p %q", h1, h2, h1.History[0].Methodology)
	}

	want := []string{"earningsTrendGaap", "earningsHistoryNonGaap"}
	if got := fake.requested(); !reflect.DeepEqual(got, want) {
		t.Fatalf("requests = %v, want %v", got, want)
	}
}

func TestEPSMethodologyInvalidFailsBeforeRequest(t *testing.T) {
	tkr, fake := newMSFTAnalysisTicker(t)
	calls := map[string]func(models.EPSMethodology) error{
		"EarningsEstimate": func(m models.EPSMethodology) error {
			r, err := tkr.EarningsEstimateWithMethodology(m)
			if r != nil {
				return errors.New("non-nil result")
			}
			return err
		},
		"EPSTrend": func(m models.EPSMethodology) error {
			r, err := tkr.EPSTrendWithMethodology(m)
			if r != nil {
				return errors.New("non-nil result")
			}
			return err
		},
		"EPSRevisions": func(m models.EPSMethodology) error {
			r, err := tkr.EPSRevisionsWithMethodology(m)
			if r != nil {
				return errors.New("non-nil result")
			}
			return err
		},
		"EarningsHistory": func(m models.EPSMethodology) error {
			r, err := tkr.EarningsHistoryWithMethodology(m)
			if r != nil {
				return errors.New("non-nil result")
			}
			return err
		},
	}
	for name, fn := range calls {
		for _, m := range []models.EPSMethodology{"ifrs", "nongaap", "GAAP ", "non-gaap"} {
			err := fn(m)
			if !errors.Is(err, models.ErrInvalidEPSMethodology) {
				t.Fatalf("%s(%q) error = %v, want ErrInvalidEPSMethodology", name, m, err)
			}
		}
	}
	if got := fake.requested(); len(got) != 0 {
		t.Fatalf("invalid methodology made requests: %v", got)
	}
	if tkr.analysisCache != nil {
		t.Fatal("invalid methodology touched the analysis cache")
	}
}

func TestEPSMethodologyLabel(t *testing.T) {
	cases := []struct {
		name   string
		m      models.EPSMethodology
		module map[string]interface{}
		want   string
	}{
		{"default nongaap", "", map[string]interface{}{"defaultMethodology": "nongaap"}, "normalized"},
		{"default gaap", "", map[string]interface{}{"defaultMethodology": "gaap"}, "gaap"},
		{"default missing", "", map[string]interface{}{}, ""},
		{"default nil module", "", nil, ""},
		{"default unknown passes through", "", map[string]interface{}{"defaultMethodology": "ifrs"}, "ifrs"},
		{"default non-string", "", map[string]interface{}{"defaultMethodology": 1.0}, ""},
		{"requested gaap wins", "gaap", map[string]interface{}{"defaultMethodology": "nongaap"}, "gaap"},
		{"requested normalized wins", "normalized", map[string]interface{}{"defaultMethodology": "gaap"}, "normalized"},
		{"requested without default", "gaap", map[string]interface{}{}, "gaap"},
	}
	for _, tc := range cases {
		if got := epsMethodologyLabel(tc.m, tc.module); got != tc.want {
			t.Errorf("%s: label = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestEPSMethodologyFromDefaultMethodologyEndToEnd(t *testing.T) {
	cases := []struct {
		name    string
		extra   string // appended module fields
		want    string
		request models.EPSMethodology
	}{
		{"nongaap", `,"defaultMethodology":"nongaap"`, "normalized", ""},
		{"gaap", `,"defaultMethodology":"gaap"`, "gaap", ""},
		{"missing", ``, "", ""},
		{"unknown", `,"defaultMethodology":"ifrs"`, "ifrs", ""},
		{"null", `,"defaultMethodology":null`, "", ""},
		{"requested overrides", `,"defaultMethodology":"gaap"`, "normalized", "normalized"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := newFakeQuoteSummary()
			trendModule := epsModuleName("earningsTrend", tc.request)
			histModule := epsModuleName("earningsHistory", tc.request)
			fake.set(trendModule, 200, quoteSummaryBody(trendModule, `{"trend":[{"period":"0q",
				"earningsEstimate":{"avg":{"raw":1.5},"earningsCurrency":"USD"},
				"revenueEstimate":{"avg":{"raw":100}},
				"epsTrend":{"current":{"raw":1.5}},
				"epsRevisions":{"upLast7days":{"raw":2}}}]`+tc.extra+`}`))
			fake.set(histModule, 200, quoteSummaryBody(histModule,
				`{"history":[{"period":"-1q","epsActual":{"raw":1.2}}]`+tc.extra+`}`))
			tkr := newFakeAnalysisTicker(t, fake)

			est, err := tkr.EarningsEstimateWithMethodology(tc.request)
			if err != nil {
				t.Fatal(err)
			}
			trend, err := tkr.EPSTrendWithMethodology(tc.request)
			if err != nil {
				t.Fatal(err)
			}
			rev, err := tkr.EPSRevisionsWithMethodology(tc.request)
			if err != nil {
				t.Fatal(err)
			}
			hist, err := tkr.EarningsHistoryWithMethodology(tc.request)
			if err != nil {
				t.Fatal(err)
			}
			got := []string{est[0].Methodology, trend[0].Methodology, rev[0].Methodology, hist.History[0].Methodology}
			for i, g := range got {
				if g != tc.want {
					t.Fatalf("table %d Methodology = %q, want %q (all: %v)", i, g, tc.want, got)
				}
			}
		})
	}
}

func TestEPSMethodologyEmptyTablesHaveNoMethodology(t *testing.T) {
	fake := newFakeQuoteSummary()
	// Rows whose sub-objects carry no data (only a currency or empty dicts),
	// as Yahoo returns for tickers without analyst coverage.
	fake.set("earningsTrendGaap", 200, quoteSummaryBody("earningsTrendGaap", `{"trend":[
		{"period":"0q","earningsEstimate":{"earningsCurrency":"USD","avg":{}},
		 "epsTrend":{},"epsRevisions":{"downLast90days":{},"epsRevisionsCurrency":"USD"}},
		{"period":"+1q","earningsEstimate":{},"epsTrend":{"current":{}},"epsRevisions":{}}
	],"defaultMethodology":"gaap"}`))
	fake.set("earningsTrend", 200, quoteSummaryBody("earningsTrend", `{"trend":[],"defaultMethodology":"nongaap"}`))
	fake.set("earningsHistoryGaap", 200, quoteSummaryBody("earningsHistoryGaap", `{"history":[],"defaultMethodology":"gaap"}`))
	tkr := newFakeAnalysisTicker(t, fake)

	est, err := tkr.EarningsEstimateWithMethodology("gaap")
	if err != nil {
		t.Fatal(err)
	}
	trend, err := tkr.EPSTrendWithMethodology("gaap")
	if err != nil {
		t.Fatal(err)
	}
	rev, err := tkr.EPSRevisionsWithMethodology("gaap")
	if err != nil {
		t.Fatal(err)
	}
	if len(est) != 2 || len(trend) != 2 || len(rev) != 2 {
		t.Fatalf("row counts = %d %d %d, want 2", len(est), len(trend), len(rev))
	}
	for i := 0; i < 2; i++ {
		if est[i].Methodology != "" || trend[i].Methodology != "" || rev[i].Methodology != "" {
			t.Fatalf("row %d of a data-less table has a methodology: %q %q %q",
				i, est[i].Methodology, trend[i].Methodology, rev[i].Methodology)
		}
	}
	if est[0].Currency != "USD" {
		t.Fatalf("currency = %q, want USD", est[0].Currency)
	}

	empty, err := tkr.EarningsEstimate()
	if err != nil {
		t.Fatal(err)
	}
	if empty == nil || len(empty) != 0 {
		t.Fatalf("empty trend = %#v, want empty non-nil slice", empty)
	}

	hist, err := tkr.EarningsHistoryWithMethodology("gaap")
	if err != nil {
		t.Fatal(err)
	}
	if hist == nil || len(hist.History) != 0 {
		t.Fatalf("empty history = %#v", hist)
	}
}

func TestEPSMethodologyPartialDataTableGetsMethodology(t *testing.T) {
	fake := newFakeQuoteSummary()
	fake.set("earningsTrendNonGaap", 200, quoteSummaryBody("earningsTrendNonGaap", `{"trend":[
		{"period":"0q","earningsEstimate":{"avg":{}}},
		{"period":"+1q","earningsEstimate":{"avg":{"raw":2.5}}}
	]}`))
	tkr := newFakeAnalysisTicker(t, fake)
	est, err := tkr.EarningsEstimateWithMethodology("normalized")
	if err != nil {
		t.Fatal(err)
	}
	// As in pandas, the column is added to every row once any row has data.
	if est[0].Methodology != "normalized" || est[1].Methodology != "normalized" {
		t.Fatalf("labels = %q %q", est[0].Methodology, est[1].Methodology)
	}
}

func TestRevenueAndGrowthUseDefaultTrend(t *testing.T) {
	fake := newFakeQuoteSummary()
	fake.set("earningsTrend", 200, quoteSummaryBody("earningsTrend", `{"trend":[
		{"period":"0q","growth":{"raw":0.11},"earningsEstimate":{"avg":{"raw":1}},
		 "revenueEstimate":{"avg":{"raw":1000},"revenueCurrency":"USD"}}
	],"defaultMethodology":"nongaap"}`))
	fake.set("earningsTrendGaap", 200, quoteSummaryBody("earningsTrendGaap", `{"trend":[
		{"period":"0q","growth":{"raw":0.99},"earningsEstimate":{"avg":{"raw":2}},
		 "revenueEstimate":{"avg":{"raw":9999},"revenueCurrency":"USD"}}
	],"defaultMethodology":"nongaap"}`))
	fake.set("industryTrend,sectorTrend,indexTrend", 200,
		`{"quoteSummary":{"result":[{"indexTrend":{"estimates":[{"period":"0q","growth":0.27}]}}]}}`)
	tkr := newFakeAnalysisTicker(t, fake)

	// Warm only the GAAP trend first: revenue/growth must not reuse it.
	if _, err := tkr.EarningsEstimateWithMethodology("gaap"); err != nil {
		t.Fatal(err)
	}

	rev, err := tkr.RevenueEstimate()
	if err != nil {
		t.Fatal(err)
	}
	if len(rev) != 1 || rev[0].Avg != 1000 || rev[0].Currency != "USD" {
		t.Fatalf("revenue = %+v, want default-trend values", rev)
	}

	growth, err := tkr.GrowthEstimates()
	if err != nil {
		t.Fatal(err)
	}
	if len(growth) != 1 || growth[0].StockGrowth == nil || *growth[0].StockGrowth != 0.11 ||
		growth[0].IndexGrowth == nil || *growth[0].IndexGrowth != 0.27 {
		t.Fatalf("growth = %+v, want default-trend stock growth", growth)
	}

	// The default EPS table reuses the default trend module fetched for revenue.
	if _, err := tkr.EarningsEstimate(); err != nil {
		t.Fatal(err)
	}
	want := []string{"earningsTrendGaap", "earningsTrend", "industryTrend,sectorTrend,indexTrend"}
	if got := fake.requested(); !reflect.DeepEqual(got, want) {
		t.Fatalf("requests = %v, want %v", got, want)
	}
}

func TestEPSMethodologyCacheIsolationAndRequestCounts(t *testing.T) {
	tkr, fake := newMSFTAnalysisTicker(t)
	methodologies := []models.EPSMethodology{"", "gaap", "normalized"}

	for round := 0; round < 3; round++ {
		for _, m := range methodologies {
			est, err := tkr.EarningsEstimateWithMethodology(m)
			if err != nil {
				t.Fatal(err)
			}
			trend, err := tkr.EPSTrendWithMethodology(m)
			if err != nil {
				t.Fatal(err)
			}
			rev, err := tkr.EPSRevisionsWithMethodology(m)
			if err != nil {
				t.Fatal(err)
			}
			hist, err := tkr.EarningsHistoryWithMethodology(m)
			if err != nil {
				t.Fatal(err)
			}
			if len(est) != 4 || len(trend) != 4 || len(rev) != 4 || len(hist.History) != 4 {
				t.Fatalf("%q: unexpected row counts", m)
			}
		}
		if _, err := tkr.RevenueEstimate(); err != nil {
			t.Fatal(err)
		}
		if _, err := tkr.GrowthEstimates(); err != nil {
			t.Fatal(err)
		}
	}

	for _, module := range []string{
		"earningsTrend", "earningsTrendGaap", "earningsTrendNonGaap",
		"earningsHistory", "earningsHistoryGaap", "earningsHistoryNonGaap",
		"industryTrend,sectorTrend,indexTrend",
	} {
		if n := fake.count(module); n != 1 {
			t.Fatalf("%s requested %d times, want 1 (all: %v)", module, n, fake.requested())
		}
	}

	gaap, _ := tkr.EarningsEstimateWithMethodology("gaap")
	dflt, _ := tkr.EarningsEstimate()
	if gaap[0].Avg == dflt[0].Avg || gaap[0].Methodology != "gaap" || dflt[0].Methodology != "normalized" {
		t.Fatalf("caches not isolated: gaap=%+v default=%+v", gaap[0], dflt[0])
	}
}

func TestEPSMethodologyClearCache(t *testing.T) {
	tkr, fake := newMSFTAnalysisTicker(t)

	if _, err := tkr.EarningsEstimateWithMethodology("gaap"); err != nil {
		t.Fatal(err)
	}
	if _, err := tkr.EarningsHistoryWithMethodology("normalized"); err != nil {
		t.Fatal(err)
	}
	tkr.ClearCache()
	if tkr.analysisCache != nil {
		t.Fatal("ClearCache did not reset the analysis cache")
	}
	if _, err := tkr.EarningsEstimateWithMethodology("gaap"); err != nil {
		t.Fatal(err)
	}
	if _, err := tkr.EarningsHistoryWithMethodology("normalized"); err != nil {
		t.Fatal(err)
	}
	if fake.count("earningsTrendGaap") != 2 || fake.count("earningsHistoryNonGaap") != 2 {
		t.Fatalf("ClearCache did not force refetch: %v", fake.requested())
	}
}

func TestEPSMethodologyClearCacheDuringFetchDiscardsResult(t *testing.T) {
	tkr, fake := newMSFTAnalysisTicker(t)
	release := make(chan struct{})
	fake.mu.Lock()
	fake.block["earningsTrendGaap"] = release
	fake.mu.Unlock()

	done := make(chan error, 1)
	go func() {
		_, err := tkr.EPSTrendWithMethodology("gaap")
		done <- err
	}()
	for fake.count("earningsTrendGaap") == 0 {
		time.Sleep(time.Millisecond)
	}
	tkr.ClearCache()
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}

	tkr.mu.RLock()
	stale := tkr.analysisCache != nil
	tkr.mu.RUnlock()
	if stale {
		t.Fatal("result fetched before ClearCache was cached after it")
	}
}

func TestEPSMethodologyMissingModule(t *testing.T) {
	fake := newFakeQuoteSummary()
	// Yahoo answers but without the requested module.
	fake.set("earningsTrendGaap", 200, `{"quoteSummary":{"result":[{}],"error":null}}`)
	fake.set("earningsHistoryGaap", 200, `{"quoteSummary":{"result":[{}],"error":null}}`)
	tkr := newFakeAnalysisTicker(t, fake)

	for i := 0; i < 2; i++ {
		est, err := tkr.EarningsEstimateWithMethodology("gaap")
		if err != nil || est != nil {
			t.Fatalf("missing trend module: est=%v err=%v, want nil, nil", est, err)
		}
		trend, err := tkr.EPSTrendWithMethodology("gaap")
		if err != nil || trend != nil {
			t.Fatalf("missing trend module: trend=%v err=%v", trend, err)
		}
	}
	// The (empty) module is cached: one request for all trend tables.
	if n := fake.count("earningsTrendGaap"); n != 1 {
		t.Fatalf("earningsTrendGaap requested %d times, want 1", n)
	}

	for i := 0; i < 2; i++ {
		hist, err := tkr.EarningsHistoryWithMethodology("gaap")
		if err == nil || hist != nil || !strings.Contains(err.Error(), "earningsHistoryGaap not found") {
			t.Fatalf("missing history module: hist=%v err=%v", hist, err)
		}
	}
	// Errors are not cached.
	if n := fake.count("earningsHistoryGaap"); n != 2 {
		t.Fatalf("earningsHistoryGaap requested %d times, want 2", n)
	}
}

func TestEPSMethodologyMalformedPayloads(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		body    string
		wantErr string
	}{
		{"http error", 500, `{}`, "status 500"},
		{"invalid json", 200, `{"quoteSummary":`, "failed to parse response"},
		{"no quoteSummary", 200, `{"finance":{"error":{"code":"Not Found"}}}`, "missing quoteSummary"},
		{"null result", 200, `{"quoteSummary":{"result":null,"error":{"code":"Not Found"}}}`, "missing result"},
		{"empty result", 200, `{"quoteSummary":{"result":[]}}`, "missing result"},
		{"result not object", 200, `{"quoteSummary":{"result":["x"]}}`, "result is not an object"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := newFakeQuoteSummary()
			fake.set("earningsTrendNonGaap", tc.status, tc.body)
			fake.set("earningsHistoryNonGaap", tc.status, tc.body)
			tkr := newFakeAnalysisTicker(t, fake)

			if _, err := tkr.EPSRevisionsWithMethodology("normalized"); err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("EPSRevisions error = %v, want %q", err, tc.wantErr)
			}
			if _, err := tkr.EarningsHistoryWithMethodology("normalized"); err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("EarningsHistory error = %v, want %q", err, tc.wantErr)
			}
			// Failures are not cached: a retry requests again.
			_, _ = tkr.EPSRevisionsWithMethodology("normalized")
			if n := fake.count("earningsTrendNonGaap"); n != 2 {
				t.Fatalf("earningsTrendNonGaap requested %d times after failure, want 2", n)
			}
		})
	}
}

func TestEPSMethodologyMalformedModuleContents(t *testing.T) {
	fake := newFakeQuoteSummary()
	fake.set("earningsTrendGaap", 200, quoteSummaryBody("earningsTrendGaap",
		`{"trend":{"period":"0q"},"defaultMethodology":"gaap"}`))
	fake.set("earningsTrendNonGaap", 200, quoteSummaryBody("earningsTrendNonGaap",
		`{"trend":["bad",null,{"period":"0q","earningsEstimate":"bad"},
		  {"period":"+1q","earningsEstimate":{"avg":{"raw":3.25}},"epsTrend":null}],"defaultMethodology":"gaap"}`))
	fake.set("earningsTrend", 200, quoteSummaryBody("earningsTrend", `"not an object"`))
	fake.set("earningsHistoryGaap", 200, quoteSummaryBody("earningsHistoryGaap", `{"history":"bad"}`))
	fake.set("earningsHistoryNonGaap", 200, quoteSummaryBody("earningsHistoryNonGaap",
		`{"history":["bad",{"period":"-1q","quarter":"bad","epsActual":{"raw":"x"},"epsEstimate":2.5}],"defaultMethodology":"gaap"}`))
	tkr := newFakeAnalysisTicker(t, fake)

	if est, err := tkr.EarningsEstimateWithMethodology("gaap"); err != nil || est != nil {
		t.Fatalf("trend not a list: %v %v", est, err)
	}
	if est, err := tkr.EarningsEstimate(); err != nil || est != nil {
		t.Fatalf("module not an object: %v %v", est, err)
	}

	est, err := tkr.EarningsEstimateWithMethodology("normalized")
	if err != nil {
		t.Fatal(err)
	}
	if len(est) != 1 || est[0].Period != "+1q" || est[0].Avg != 3.25 || est[0].Methodology != "normalized" {
		t.Fatalf("malformed rows not skipped: %+v", est)
	}
	trend, err := tkr.EPSTrendWithMethodology("normalized")
	if err != nil || trend == nil || len(trend) != 0 {
		t.Fatalf("trend without epsTrend rows = %#v, %v; want empty", trend, err)
	}

	if _, err := tkr.EarningsHistoryWithMethodology("gaap"); err == nil || !strings.Contains(err.Error(), "history data not found") {
		t.Fatalf("history not a list: err = %v", err)
	}
	hist, err := tkr.EarningsHistoryWithMethodology("normalized")
	if err != nil {
		t.Fatal(err)
	}
	if len(hist.History) != 1 {
		t.Fatalf("history rows = %d, want 1", len(hist.History))
	}
	h := hist.History[0]
	if h.Period != "-1q" || !h.Quarter.IsZero() || h.EPSActual != 0 || h.EPSEstimate != 2.5 || h.Methodology != "normalized" {
		t.Fatalf("history row = %+v", h)
	}
}

func TestEPSMethodologyConcurrentAccess(t *testing.T) {
	tkr, fake := newMSFTAnalysisTicker(t)
	methodologies := []models.EPSMethodology{"", "gaap", "GAAP", "normalized", "Normalized"}
	label := map[models.EPSMethodology]string{"": "normalized", "gaap": "gaap", "GAAP": "gaap", "normalized": "normalized", "Normalized": "normalized"}

	const workers = 16
	var wg sync.WaitGroup
	errs := make(chan error, workers*len(methodologies)*4+workers)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for _, m := range methodologies {
				est, err := tkr.EarningsEstimateWithMethodology(m)
				if err == nil && est[0].Methodology != label[m] {
					err = fmt.Errorf("estimate %q label %q", m, est[0].Methodology)
				}
				errs <- err
				trend, err := tkr.EPSTrendWithMethodology(m)
				if err == nil && trend[0].Methodology != label[m] {
					err = fmt.Errorf("trend %q label %q", m, trend[0].Methodology)
				}
				errs <- err
				rev, err := tkr.EPSRevisionsWithMethodology(m)
				if err == nil && rev[0].Methodology != label[m] {
					err = fmt.Errorf("revisions %q label %q", m, rev[0].Methodology)
				}
				errs <- err
				hist, err := tkr.EarningsHistoryWithMethodology(m)
				if err == nil && hist.History[0].Methodology != label[m] {
					err = fmt.Errorf("history %q label %q", m, hist.History[0].Methodology)
				}
				errs <- err
			}
			if w%4 == 0 {
				tkr.ClearCache()
			}
			_, err := tkr.GrowthEstimates()
			errs <- err
		}(w)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(fake.requested()) == 0 {
		t.Fatal("no requests recorded")
	}
}

func TestQuoteSummaryModulesIncludeEPSMethodologyVariants(t *testing.T) {
	valid := make(map[string]bool, len(endpoints.QuoteSummaryModules))
	for _, m := range endpoints.QuoteSummaryModules {
		valid[m] = true
	}
	for _, base := range []string{"earningsTrend", "earningsHistory"} {
		for m := range epsMethodologyModuleSuffix {
			if name := epsModuleName(base, m); !valid[name] {
				t.Fatalf("%s missing from endpoints.QuoteSummaryModules", name)
			}
		}
	}
}
