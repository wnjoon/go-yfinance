package repair

import (
	"encoding/csv"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

// loadBarsCSV reads an upstream python-yfinance price fixture
// (Datetime,Open,High,Low,Close,Adj Close,Volume[,Dividends,Stock Splits]).
// Missing action columns, as in intraday fixtures, read as zero.
func loadBarsCSV(t *testing.T, name string) []models.Bar {
	t.Helper()

	f, err := os.Open(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("open fixture %s: %v", name, err)
	}
	defer func() { _ = f.Close() }()

	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	if len(rows) < 2 {
		t.Fatalf("fixture %s has no data rows", name)
	}

	parseF := func(s string) float64 {
		if s == "" {
			// pandas writes NaN as an empty field.
			return math.NaN()
		}
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			t.Fatalf("fixture %s: bad float %q: %v", name, s, err)
		}
		return v
	}

	bars := make([]models.Bar, 0, len(rows)-1)
	for _, row := range rows[1:] {
		date, err := time.Parse("2006-01-02 15:04:05-07:00", row[0])
		if err != nil {
			t.Fatalf("fixture %s: bad datetime %q: %v", name, row[0], err)
		}
		bars = append(bars, models.Bar{
			Date:     date,
			Open:     parseF(row[1]),
			High:     parseF(row[2]),
			Low:      parseF(row[3]),
			Close:    parseF(row[4]),
			AdjClose: parseF(row[5]),
			// pandas finishes repair with round().astype(int64); Go's model is
			// already integral, so preserve that final semantic at ingestion.
			Volume:    int64(math.RoundToEven(parseF(row[6]))),
			Dividends: optionalColumn(row, 7, parseF),
			Splits:    optionalColumn(row, 8, parseF),
		})
	}
	return bars
}

// optionalColumn parses row[i], or returns 0 when the column is absent.
func optionalColumn(row []string, i int, parse func(string) float64) float64 {
	if i >= len(row) {
		return 0
	}
	return parse(row[i])
}

// TestRepairUnitMixupASAIGolden ports upstream test_repair_100x_random_1h:
// one hourly ASAI.L bar has Low/Close/Adj Close 100x too small while
// Open/High are correct. The repair must fix only the corrupt cells,
// recalculate Low as min(Open, Close), and keep prices in pence.
func TestRepairUnitMixupASAIGolden(t *testing.T) {
	bad := loadBarsCSV(t, "ASAI-L-1h-bad-unit.csv")
	fixed := loadBarsCSV(t, "ASAI-L-1h-bad-unit-fixed.csv")
	if len(bad) != len(fixed) {
		t.Fatalf("fixture length mismatch: %d vs %d", len(bad), len(fixed))
	}

	opts := DefaultOptions()
	opts.Ticker = "ASAI.L"
	opts.Interval = "1h"
	opts.Currency = "GBp"
	opts.Timezone = "Europe/London"
	repairer := New(opts)

	result, err := repairer.Repair(bad)
	if err != nil {
		t.Fatalf("Repair returned error: %v", err)
	}

	const rtol = 1e-7
	repairedCount := 0
	for i := range result {
		if result[i].Repaired {
			repairedCount++
		}
		for _, c := range []struct {
			name      string
			got, want float64
		}{
			{"Open", result[i].Open, fixed[i].Open},
			{"High", result[i].High, fixed[i].High},
			{"Low", result[i].Low, fixed[i].Low},
			{"Close", result[i].Close, fixed[i].Close},
		} {
			if !closeTo(c.got, c.want, rtol) {
				t.Errorf("bar %d (%s) %s: got %v, want %v",
					i, result[i].Date.Format("2006-01-02 15:04"), c.name, c.got, c.want)
			}
		}
	}

	if repairedCount != 1 {
		t.Errorf("expected exactly 1 repaired bar, got %d", repairedCount)
	}
}

// TestRepairStockSplitsNFEGolden ports the Python yfinance 1.8.0 NFE fixture
// (upstream commits aec8332 and 5b63983). A 1:50 reverse split was not
// applied to the older rows, the split-day row has no prices, and volume
// genuinely spiked for a few days after the split, so the volume ratio alone
// (about 5.8x) misses the ~10.8x threshold. The range is kept only by the
// >15-standard-deviation fallback.
func TestRepairStockSplitsNFEGolden(t *testing.T) {
	bad := loadBarsCSV(t, "NFE-1d-bad-stock-split.csv")
	fixed := loadBarsCSV(t, "NFE-1d-bad-stock-split-fixed.csv")
	opts := DefaultOptions()
	opts.Interval = "1d"
	got := New(opts).repairStockSplits(bad)
	assertSplitGolden(t, "NFE", got, fixed)

	// Upstream marks every row before the split repaired; the NaN split-day
	// row is set aside before repair and restored untouched.
	for i := range got {
		want := i < 14
		if got[i].Repaired != want {
			t.Errorf("row=%d (%s) Repaired=%v, want %v", i, got[i].Date.Format("2006-01-02"), got[i].Repaired, want)
		}
	}
	if !math.IsNaN(got[14].Close) || got[14].Splits != 0.02 {
		t.Errorf("split-day row changed: %+v", got[14])
	}
}
