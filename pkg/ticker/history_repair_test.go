package ticker

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/client"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

// fixtureRow is one row of an upstream price fixture shared with pkg/repair.
type fixtureRow struct {
	ts                     int64
	open, high, low, close float64
	adjClose               float64
	volume                 int64
}

func loadFixtureRows(t *testing.T, name string) []fixtureRow {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "repair", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	pf := func(s string) float64 {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	out := make([]fixtureRow, 0, len(rows)-1)
	for _, r := range rows[1:] {
		d, err := time.Parse("2006-01-02 15:04:05-07:00", r[0])
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, fixtureRow{d.Unix(), pf(r[1]), pf(r[2]), pf(r[3]), pf(r[4]), pf(r[5]), int64(pf(r[6]))})
	}
	return out
}

// chartJSON renders fixture rows as a Yahoo chart API response.
func chartJSON(t *testing.T, rows []fixtureRow, granularity string) string {
	t.Helper()
	var res models.ChartResult
	res.Meta = models.ChartMeta{
		Symbol: "SOXS", Currency: "USD", ExchangeTimezoneName: "America/New_York",
		InstrumentType: "ETF", DataGranularity: granularity,
	}
	q := models.ChartQuote{}
	adj := models.ChartAdjClose{}
	for _, r := range rows {
		r := r
		res.Timestamp = append(res.Timestamp, r.ts)
		q.Open = append(q.Open, &r.open)
		q.High = append(q.High, &r.high)
		q.Low = append(q.Low, &r.low)
		q.Close = append(q.Close, &r.close)
		q.Volume = append(q.Volume, &r.volume)
		adj.AdjClose = append(adj.AdjClose, &r.adjClose)
	}
	res.Indicators.Quote = []models.ChartQuote{q}
	res.Indicators.AdjClose = []models.ChartAdjClose{adj}
	var resp models.ChartResponse
	resp.Chart.Result = []models.ChartResult{res}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// soxsTicker returns a Ticker whose chart requests are served from the SOXS
// level-shift fixtures, recording every request. hourlyErr, when set, is
// returned for 1h requests.
func soxsTicker(t *testing.T, hourlyErr error) (*Ticker, *[]url.Values) {
	t.Helper()
	daily := chartJSON(t, loadFixtureRows(t, "SOXS-1d-bad-level-shift.csv"), "1d")
	hourly := chartJSON(t, loadFixtureRows(t, "SOXS-1h-bad-level-shift.csv"), "1h")
	tkr, err := New("SOXS")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(tkr.Close)
	var calls []url.Values
	tkr.chartBodyFetcher = func(p url.Values) (string, error) {
		calls = append(calls, p)
		switch p.Get("interval") {
		case "1d":
			return daily, nil
		case "1h":
			if hourlyErr != nil {
				return "", hourlyErr
			}
			return hourly, nil
		}
		return "", errors.New("unexpected interval " + p.Get("interval"))
	}
	// One week after the fixture's last row (2026-06-05).
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	tkr.clock = func() time.Time { return now }
	return tkr, &calls
}

func soxsHistoryParams() models.HistoryParams {
	start := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 6, 0, 0, 0, 0, time.UTC)
	return models.HistoryParams{Start: &start, End: &end, Interval: "1d", Actions: true, Repair: true}
}

// TestHistoryRepairFetchesHourlyOnce checks History wires the level-shift
// repair: one raw 1h request for the 729-day window (exchange-timezone
// midnights), no recursive repair, metadata kept from the 1d request, and the
// SOXS 15x shift fixed.
func TestHistoryRepairFetchesHourlyOnce(t *testing.T) {
	tkr, calls := soxsTicker(t, nil)
	bars, err := tkr.History(soxsHistoryParams())
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(*calls) != 2 {
		t.Fatalf("chart requests = %d, want 2 (1d + one 1h)", len(*calls))
	}
	h := (*calls)[1]
	ny, _ := time.LoadLocation("America/New_York")
	wantStart := time.Date(2024, 6, 13, 0, 0, 0, 0, ny).Unix()
	wantEnd := time.Date(2026, 6, 13, 0, 0, 0, 0, ny).Unix()
	if h.Get("interval") != "1h" || h.Get("includePrePost") != "false" || h.Get("range") != "" ||
		h.Get("period1") != strconv.FormatInt(wantStart, 10) || h.Get("period2") != strconv.FormatInt(wantEnd, 10) {
		t.Fatalf("1h request params = %v", h)
	}
	if meta := tkr.GetHistoryMetadata(); meta == nil || meta.DataGranularity != "1d" {
		t.Fatalf("history metadata overwritten by 1h request: %+v", meta)
	}

	if len(bars) != 19 {
		t.Fatalf("bars = %d", len(bars))
	}
	// First row: 1297.5 / 15 = 86.5, volume 2004827 * 15.
	if math.Abs(bars[0].Open-86.5) > 1e-9 || bars[0].Volume != 30072405 || !bars[0].Repaired {
		t.Fatalf("first bar not repaired: %+v", bars[0])
	}
	if bars[10].Repaired || bars[18].Repaired {
		t.Fatal("bars after the shift were repaired")
	}
}

func TestHistoryRepairLevelShiftDisabled(t *testing.T) {
	tkr, calls := soxsTicker(t, nil)
	params := soxsHistoryParams()
	opts := models.DefaultRepairOptions()
	opts.FixLevelShifts = false
	params.RepairOptions = &opts
	bars, err := tkr.History(params)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(*calls) != 1 {
		t.Fatalf("chart requests = %d, want 1", len(*calls))
	}
	if bars[0].Open != 1297.5 {
		t.Fatalf("bar repaired with FixLevelShifts=false: %+v", bars[0])
	}
}

func TestHistoryWithoutRepairDoesNotFetchHourly(t *testing.T) {
	tkr, calls := soxsTicker(t, nil)
	params := soxsHistoryParams()
	params.Repair = false
	if _, err := tkr.History(params); err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(*calls) != 1 {
		t.Fatalf("chart requests = %d, want 1", len(*calls))
	}
}

func TestHistoryRepairHourlyRateLimit(t *testing.T) {
	tkr, calls := soxsTicker(t, client.WrapRateLimitError())
	bars, err := tkr.History(soxsHistoryParams())
	if !client.IsRateLimitError(err) {
		t.Fatalf("err = %v, want rate limit", err)
	}
	if bars != nil || len(*calls) != 2 {
		t.Fatalf("bars=%v calls=%d", bars, len(*calls))
	}
}

func TestHistoryRepairHourlyErrorLeavesDataUnrepaired(t *testing.T) {
	tkr, calls := soxsTicker(t, client.HTTPStatusToError(500, "boom"))
	bars, err := tkr.History(soxsHistoryParams())
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(*calls) != 2 {
		t.Fatalf("chart requests = %d, want 2", len(*calls))
	}
	if bars[0].Open != 1297.5 || bars[0].Repaired {
		t.Fatalf("bar changed despite failed 1h fetch: %+v", bars[0])
	}
}

func TestRepairHourlyFetcherReturnsRawBars(t *testing.T) {
	tkr, calls := soxsTicker(t, nil)
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	bars, err := tkr.repairHourlyFetcher()(start, start.AddDate(0, 2, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 1 || len(bars) != 133 {
		t.Fatalf("calls=%d bars=%d", len(*calls), len(bars))
	}
	if bars[0].Open != 8.649999618530273 || bars[0].Repaired {
		t.Fatalf("first hourly bar = %+v", bars[0])
	}
	if tkr.GetHistoryMetadata() != nil {
		t.Fatal("hourly fetch cached metadata")
	}
}
