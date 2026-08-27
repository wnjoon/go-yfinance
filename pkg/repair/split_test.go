package repair

import (
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

func TestApplySplitCorrection(t *testing.T) {
	// Create bars representing unadjusted data before a 2:1 split
	bars := []models.Bar{
		{
			Date:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Open:     100,
			High:     110,
			Low:      95,
			Close:    105,
			AdjClose: 105,
			Volume:   1000,
		},
		{
			Date:     time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			Open:     106,
			High:     112,
			Low:      100,
			Close:    108,
			AdjClose: 108,
			Volume:   1200,
		},
		{
			Date:     time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
			Open:     54, // After 2:1 split, price halved
			High:     58,
			Low:      50,
			Close:    55,
			AdjClose: 55,
			Volume:   2400,
			Splits:   2.0,
		},
	}

	// Apply correction for 2:1 split at index 2
	result := applySplitCorrection(bars, 2, 2.0)

	// Pre-split bars should be divided by 2
	if result[0].Close != 52.5 {
		t.Errorf("Expected Close=52.5, got %.2f", result[0].Close)
	}
	if result[1].Close != 54 {
		t.Errorf("Expected Close=54, got %.2f", result[1].Close)
	}

	// Volume should be multiplied by 2
	if result[0].Volume != 2000 {
		t.Errorf("Expected Volume=2000, got %d", result[0].Volume)
	}

	// Bars should be marked as repaired
	if !result[0].Repaired || !result[1].Repaired {
		t.Error("Pre-split bars should be marked as repaired")
	}
	if !result[2].Repaired {
		t.Error("Split date bar should be marked as repaired")
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

func TestReverseSplitCorrection(t *testing.T) {
	// Test reverse split (1:4 -> ratio = 0.25)
	bars := []models.Bar{
		{
			Date:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Open:     25,
			High:     27,
			Low:      23,
			Close:    26,
			AdjClose: 26,
			Volume:   4000,
		},
		{
			Date:     time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			Open:     100, // After 1:4 reverse split, price 4x
			High:     108,
			Low:      95,
			Close:    104,
			AdjClose: 104,
			Volume:   1000,
			Splits:   0.25, // 1:4 reverse split
		},
	}

	// For reverse split with ratio 0.25, pre-split prices should be multiplied by 4
	result := applySplitCorrection(bars, 1, 0.25)

	// Pre-split bar should have price multiplied by 4
	if result[0].Close != 104 { // 26 * 4 = 104
		t.Errorf("Expected Close=104, got %.2f", result[0].Close)
	}

	// Volume should be divided by 4
	if result[0].Volume != 1000 { // 4000 / 4 = 1000
		t.Errorf("Expected Volume=1000, got %d", result[0].Volume)
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
		for i := range got {
			for _, c := range []struct {
				name      string
				got, want float64
			}{
				{"Open", got[i].Open, fixed[i].Open}, {"High", got[i].High, fixed[i].High},
				{"Low", got[i].Low, fixed[i].Low}, {"Close", got[i].Close, fixed[i].Close},
				{"AdjClose", got[i].AdjClose, fixed[i].AdjClose},
			} {
				if !closeTo(c.got, c.want, 5e-5) {
					t.Fatalf("last=%d row=%d %s: got %v want %v", last, i, c.name, c.got, c.want)
				}
			}
			if got[i].Volume != fixed[i].Volume {
				t.Fatalf("last=%d row=%d Volume: got %d want %d", last, i, got[i].Volume, fixed[i].Volume)
			}
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

func TestApplySplitRangesUsesPythonBankersRounding(t *testing.T) {
	bars := []models.Bar{{Open: 1, High: 1, Low: 1, Close: 1, AdjClose: 1, Volume: 5}}
	got := applySplitRanges(bars, 1, 2, []bool{false}, []bool{true})
	if got[0].Volume != 2 {
		t.Fatalf("2.5 volume rounded to %d, want Python/NumPy half-even 2", got[0].Volume)
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
	got := repairSplitAtIndex(bars, 5, 2, "1d")
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

func TestSuppressLocalSplitVolatility(t *testing.T) {
	ratio := []float64{1, 0.8, 1.8, 1.8, 0.8, 1.8}
	up, down := make([]bool, len(ratio)), make([]bool, len(ratio))
	up[3] = true
	suppressLocalVolatility(ratio, up, down, 2, "1d")
	if up[3] {
		t.Fatal("locally ordinary candidate should be suppressed")
	}
}

func TestSuppressSplitVolumeSpikeFalsePositive(t *testing.T) {
	bars := make([]models.Bar, 20)
	for i := range bars {
		bars[i].Volume = int64(950 + (i%5)*25)
	}
	// Descending candidate idx=6 refers to the actual drop row at desc idx=5.
	bars[len(bars)-1-5].Volume = 10000
	up, down := make([]bool, len(bars)), make([]bool, len(bars))
	up[6] = true
	suppressSplitVolumeSpikes(bars, up, down)
	if up[6] {
		t.Fatal("catastrophic move on exceptional volume should be suppressed")
	}
}
