package ticker

import (
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func TestCalendarModel(t *testing.T) {
	now := time.Now()
	earningsDate1 := time.Date(2024, 4, 25, 0, 0, 0, 0, time.UTC)
	earningsDate2 := time.Date(2024, 4, 26, 0, 0, 0, 0, time.UTC)
	earningsHigh := 2.50
	earningsLow := 2.30
	earningsAvg := 2.40

	calendar := models.Calendar{
		DividendDate:    &now,
		ExDividendDate:  &now,
		EarningsDate:    []time.Time{earningsDate1, earningsDate2},
		EarningsHigh:    &earningsHigh,
		EarningsLow:     &earningsLow,
		EarningsAverage: &earningsAvg,
	}

	if calendar.DividendDate == nil {
		t.Error("Expected DividendDate to be set")
	}

	if len(calendar.EarningsDate) != 2 {
		t.Errorf("Expected 2 earnings dates, got %d", len(calendar.EarningsDate))
	}

	if *calendar.EarningsHigh != 2.50 {
		t.Errorf("Expected EarningsHigh 2.50, got %f", *calendar.EarningsHigh)
	}
}

func TestCalendarHasEarnings(t *testing.T) {
	tests := []struct {
		name     string
		calendar models.Calendar
		expected bool
	}{
		{
			name:     "empty calendar",
			calendar: models.Calendar{},
			expected: false,
		},
		{
			name: "has earnings date",
			calendar: models.Calendar{
				EarningsDate: []time.Time{time.Now()},
			},
			expected: true,
		},
		{
			name: "has earnings high",
			calendar: models.Calendar{
				EarningsHigh: ptrFloat64(2.50),
			},
			expected: true,
		},
		{
			name: "has earnings low",
			calendar: models.Calendar{
				EarningsLow: ptrFloat64(2.30),
			},
			expected: true,
		},
		{
			name: "has earnings average",
			calendar: models.Calendar{
				EarningsAverage: ptrFloat64(2.40),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.calendar.HasEarnings()
			if got != tt.expected {
				t.Errorf("HasEarnings() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCalendarHasDividend(t *testing.T) {
	tests := []struct {
		name     string
		calendar models.Calendar
		expected bool
	}{
		{
			name:     "empty calendar",
			calendar: models.Calendar{},
			expected: false,
		},
		{
			name: "has dividend date",
			calendar: models.Calendar{
				DividendDate: ptrTime(time.Now()),
			},
			expected: true,
		},
		{
			name: "has ex-dividend date",
			calendar: models.Calendar{
				ExDividendDate: ptrTime(time.Now()),
			},
			expected: true,
		},
		{
			name: "has both dates",
			calendar: models.Calendar{
				DividendDate:   ptrTime(time.Now()),
				ExDividendDate: ptrTime(time.Now()),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.calendar.HasDividend()
			if got != tt.expected {
				t.Errorf("HasDividend() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCalendarNextEarningsDate(t *testing.T) {
	tests := []struct {
		name     string
		calendar models.Calendar
		expected *time.Time
	}{
		{
			name:     "empty earnings dates",
			calendar: models.Calendar{},
			expected: nil,
		},
		{
			name: "single earnings date",
			calendar: models.Calendar{
				EarningsDate: []time.Time{
					time.Date(2024, 4, 25, 0, 0, 0, 0, time.UTC),
				},
			},
			expected: ptrTime(time.Date(2024, 4, 25, 0, 0, 0, 0, time.UTC)),
		},
		{
			name: "multiple earnings dates - returns earliest",
			calendar: models.Calendar{
				EarningsDate: []time.Time{
					time.Date(2024, 4, 26, 0, 0, 0, 0, time.UTC),
					time.Date(2024, 4, 24, 0, 0, 0, 0, time.UTC),
					time.Date(2024, 4, 25, 0, 0, 0, 0, time.UTC),
				},
			},
			expected: ptrTime(time.Date(2024, 4, 24, 0, 0, 0, 0, time.UTC)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.calendar.NextEarningsDate()
			if tt.expected == nil {
				if got != nil {
					t.Errorf("NextEarningsDate() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Errorf("NextEarningsDate() = nil, want %v", tt.expected)
				} else if !got.Equal(*tt.expected) {
					t.Errorf("NextEarningsDate() = %v, want %v", got, tt.expected)
				}
			}
		})
	}
}

func TestCalendarCacheInitialization(t *testing.T) {
	tkr, err := New("AAPL")
	if err != nil {
		t.Fatalf("Failed to create ticker: %v", err)
	}
	defer tkr.Close()

	if tkr.calendarCache != nil {
		t.Error("Expected calendarCache to be nil initially")
	}
}

func TestClearCacheIncludesCalendar(t *testing.T) {
	tkr, err := New("AAPL")
	if err != nil {
		t.Fatalf("Failed to create ticker: %v", err)
	}
	defer tkr.Close()

	// Manually set calendarCache
	tkr.calendarCache = &models.Calendar{
		EarningsDate: []time.Time{time.Now()},
	}

	// Clear cache
	tkr.ClearCache()

	if tkr.calendarCache != nil {
		t.Error("Expected calendarCache to be nil after ClearCache")
	}
}

// Helper functions
func ptrFloat64(v float64) *float64 {
	return &v
}

func ptrTime(v time.Time) *time.Time {
	return &v
}

// Integration test - commented out for CI, run manually
// func TestCalendarLive(t *testing.T) {
// 	tkr, err := New("AAPL")
// 	if err != nil {
// 		t.Fatalf("Failed to create ticker: %v", err)
// 	}
// 	defer tkr.Close()
//
// 	calendar, err := tkr.Calendar()
// 	if err != nil {
// 		t.Fatalf("Failed to get calendar: %v", err)
// 	}
//
// 	t.Logf("Calendar for AAPL:")
//
// 	if calendar.DividendDate != nil {
// 		t.Logf("  Dividend Date: %s", calendar.DividendDate.Format("2006-01-02"))
// 	}
//
// 	if calendar.ExDividendDate != nil {
// 		t.Logf("  Ex-Dividend Date: %s", calendar.ExDividendDate.Format("2006-01-02"))
// 	}
//
// 	if len(calendar.EarningsDate) > 0 {
// 		t.Logf("  Earnings Dates:")
// 		for _, d := range calendar.EarningsDate {
// 			t.Logf("    - %s", d.Format("2006-01-02"))
// 		}
// 	}
//
// 	if calendar.EarningsAverage != nil {
// 		t.Logf("  Earnings Estimate: Avg=$%.2f (High=$%.2f, Low=$%.2f)",
// 			*calendar.EarningsAverage,
// 			getOrZero(calendar.EarningsHigh),
// 			getOrZero(calendar.EarningsLow))
// 	}
//
// 	if calendar.RevenueAverage != nil {
// 		t.Logf("  Revenue Estimate: Avg=$%.0f (High=$%.0f, Low=$%.0f)",
// 			*calendar.RevenueAverage,
// 			getOrZero(calendar.RevenueHigh),
// 			getOrZero(calendar.RevenueLow))
// 	}
//
// 	t.Logf("  Has Earnings: %v", calendar.HasEarnings())
// 	t.Logf("  Has Dividend: %v", calendar.HasDividend())
// }
//
// func getOrZero(p *float64) float64 {
// 	if p == nil {
// 		return 0
// 	}
// 	return *p
// }
