package repair

import (
	"math"
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func TestFindDividendIndices(t *testing.T) {
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Dividends: 0},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Dividends: 0.5},
		{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Dividends: 0},
		{Date: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), Dividends: 0.3},
	}

	indices := findDividendIndices(bars)

	if len(indices) != 2 {
		t.Errorf("Expected 2 dividend indices, got %d", len(indices))
	}
	if indices[0] != 1 || indices[1] != 3 {
		t.Errorf("Expected indices [1, 3], got %v", indices)
	}
}

func TestAnalyzeDividendNormal(t *testing.T) {
	// Create properly adjusted dividend data
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100, Low: 98, High: 102, AdjClose: 98.5},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Close: 98, Low: 97, High: 100, AdjClose: 98, Dividends: 1.5},
	}

	status := analyzeDividend(bars, 1, 100.0)

	if status.IsMissingAdj {
		t.Error("Should not flag as missing adjustment for properly adjusted data")
	}
	if status.IsTooLarge {
		t.Error("Should not flag normal dividend as too large")
	}
	if status.IsTooSmall {
		t.Error("Should not flag normal dividend as too small")
	}
}

func TestAnalyzeDividendMissingAdj(t *testing.T) {
	// Adj Close = Close (no dividend adjustment)
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100, Low: 98, High: 102, AdjClose: 100},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Close: 98, Low: 97, High: 100, AdjClose: 98, Dividends: 1.5},
	}

	status := analyzeDividend(bars, 1, 100.0)

	if !status.IsMissingAdj {
		t.Error("Should flag missing adjustment when Adj Close = Close")
	}
}

func TestAnalyzeDividendTooLarge(t *testing.T) {
	// Dividend that's 100x too big
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100, Low: 98, High: 102, AdjClose: 100},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Close: 98, Low: 96, High: 100, AdjClose: 98, Dividends: 150}, // 150% dividend - clearly wrong
	}

	status := analyzeDividend(bars, 1, 100.0)

	if !status.IsTooLarge {
		t.Error("Should flag dividend > 100% as too large")
	}
}

func TestFixMissingDivAdj(t *testing.T) {
	// Simulate missing dividend adjustment
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100, AdjClose: 100},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Close: 99, AdjClose: 99, Dividends: 1.0},
	}

	result := fixMissingDivAdj(bars, 1)

	// Pre-dividend bar should have adjusted close < close
	expectedAdj := 1.0 - 1.0/100.0 // 0.99
	if math.Abs(result[0].AdjClose-100*expectedAdj) > 0.01 {
		t.Errorf("Expected AdjClose ~99, got %.2f", result[0].AdjClose)
	}

	if !result[0].Repaired {
		t.Error("Pre-dividend bar should be marked as repaired")
	}
}

func TestFixDivTooLarge(t *testing.T) {
	// Dividend is 100x too big
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100, AdjClose: 100},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Close: 99, AdjClose: 99, Dividends: 100}, // Should be 1.0
	}

	result := fixDivTooLarge(bars, 1, 100.0)

	// Dividend should be divided by 100
	if math.Abs(result[1].Dividends-1.0) > 0.01 {
		t.Errorf("Expected Dividends=1.0, got %.2f", result[1].Dividends)
	}

	if !result[1].Repaired {
		t.Error("Dividend bar should be marked as repaired")
	}
}

func TestFixDivTooSmall(t *testing.T) {
	// Dividend is 100x too small
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100, AdjClose: 100},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Close: 99, AdjClose: 99, Dividends: 0.01}, // Should be 1.0
	}

	result := fixDivTooSmall(bars, 1, 100.0)

	// Dividend should be multiplied by 100
	if math.Abs(result[1].Dividends-1.0) > 0.01 {
		t.Errorf("Expected Dividends=1.0, got %.2f", result[1].Dividends)
	}

	if !result[1].Repaired {
		t.Error("Dividend bar should be marked as repaired")
	}
}

func TestIsPhantomDividend(t *testing.T) {
	// Create data with two similar dividends close together
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100, Low: 98},
		{Date: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), Close: 99, Low: 98, Dividends: 1.0},  // First div - larger drop
		{Date: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), Close: 99, Low: 99, Dividends: 1.0}, // Phantom - smaller drop
	}

	// The second dividend (smaller drop) should be detected as phantom
	isPhantom := isPhantomDividend(bars, 2)

	if !isPhantom {
		t.Error("Should detect duplicate dividend as phantom")
	}
}

func TestRemovePhantomDiv(t *testing.T) {
	// Create data with phantom dividend
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100, AdjClose: 98},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Close: 99, AdjClose: 99, Dividends: 1.0},
	}

	result := removePhantomDiv(bars, 1)

	// Dividend should be zeroed
	if result[1].Dividends != 0 {
		t.Errorf("Expected Dividends=0, got %.2f", result[1].Dividends)
	}

	if !result[1].Repaired {
		t.Error("Phantom dividend bar should be marked as repaired")
	}
}

func TestRepairDividendsNoDividends(t *testing.T) {
	repairer := New(DefaultOptions())

	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Close: 101},
		{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Close: 102},
	}

	result := repairer.repairDividends(bars)

	// No changes should be made
	for i, bar := range result {
		if bar.Repaired {
			t.Errorf("Bar %d should not be repaired (no dividends)", i)
		}
	}
}

func TestRepairDividendsSkipWeekly(t *testing.T) {
	opts := DefaultOptions()
	opts.Interval = "1wk"
	repairer := New(opts)

	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100, AdjClose: 100},
		{Date: time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC), Close: 99, AdjClose: 99, Dividends: 100}, // Wrong but should be skipped
	}

	result := repairer.repairDividends(bars)

	// Should not repair weekly data
	if result[1].Dividends != 100 {
		t.Error("Should not modify weekly data")
	}
}

func TestRepairDividendsSkipCapitalGains(t *testing.T) {
	repairer := New(DefaultOptions())

	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100, AdjClose: 100, CapitalGains: 0.5},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Close: 99, AdjClose: 99, Dividends: 100}, // Wrong but should be skipped
	}

	result := repairer.repairDividends(bars)

	// Should not repair when capital gains present
	if result[1].Dividends != 100 {
		t.Error("Should not modify data with capital gains")
	}
}

func TestAnalyzeDividends(t *testing.T) {
	repairer := New(DefaultOptions())

	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100, Low: 98, AdjClose: 100},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Close: 99, Low: 98, AdjClose: 99, Dividends: 1.0},
		{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Close: 98, Low: 97, AdjClose: 98},
		{Date: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), Close: 97, Low: 96, AdjClose: 97, Dividends: 150}, // Too big
	}

	stats := repairer.AnalyzeDividends(bars)

	if stats.TotalDividends != 2 {
		t.Errorf("Expected 2 dividends, got %d", stats.TotalDividends)
	}

	// Second dividend should be flagged as too large
	if len(stats.Dividends) >= 2 && !stats.Dividends[1].IsTooLarge {
		t.Error("Second dividend should be flagged as too large")
	}
}

func TestDetectBadDividends(t *testing.T) {
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100, Low: 98, AdjClose: 100},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Close: 99, Low: 98, AdjClose: 99, Dividends: 1.0}, // Normal
		{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Close: 98, Low: 97, AdjClose: 98},
		{Date: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), Close: 97, Low: 96, AdjClose: 97, Dividends: 200}, // Bad
	}

	badIndices := DetectBadDividends(bars, "USD")

	// At least the 200 dividend should be detected as bad
	if len(badIndices) == 0 {
		t.Error("Expected at least one bad dividend to be detected")
	}

	found := false
	for _, idx := range badIndices {
		if idx == 3 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected index 3 in bad indices: %v", badIndices)
	}
}

func TestCalculateWindowVolatility(t *testing.T) {
	// Create bars where Close[i] - Low[i+1] averages to about 1.0
	bars := []models.Bar{
		{Close: 100, Low: 98},
		{Close: 101, Low: 100}, // diff = 100 - 100 = 0
		{Close: 102, Low: 100}, // diff = 101 - 100 = 1
		{Close: 103, Low: 101}, // diff = 102 - 101 = 1
		{Close: 104, Low: 102}, // diff = 103 - 102 = 1
		{Close: 105, Low: 103}, // diff = 104 - 103 = 1
		{Close: 106, Low: 104}, // diff = 105 - 104 = 1
		{Close: 107, Low: 105}, // diff = 106 - 105 = 1
	}

	vol := calculateWindowVolatility(bars, 4)

	if math.IsNaN(vol) {
		t.Error("Volatility should not be NaN with sufficient data")
	}

	// The diffs are abs(Close[i] - Low[i+1]), which averages to ~1.0
	if vol <= 0 || vol > 3.0 {
		t.Errorf("Expected reasonable volatility, got %.2f", vol)
	}
}

func TestRepairDividendsEmpty(t *testing.T) {
	repairer := New(DefaultOptions())

	result := repairer.repairDividends([]models.Bar{})
	if len(result) != 0 {
		t.Error("Empty bars should return empty result")
	}

	// Test with too few bars
	short := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100, Dividends: 1.0},
	}
	result = repairer.repairDividends(short)
	if len(result) != 1 {
		t.Error("Short bars should return unchanged")
	}
}

func TestRepairDividendsKWF(t *testing.T) {
	opts := DefaultOptions()
	opts.Currency = "KWF" // Kuwaiti Dinar - 1000 divisor
	repairer := New(opts)

	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100, Low: 98, AdjClose: 100},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Close: 99, Low: 98, AdjClose: 99, Dividends: 1500}, // 1000x too big for KWF
		{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Close: 98, Low: 97, AdjClose: 98},
	}

	stats := repairer.AnalyzeDividends(bars)

	// Should use 1000 divisor for KWF
	if stats.TotalDividends != 1 {
		t.Errorf("Expected 1 dividend, got %d", stats.TotalDividends)
	}
}
