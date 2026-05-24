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
		parseEarningsCalendar(calendar, earnings)
	}

	return calendar, nil
}

func parseEarningsCalendar(calendar *models.Calendar, earnings map[string]interface{}) {
	calendar.EarningsDate = parseEarningsDates(earnings["earningsDate"])
	assignCalendarEstimate(&calendar.EarningsHigh, earnings, "earningsHigh")
	assignCalendarEstimate(&calendar.EarningsLow, earnings, "earningsLow")
	assignCalendarEstimate(&calendar.EarningsAverage, earnings, "earningsAverage")
	assignCalendarEstimate(&calendar.RevenueHigh, earnings, "revenueHigh")
	assignCalendarEstimate(&calendar.RevenueLow, earnings, "revenueLow")
	assignCalendarEstimate(&calendar.RevenueAverage, earnings, "revenueAverage")
}

func parseEarningsDates(value interface{}) []time.Time {
	earningsDates, ok := value.([]interface{})
	if !ok {
		return nil
	}
	dates := make([]time.Time, 0, len(earningsDates))
	for _, dateVal := range earningsDates {
		if timestamp := calendarTimestamp(dateVal); timestamp > 0 {
			dates = append(dates, time.Unix(int64(timestamp), 0))
		}
	}
	return dates
}

func calendarTimestamp(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case map[string]interface{}:
		raw, _ := v["raw"].(float64)
		return raw
	default:
		return 0
	}
}

func assignCalendarEstimate(target **float64, earnings map[string]interface{}, key string) {
	if value := getNestedFloatPtr(earnings, key); value != nil {
		*target = value
	}
}
