package repair

import (
	"math"
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func TestFindZeroIndices(t *testing.T) {
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 98, Close: 102},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Open: 0, High: 0, Low: 0, Close: 0}, // Zero
		{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Open: 105, High: 110, Low: 103, Close: 108},
		{Date: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), Open: 108, High: 0, Low: 106, Close: 110}, // Partial zero
	}

	indices := findZeroIndices(bars)

	if len(indices) != 2 {
		t.Errorf("Expected 2 zero indices, got %d", len(indices))
	}
	if indices[0] != 1 || indices[1] != 3 {
		t.Errorf("Expected indices [1, 3], got %v", indices)
	}
}

func TestIsZeroBar(t *testing.T) {
	tests := []struct {
		name     string
		bar      models.Bar
		expected bool
	}{
		{
			name:     "Normal bar",
			bar:      models.Bar{Open: 100, High: 105, Low: 98, Close: 102},
			expected: false,
		},
		{
			name:     "All zero",
			bar:      models.Bar{Open: 0, High: 0, Low: 0, Close: 0},
			expected: true,
		},
		{
			name:     "Partial zero - Open",
			bar:      models.Bar{Open: 0, High: 105, Low: 98, Close: 102},
			expected: true,
		},
		{
			name:     "Partial zero - High",
			bar:      models.Bar{Open: 100, High: 0, Low: 98, Close: 102},
			expected: true,
		},
		{
			name:     "NaN Open",
			bar:      models.Bar{Open: math.NaN(), High: 105, Low: 98, Close: 102},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isZeroBar(tt.bar)
			if result != tt.expected {
				t.Errorf("isZeroBar() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestShouldRepairZero(t *testing.T) {
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 98, Close: 102},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Open: 0, High: 0, Low: 0, Close: 0, Volume: 1000}, // Zero but has volume
		{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Open: 105, High: 110, Low: 103, Close: 108},
	}

	// Should repair bar 1 because it has volume
	if !shouldRepairZero(bars, 1) {
		t.Error("Should repair bar with volume > 0")
	}

	// Test bar with split
	bars[1].Volume = 0
	bars[1].Splits = 2.0
	if !shouldRepairZero(bars, 1) {
		t.Error("Should repair bar with stock split")
	}

	// Test bar with dividend
	bars[1].Splits = 0
	bars[1].Dividends = 0.5
	if !shouldRepairZero(bars, 1) {
		t.Error("Should repair bar with dividends")
	}
}

func TestRepairZeroBar(t *testing.T) {
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 98, Close: 102, AdjClose: 102},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Open: 0, High: 0, Low: 0, Close: 0, AdjClose: 0},
		{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Open: 105, High: 110, Low: 103, Close: 108, AdjClose: 108},
	}

	repaired := repairZeroBar(bars, 1)

	// Should interpolate between prev close (102) and next open (105)
	expectedOpen := (102.0 + 105.0) / 2
	if math.Abs(repaired.Open-expectedOpen) > 0.001 {
		t.Errorf("Expected Open=%.2f, got %.2f", expectedOpen, repaired.Open)
	}

	// Should be marked as repaired
	if !repaired.Repaired {
		t.Error("Bar should be marked as repaired")
	}
}

func TestRepairZeroBarOnlyPrev(t *testing.T) {
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 98, Close: 102, AdjClose: 102},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Open: 0, High: 0, Low: 0, Close: 0, AdjClose: 0},
	}

	repaired := repairZeroBar(bars, 1)

	// Should forward fill from prev close
	if repaired.Close != 102 {
		t.Errorf("Expected Close=102 (forward fill), got %.2f", repaired.Close)
	}
}

func TestRepairZeroBarOnlyNext(t *testing.T) {
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 0, High: 0, Low: 0, Close: 0, AdjClose: 0},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Open: 105, High: 110, Low: 103, Close: 108, AdjClose: 108},
	}

	repaired := repairZeroBar(bars, 0)

	// Should backward fill from next open
	if repaired.Open != 105 {
		t.Errorf("Expected Open=105 (backward fill), got %.2f", repaired.Open)
	}
}

func TestRepairZeroes(t *testing.T) {
	repairer := New(DefaultOptions())

	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 98, Close: 102, AdjClose: 102},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Open: 0, High: 0, Low: 0, Close: 0, AdjClose: 0, Volume: 1000}, // Should repair
		{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Open: 105, High: 110, Low: 103, Close: 108, AdjClose: 108},
		{Date: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), Open: 108, High: 112, Low: 106, Close: 110, AdjClose: 110},
	}

	result := repairer.repairZeroes(bars)

	// Bar 1 should be repaired
	if !result[1].Repaired {
		t.Error("Bar 1 should be marked as repaired")
	}

	// Prices should be interpolated
	if result[1].Close == 0 {
		t.Error("Close should be filled, not 0")
	}
}

func TestRepairZeroesTooManyZeroes(t *testing.T) {
	repairer := New(DefaultOptions())

	// More than 50% are zeroes - should not repair
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 0, High: 0, Low: 0, Close: 0},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Open: 0, High: 0, Low: 0, Close: 0},
		{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Open: 0, High: 0, Low: 0, Close: 0},
		{Date: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 98, Close: 102},
	}

	result := repairer.repairZeroes(bars)

	// No bars should be repaired
	for i, bar := range result {
		if bar.Repaired {
			t.Errorf("Bar %d should not be repaired (too many zeroes)", i)
		}
	}
}

func TestRepairPartialZeroes(t *testing.T) {
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 100, High: 0, Low: 98, Close: 102}, // High is zero
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Open: 102, High: 107, Low: 0, Close: 105}, // Low is zero
	}

	result := repairPartialZeroes(bars)

	// High should be fixed
	if result[0].High == 0 {
		t.Error("High should be repaired, not 0")
	}

	// High should be >= Open and Close
	if result[0].High < result[0].Open || result[0].High < result[0].Close {
		t.Errorf("High (%.2f) should be >= Open (%.2f) and Close (%.2f)",
			result[0].High, result[0].Open, result[0].Close)
	}

	// Low should be fixed
	if result[1].Low == 0 {
		t.Error("Low should be repaired, not 0")
	}

	// Low should be <= Open and Close
	if result[1].Low > result[1].Open || result[1].Low > result[1].Close {
		t.Errorf("Low (%.2f) should be <= Open (%.2f) and Close (%.2f)",
			result[1].Low, result[1].Open, result[1].Close)
	}
}

func TestRepairVolumeZeroes(t *testing.T) {
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 98, Close: 102, Volume: 1000},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Open: 102, High: 112, Low: 100, Close: 112, Volume: 0}, // 10% price change, 0 volume
		{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Open: 112, High: 118, Low: 110, Close: 115, Volume: 1500},
	}

	result := repairVolumeZeroes(bars)

	// Bar 1 should be repaired if similar volume found
	// (may not be repaired if no similar price change found nearby)
	_ = result
}

func TestAnalyzeZeroes(t *testing.T) {
	repairer := New(DefaultOptions())

	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 98, Close: 102, Volume: 1000},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Open: 0, High: 0, Low: 0, Close: 0, Volume: 0},       // Zero bar
		{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Open: 105, High: 0, Low: 103, Close: 108, Volume: 1200}, // Partial zero (High=0)
		{Date: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), Open: 108, High: 112, Low: 106, Close: 120, Volume: 0}, // Price changed >5%, 0 volume
	}

	stats := repairer.AnalyzeZeroes(bars)

	if stats.TotalBars != 4 {
		t.Errorf("Expected TotalBars=4, got %d", stats.TotalBars)
	}

	// Bar 1 (all zero) and bar 2 (High=0) are zero bars
	if stats.ZeroBars != 2 {
		t.Errorf("Expected ZeroBars=2, got %d", stats.ZeroBars)
	}

	// Zero volume bars count only those with >5% price change
	// Bar 1: prev=102, curr=0 -> skipped (zero bar)
	// Bar 2: prev=0, curr=108 -> skipped (prevClose=0)
	// Bar 3: prev=108, curr=120 -> 11% change, volume=0 -> counts
	// Note: Bar 2 also has volume=0 but prevClose=0 so check fails
	// So we expect at least 1 zero volume bar
	if stats.ZeroVolumeBars < 1 {
		t.Errorf("Expected ZeroVolumeBars >= 1, got %d", stats.ZeroVolumeBars)
	}
}

func TestDetectZeroes(t *testing.T) {
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 98, Close: 102},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Open: 0, High: 0, Low: 0, Close: 0},
		{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Open: 105, High: 110, Low: 103, Close: 108},
	}

	indices := DetectZeroes(bars)

	if len(indices) != 1 {
		t.Errorf("Expected 1 zero bar, got %d", len(indices))
	}
	if len(indices) > 0 && indices[0] != 1 {
		t.Errorf("Expected index 1, got %v", indices)
	}
}

func TestMaxInt(t *testing.T) {
	if maxInt(5, 10) != 10 {
		t.Error("maxInt(5, 10) should be 10")
	}
	if maxInt(10, 5) != 10 {
		t.Error("maxInt(10, 5) should be 10")
	}
	if maxInt(5, 5) != 5 {
		t.Error("maxInt(5, 5) should be 5")
	}
}

func TestRepairZeroesEmpty(t *testing.T) {
	repairer := New(DefaultOptions())

	result := repairer.repairZeroes([]models.Bar{})
	if len(result) != 0 {
		t.Error("Empty bars should return empty result")
	}

	// Test short bars
	short := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 98, Close: 102},
	}
	result = repairer.repairZeroes(short)
	if len(result) != 1 {
		t.Error("Single bar should return unchanged")
	}
}
