package repair

import (
	"math"
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func TestFindCapitalGainsIndices(t *testing.T) {
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100, CapitalGains: 0},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Close: 99, CapitalGains: 0.5},
		{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Close: 98, CapitalGains: 0},
		{Date: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), Close: 97, CapitalGains: 0.3},
	}

	indices := findCapitalGainsIndices(bars)

	if len(indices) != 2 {
		t.Errorf("Expected 2 capital gains events, got %d", len(indices))
	}
	if indices[0] != 1 || indices[1] != 3 {
		t.Errorf("Expected indices [1, 3], got %v", indices)
	}
}

func TestCalculateNormalVolatility(t *testing.T) {
	// Create bars with known volatility pattern
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Close: 101}, // +1%
		{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Close: 100}, // -1%
		{Date: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), Close: 102}, // +2%
		{Date: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), Close: 100}, // -2%
	}

	vol := calculateNormalVolatility(bars, nil)

	// Expected: mean of |1%, -1%, 2%, -2%| = mean of |0.01, 0.01, 0.02, 0.02| = 0.015
	expected := 0.015
	if math.Abs(vol-expected) > 0.001 {
		t.Errorf("Expected volatility ~%.4f, got %.4f", expected, vol)
	}
}

func TestDetectDoubleCount(t *testing.T) {
	// Scenario: CG double-count
	// Previous close: 100
	// Current close: 99 (dropped ~1%)
	// Dividend reported: 1.5 (includes CG)
	// Capital Gains: 0.5
	// True dividend: 1.0
	// If not double-counted, drop should be (1.5 + 0.5)/100 = 2%
	// Actual drop is ~1%, which matches true dividend (1.0/100 = 1%)
	// This suggests CG was double-counted

	bars := []models.Bar{
		{
			Date:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Close:    100,
			AdjClose: 100,
		},
		{
			Date:         time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			Close:        99,
			AdjClose:     99,
			Dividends:    1.5,  // Includes CG (double-counted)
			CapitalGains: 0.5,
		},
	}

	normalVol := 0.001 // Very low volatility for this test

	result := detectDoubleCount(bars, 1, normalVol)

	if !result {
		t.Error("Expected double-count to be detected")
	}
}

func TestApplyCapitalGainsRepair(t *testing.T) {
	// Create test bars with known double-counting issue
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100, AdjClose: 98},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Close: 100, AdjClose: 99},
		{
			Date:         time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
			Close:        99,
			AdjClose:     99,
			Dividends:    1.5,
			CapitalGains: 0.5,
		},
	}

	cgIndices := []int{2}
	result := applyCapitalGainsRepair(bars, cgIndices)

	// Check that earlier bars are marked as repaired
	if !result[0].Repaired {
		t.Error("Bar 0 should be marked as repaired")
	}
	if !result[1].Repaired {
		t.Error("Bar 1 should be marked as repaired")
	}
	if !result[2].Repaired {
		t.Error("Bar 2 (CG date) should be marked as repaired")
	}

	// AdjClose values should be different from original (corrected)
	if result[0].AdjClose == bars[0].AdjClose && result[1].AdjClose == bars[1].AdjClose {
		t.Error("AdjClose values should be corrected")
	}
}

func TestCapitalGainsRepairNoAction(t *testing.T) {
	repairer := New(DefaultOptions())

	// No capital gains - should return unchanged
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100, AdjClose: 100},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Close: 101, AdjClose: 101},
	}

	result := repairer.repairCapitalGains(bars)

	if len(result) != len(bars) {
		t.Errorf("Expected %d bars, got %d", len(bars), len(result))
	}

	// No bars should be repaired
	for i, bar := range result {
		if bar.Repaired {
			t.Errorf("Bar %d should not be repaired (no CG data)", i)
		}
	}
}

func TestAnalyzeCapitalGains(t *testing.T) {
	repairer := New(DefaultOptions())

	// Create bars with capital gains
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100, AdjClose: 100},
		{
			Date:         time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			Close:        99,
			AdjClose:     99,
			Dividends:    1.5,
			CapitalGains: 0.5,
		},
	}

	stats := repairer.AnalyzeCapitalGains(bars)

	if stats.TotalEvents != 1 {
		t.Errorf("Expected 1 CG event, got %d", stats.TotalEvents)
	}
}

func TestHasCapitalGains(t *testing.T) {
	withCG := []models.Bar{
		{CapitalGains: 0},
		{CapitalGains: 0.5},
	}

	withoutCG := []models.Bar{
		{CapitalGains: 0},
		{CapitalGains: 0},
	}

	if !HasCapitalGains(withCG) {
		t.Error("Should detect capital gains")
	}

	if HasCapitalGains(withoutCG) {
		t.Error("Should not detect capital gains")
	}
}

func TestHasDividends(t *testing.T) {
	withDiv := []models.Bar{
		{Dividends: 0},
		{Dividends: 0.5},
	}

	withoutDiv := []models.Bar{
		{Dividends: 0},
		{Dividends: 0},
	}

	if !HasDividends(withDiv) {
		t.Error("Should detect dividends")
	}

	if HasDividends(withoutDiv) {
		t.Error("Should not detect dividends")
	}
}

func TestHasSplits(t *testing.T) {
	withSplit := []models.Bar{
		{Splits: 0},
		{Splits: 4.0}, // 4:1 split
	}

	noSplit := []models.Bar{
		{Splits: 0},
		{Splits: 1.0}, // 1:1 is no split
	}

	if !HasSplits(withSplit) {
		t.Error("Should detect splits")
	}

	if HasSplits(noSplit) {
		t.Error("Should not detect splits (1:1)")
	}
}
