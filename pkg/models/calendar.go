// Package models provides data structures for Yahoo Finance API responses.
package models

import "time"

// Calendar represents upcoming calendar events for a ticker.
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
type Calendar struct {
	// DividendDate is the next dividend payment date.
	DividendDate *time.Time `json:"dividendDate,omitempty"`

	// ExDividendDate is the ex-dividend date (must own before this date).
	ExDividendDate *time.Time `json:"exDividendDate,omitempty"`

	// EarningsDate contains the expected earnings announcement date(s).
	// Often contains a range (start and end date).
	EarningsDate []time.Time `json:"earningsDate,omitempty"`

	// EarningsHigh is the highest earnings estimate.
	EarningsHigh *float64 `json:"earningsHigh,omitempty"`

	// EarningsLow is the lowest earnings estimate.
	EarningsLow *float64 `json:"earningsLow,omitempty"`

	// EarningsAverage is the average earnings estimate.
	EarningsAverage *float64 `json:"earningsAverage,omitempty"`

	// RevenueHigh is the highest revenue estimate.
	RevenueHigh *float64 `json:"revenueHigh,omitempty"`

	// RevenueLow is the lowest revenue estimate.
	RevenueLow *float64 `json:"revenueLow,omitempty"`

	// RevenueAverage is the average revenue estimate.
	RevenueAverage *float64 `json:"revenueAverage,omitempty"`
}

// HasEarnings returns true if earnings data is available.
func (c *Calendar) HasEarnings() bool {
	return len(c.EarningsDate) > 0 ||
		c.EarningsHigh != nil ||
		c.EarningsLow != nil ||
		c.EarningsAverage != nil
}

// HasDividend returns true if dividend data is available.
func (c *Calendar) HasDividend() bool {
	return c.DividendDate != nil || c.ExDividendDate != nil
}

// NextEarningsDate returns the earliest earnings date, or nil if none.
func (c *Calendar) NextEarningsDate() *time.Time {
	if len(c.EarningsDate) == 0 {
		return nil
	}
	earliest := c.EarningsDate[0]
	for _, d := range c.EarningsDate[1:] {
		if d.Before(earliest) {
			earliest = d
		}
	}
	return &earliest
}
