package ticker

import (
	"fmt"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

// Calendar returns upcoming calendar events for the ticker.
//
// This includes dividend dates, earnings dates, and earnings estimates.
//
// Example:
//
//	calendar, err := ticker.Calendar()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if calendar.ExDividendDate != nil {
//	    fmt.Printf("Ex-Dividend: %s\n", calendar.ExDividendDate.Format("2006-01-02"))
//	}
//	for _, date := range calendar.EarningsDate {
//	    fmt.Printf("Earnings: %s\n", date.Format("2006-01-02"))
//	}
func (t *Ticker) Calendar() (*models.Calendar, error) {
	t.mu.RLock()
	if t.calendarCache != nil {
		defer t.mu.RUnlock()
		return t.calendarCache, nil
	}
	t.mu.RUnlock()

	data, err := t.fetchQuoteSummary([]string{"calendarEvents"})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch calendar data: %w", err)
	}

	calendar, err := t.parseCalendar(data)
	if err != nil {
		return nil, err
	}

	t.mu.Lock()
	t.calendarCache = calendar
	t.mu.Unlock()

	return calendar, nil
}

// parseCalendar parses calendarEvents data.
func (t *Ticker) parseCalendar(data map[string]interface{}) (*models.Calendar, error) {
	events, ok := data["calendarEvents"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("calendarEvents not found")
	}

	calendar := &models.Calendar{}

	// Parse dividend dates
	if dividendDate := getNestedFloat(events, "dividendDate"); dividendDate > 0 {
		t := time.Unix(int64(dividendDate), 0)
		calendar.DividendDate = &t
	}

	if exDividendDate := getNestedFloat(events, "exDividendDate"); exDividendDate > 0 {
		t := time.Unix(int64(exDividendDate), 0)
		calendar.ExDividendDate = &t
	}

	// Parse earnings data
	if earnings, ok := events["earnings"].(map[string]interface{}); ok {
		// Parse earnings dates
		if earningsDates, ok := earnings["earningsDate"].([]interface{}); ok {
			for _, dateVal := range earningsDates {
				// Could be a map with "raw" key or a direct value
				var timestamp float64
				switch v := dateVal.(type) {
				case float64:
					timestamp = v
				case map[string]interface{}:
					if raw, ok := v["raw"].(float64); ok {
						timestamp = raw
					}
				}
				if timestamp > 0 {
					calendar.EarningsDate = append(calendar.EarningsDate, time.Unix(int64(timestamp), 0))
				}
			}
		}

		// Parse earnings estimates
		if high := getNestedFloatPtr(earnings, "earningsHigh"); high != nil {
			calendar.EarningsHigh = high
		}
		if low := getNestedFloatPtr(earnings, "earningsLow"); low != nil {
			calendar.EarningsLow = low
		}
		if avg := getNestedFloatPtr(earnings, "earningsAverage"); avg != nil {
			calendar.EarningsAverage = avg
		}

		// Parse revenue estimates
		if high := getNestedFloatPtr(earnings, "revenueHigh"); high != nil {
			calendar.RevenueHigh = high
		}
		if low := getNestedFloatPtr(earnings, "revenueLow"); low != nil {
			calendar.RevenueLow = low
		}
		if avg := getNestedFloatPtr(earnings, "revenueAverage"); avg != nil {
			calendar.RevenueAverage = avg
		}
	}

	return calendar, nil
}
