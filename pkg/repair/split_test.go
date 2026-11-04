package repair

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func TestFindSplitIndices(t *testing.T) {
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Splits: 0},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Splits: 4.0}, // 4:1 split
		{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Splits: 0},
		{Date: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), Splits: 2.0}, // 2:1 split
		{Date: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), Splits: 1.0}, // 1:1 = no split
	}

	indices := findSplitIndices(bars)

	if len(indices) != 2 {
		t.Errorf("Expected 2 split events, got %d", len(indices))
	}
	if indices[0] != 1 || indices[1] != 3 {
		t.Errorf("Expected indices [1, 3], got %v", indices)
	}
}

func TestOHLCMedian(t *testing.T) {
	bar := models.Bar{
		Open:  10,
		High:  15,
		Low:   8,
		Close: 12,
	}

	median := ohlcMedian(bar)
	// Sorted: 8, 10, 12, 15 -> Median = (10+12)/2 = 11
	expected := 11.0

	if math.Abs(median-expected) > 0.001 {
		t.Errorf("Expected median %.2f, got %.2f", expected, median)
	}
}

func TestMinInt(t *testing.T) {
	if minInt(5, 10) != 5 {
		t.Error("minInt(5, 10) should be 5")
	}
	if minInt(10, 5) != 5 {
		t.Error("minInt(10, 5) should be 5")
	}
	if minInt(5, 5) != 5 {
		t.Error("minInt(5, 5) should be 5")
	}
}

func TestRepairStockSplitsNoSplits(t *testing.T) {
	repairer := New(DefaultOptions())

	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Close: 101},
		{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Close: 102},
	}

	result := repairer.repairStockSplits(bars)

	// No splits, no changes
	for i, bar := range result {
		if bar.Repaired {
			t.Errorf("Bar %d should not be repaired (no splits)", i)
		}
	}
}

func TestDetectBadSplits(t *testing.T) {
	// Create data with unadjusted split
	// Price drops by 50% on split date (should have been adjusted)
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 98, Close: 102},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Open: 51, High: 53, Low: 49, Close: 51, Splits: 2.0},
	}

	badSplits := DetectBadSplits(bars)

	// The 2:1 split at index 1 should be detected as bad (unadjusted)
	if len(badSplits) != 1 {
		t.Errorf("Expected 1 bad split, got %d", len(badSplits))
	}
	if len(badSplits) > 0 && badSplits[0] != 1 {
		t.Errorf("Expected bad split at index 1, got %v", badSplits)
	}
}

func TestDetectBadSplitsAdjusted(t *testing.T) {
	// Create properly adjusted split data
	// Price stays relatively stable around split (already adjusted)
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 98, Close: 102},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Open: 101, High: 106, Low: 99, Close: 103, Splits: 2.0},
	}

	badSplits := DetectBadSplits(bars)

	// No bad splits - data is already adjusted
	if len(badSplits) != 0 {
		t.Errorf("Expected 0 bad splits, got %d", len(badSplits))
	}
}

func TestAnalyzeSplits(t *testing.T) {
	repairer := New(DefaultOptions())

	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Close: 50, Splits: 2.0},
		{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Close: 25, Splits: 2.0},
	}

	stats := repairer.AnalyzeSplits(bars)

	if stats.TotalSplits != 2 {
		t.Errorf("Expected 2 splits, got %d", stats.TotalSplits)
	}
	if len(stats.Splits) != 2 {
		t.Errorf("Expected 2 split info entries, got %d", len(stats.Splits))
	}
	if stats.Splits[0].Ratio != 2.0 {
		t.Errorf("Expected ratio 2.0, got %.2f", stats.Splits[0].Ratio)
	}
}

// TestRepairStockSplitsNRDYGolden ports the Python yfinance 1.7.0 NRDY
// regression fixture added by ranaroussi/yfinance#2958 (0f29c859 plus the
// 5c1f64e follow-up). It exercises multiple alternating corrupt/good ranges.
func TestRepairStockSplitsNRDYGolden(t *testing.T) {
	for _, last := range []int{0, 27} {
		bad := loadBarsCSV(t, "NRDY-1d-bad-stock-split.csv")
		fixed := loadBarsCSV(t, "NRDY-1d-bad-stock-split-fixed.csv")
		if last > 0 {
			bad, fixed = bad[len(bad)-last:], fixed[len(fixed)-last:]
		}
		opts := DefaultOptions()
		opts.Interval = "1d"
		got := New(opts).repairStockSplits(bad)
		assertSplitGolden(t, fmt.Sprintf("NRDY last=%d", last), got, fixed)
	}
}

// assertSplitGolden mirrors upstream test_repair_bad_stock_splits: every price
// column must match with rtol=5e-5, with NaN equal to NaN (equal_nan=True,
// added in Python 1.8.0 for the NFE fixture's missing split-day row). Volume
// is integral in Go, so it must match exactly.
func assertSplitGolden(t *testing.T, label string, got, fixed []models.Bar) {
	t.Helper()
	if len(got) != len(fixed) {
		t.Fatalf("%s: got %d rows, want %d", label, len(got), len(fixed))
	}
	for i := range got {
		for _, c := range []struct {
			name      string
			got, want float64
		}{
			{"Open", got[i].Open, fixed[i].Open}, {"High", got[i].High, fixed[i].High},
			{"Low", got[i].Low, fixed[i].Low}, {"Close", got[i].Close, fixed[i].Close},
			{"AdjClose", got[i].AdjClose, fixed[i].AdjClose},
		} {
			if math.IsNaN(c.got) && math.IsNaN(c.want) {
				continue
			}
			if !closeTo(c.got, c.want, 5e-5) {
				t.Fatalf("%s row=%d %s: got %v want %v", label, i, c.name, c.got, c.want)
			}
		}
		if got[i].Volume != fixed[i].Volume {
			t.Fatalf("%s row=%d Volume: got %d want %d", label, i, got[i].Volume, fixed[i].Volume)
		}
	}
}

func TestRepairStockSplitsAllZeroVolumeIsNotRepaired(t *testing.T) {
	bars := make([]models.Bar, 8)
	for i := range bars {
		bars[i] = models.Bar{Date: time.Date(2024, 1, i+1, 0, 0, 0, 0, time.UTC), Open: 100, High: 100, Low: 100, Close: 100, AdjClose: 100}
	}
	bars[4].Open, bars[4].High, bars[4].Low, bars[4].Close, bars[4].AdjClose, bars[4].Splits = 50, 50, 50, 50, 50, 2
	got := New(DefaultOptions()).repairStockSplits(bars)
	if CountRepaired(got) != 0 {
		t.Fatalf("all-zero volume must veto split repair")
	}
}

func TestRepairStockSplitsOnlyInterdayIntervals(t *testing.T) {
	bars := unadjustedSplitBars([]int64{1200, 1150, 1250, 1180, 1220, 2400, 2500, 2450})
	opts := DefaultOptions()
	opts.Interval = "1h"
	got := New(opts).repairStockSplits(bars)
	if CountRepaired(got) != 0 {
		t.Fatal("intraday split repair must be disabled")
	}
}

func TestRepairSplitDailyLookAheadStopsAfterFiveRows(t *testing.T) {
	bars := make([]models.Bar, 14)
	for i := range bars {
		p := 100.0
		if i >= 12 {
			p = 50
		}
		bars[i] = models.Bar{Date: time.Date(2024, 1, i+1, 0, 0, 0, 0, time.UTC), Open: p, High: p, Low: p, Close: p, AdjClose: p, Volume: 1000}
	}
	bars[5].Splits = 2
	got := New(DefaultOptions()).repairSplitAtIndex(bars, 5, 2)
	if CountRepaired(got) != 0 {
		t.Fatal("signal beyond upstream's five-row look-ahead was used")
	}
}

func TestRepairSplitUsesAdjustedPricesForDividendDrop(t *testing.T) {
	bars := make([]models.Bar, 9)
	for i := range bars {
		close, adj := 100.0, 50.0
		if i >= 5 {
			close, adj = 50, 50
		}
		bars[i] = models.Bar{Date: time.Date(2024, 1, i+1, 0, 0, 0, 0, time.UTC), Open: close, High: close, Low: close, Close: close, AdjClose: adj, Volume: 1000}
	}
	bars[5].Splits = 2
	got := New(DefaultOptions()).repairStockSplits(bars)
	if CountRepaired(got) != 0 {
		t.Fatal("dividend-adjusted continuous prices must not be treated as a split error")
	}
}
