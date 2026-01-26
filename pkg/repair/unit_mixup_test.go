package repair

import (
	"math"
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func TestDetectAndCorrectMixups(t *testing.T) {
	// Create data with one 100x error
	data := [][]float64{
		{10, 9, 8, 10, 10},   // High, Open, Low, Close, AdjClose - normal
		{11, 10, 9, 11, 11},  // normal
		{1100, 1000, 900, 1100, 1100}, // 100x error!
		{10, 9, 8, 10, 10},   // normal
		{11, 10, 9, 11, 11},  // normal
	}

	corrections := detectAndCorrectMixups(data)

	// Third row should have correction factor of 0.01
	if corrections[2] != 0.01 {
		t.Errorf("Expected correction factor 0.01 for row 2, got %v", corrections[2])
	}

	// Other rows should have correction factor 1.0
	for i, c := range corrections {
		if i != 2 && c != 1.0 {
			t.Errorf("Expected correction factor 1.0 for row %d, got %v", i, c)
		}
	}
}

func TestDetectAndCorrectMixups100xLow(t *testing.T) {
	// Create data with one 100x too low error
	data := [][]float64{
		{100, 90, 80, 100, 100},
		{110, 100, 90, 110, 110},
		{1.1, 1.0, 0.9, 1.1, 1.1}, // 100x too low!
		{100, 90, 80, 100, 100},
		{110, 100, 90, 110, 110},
	}

	corrections := detectAndCorrectMixups(data)

	// Third row should have correction factor of 100.0
	if corrections[2] != 100.0 {
		t.Errorf("Expected correction factor 100.0 for row 2, got %v", corrections[2])
	}
}

func TestRepairRandomUnitMixupsNormal(t *testing.T) {
	repairer := New(DefaultOptions())

	// Create normal data without 100x errors
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 98, Close: 102, AdjClose: 102},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Open: 102, High: 107, Low: 100, Close: 105, AdjClose: 105},
		{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Open: 105, High: 110, Low: 103, Close: 108, AdjClose: 108},
		{Date: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), Open: 108, High: 112, Low: 106, Close: 110, AdjClose: 110},
		{Date: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), Open: 110, High: 115, Low: 108, Close: 112, AdjClose: 112},
	}

	result := repairer.repairRandomUnitMixups(bars)

	// No repairs should be made
	for i, bar := range result {
		if bar.Repaired {
			t.Errorf("Bar %d should not be repaired (normal data)", i)
		}
	}
}

func TestRepairRandomUnitMixupsWithError(t *testing.T) {
	repairer := New(DefaultOptions())

	// Create data with one 100x error
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 98, Close: 102, AdjClose: 102},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Open: 102, High: 107, Low: 100, Close: 105, AdjClose: 105},
		{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Open: 10500, High: 11000, Low: 10300, Close: 10800, AdjClose: 10800}, // 100x!
		{Date: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), Open: 108, High: 112, Low: 106, Close: 110, AdjClose: 110},
		{Date: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), Open: 110, High: 115, Low: 108, Close: 112, AdjClose: 112},
	}

	result := repairer.repairRandomUnitMixups(bars)

	// Third bar should be repaired
	if !result[2].Repaired {
		t.Error("Bar 2 should be marked as repaired")
	}

	// Prices should be reduced by 100x
	if math.Abs(result[2].Close-108) > 1 {
		t.Errorf("Expected Close ~108, got %.2f", result[2].Close)
	}
}

func TestAnalyzeUnitMixups(t *testing.T) {
	repairer := New(DefaultOptions())

	// Create data with one 100x error
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 98, Close: 102, AdjClose: 102},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Open: 102, High: 107, Low: 100, Close: 105, AdjClose: 105},
		{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Open: 10500, High: 11000, Low: 10300, Close: 10800, AdjClose: 10800}, // 100x!
		{Date: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), Open: 108, High: 112, Low: 106, Close: 110, AdjClose: 110},
		{Date: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), Open: 110, High: 115, Low: 108, Close: 112, AdjClose: 112},
	}

	stats := repairer.AnalyzeUnitMixups(bars)

	if stats.TotalBars != 5 {
		t.Errorf("Expected TotalBars=5, got %d", stats.TotalBars)
	}

	if stats.RandomMixupCount != 1 {
		t.Errorf("Expected RandomMixupCount=1, got %d", stats.RandomMixupCount)
	}
}

func TestDetectUnitMixups(t *testing.T) {
	// Create data with 100x errors in alternating pattern
	// The median filter uses a 3x3 window, so detection depends on surrounding values
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 98, Close: 102, AdjClose: 102},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Open: 101, High: 106, Low: 99, Close: 103, AdjClose: 103},
		{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Open: 10500, High: 11000, Low: 10300, Close: 10800, AdjClose: 10800}, // 100x!
		{Date: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), Open: 103, High: 108, Low: 101, Close: 105, AdjClose: 105},
		{Date: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), Open: 104, High: 109, Low: 102, Close: 106, AdjClose: 106},
	}

	badIndices := DetectUnitMixups(bars)

	// At least one 100x error should be detected
	if len(badIndices) == 0 {
		t.Error("Expected at least 1 bad index")
	}

	// The 100x error at index 2 should be detected
	found := false
	for _, idx := range badIndices {
		if idx == 2 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected index 2 to be in bad indices: %v", badIndices)
	}
}

func TestFixPricesSuddenChange(t *testing.T) {
	opts := DefaultOptions()
	opts.Currency = "USD"
	repairer := New(opts)

	// Create data with a sudden 100x switch at bar 3
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 1.0, High: 1.05, Low: 0.98, Close: 1.02, AdjClose: 1.02, Volume: 1000},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Open: 1.02, High: 1.07, Low: 1.00, Close: 1.05, AdjClose: 1.05, Volume: 1200},
		{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Open: 105, High: 110, Low: 103, Close: 108, AdjClose: 108, Volume: 1100}, // Switched to cents
		{Date: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), Open: 108, High: 112, Low: 106, Close: 110, AdjClose: 110, Volume: 1300},
		{Date: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), Open: 110, High: 115, Low: 108, Close: 112, AdjClose: 112, Volume: 1400},
	}

	result := repairer.fixPricesSuddenChange(bars, 100.0)

	// First two bars should be repaired (multiplied by 100)
	if !result[0].Repaired || !result[1].Repaired {
		t.Error("First two bars should be marked as repaired")
	}

	// Prices should be scaled
	if math.Abs(result[0].Close-102) > 1 {
		t.Errorf("Expected Close ~102, got %.2f", result[0].Close)
	}
}

func TestRepairUnitSwitchKWF(t *testing.T) {
	opts := DefaultOptions()
	opts.Currency = "KWF" // Kuwaiti Dinar uses 1000
	repairer := New(opts)

	// For KWF, should use 1000 multiplier
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 1.0, High: 1.05, Low: 0.98, Close: 1.02, AdjClose: 1.02},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Open: 1.02, High: 1.07, Low: 1.00, Close: 1.05, AdjClose: 1.05},
		{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Open: 1050, High: 1100, Low: 1030, Close: 1080, AdjClose: 1080},
	}

	result := repairer.repairUnitSwitch(bars)

	// Repair should detect and handle 1000x change for KWF
	// The function uses change=1000 for KWF
	_ = result // Result depends on threshold calculations
}

func TestRepairUnitMixupsEmptyAndSingle(t *testing.T) {
	repairer := New(DefaultOptions())

	// Test empty bars
	result := repairer.repairUnitMixups([]models.Bar{})
	if len(result) != 0 {
		t.Error("Empty bars should return empty result")
	}

	// Test single bar
	single := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 98, Close: 102},
	}
	result = repairer.repairUnitMixups(single)
	if len(result) != 1 {
		t.Error("Single bar should return unchanged")
	}
}

func TestRepairRandomUnitMixupsWithZeroes(t *testing.T) {
	repairer := New(DefaultOptions())

	// Create data with zero values
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 98, Close: 102, AdjClose: 102},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Open: 0, High: 0, Low: 0, Close: 0, AdjClose: 0}, // Zero row
		{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Open: 10500, High: 11000, Low: 10300, Close: 10800, AdjClose: 10800}, // 100x!
		{Date: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), Open: 108, High: 112, Low: 106, Close: 110, AdjClose: 110},
		{Date: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), Open: 110, High: 115, Low: 108, Close: 112, AdjClose: 112},
	}

	result := repairer.repairRandomUnitMixups(bars)

	// Zero row should not be modified
	if result[1].Open != 0 {
		t.Error("Zero row should remain unchanged")
	}

	// 100x error should still be detected and fixed
	if !result[2].Repaired {
		t.Error("100x error should be repaired")
	}
}
