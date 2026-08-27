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
// (Datetime,Open,High,Low,Close,Adj Close,Volume,Dividends,Stock Splits).
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
			Dividends: parseF(row[7]),
			Splits:    parseF(row[8]),
		})
	}
	return bars
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
