package models

import "time"

// CalendarType represents the type of calendar.
type CalendarType string

const (
	// CalendarEarnings represents the S&P earnings calendar.
	CalendarEarnings CalendarType = "sp_earnings"

	// CalendarIPO represents the IPO info calendar.
	CalendarIPO CalendarType = "ipo_info"

	// CalendarEconomicEvents represents the economic events calendar.
	CalendarEconomicEvents CalendarType = "economic_event"

	// CalendarSplits represents the stock splits calendar.
	CalendarSplits CalendarType = "splits"
)

// EarningsEvent represents an earnings calendar event.
type EarningsEvent struct {
	// Symbol is the ticker symbol.
	Symbol string `json:"symbol"`

	// CompanyName is the company's short name.
	CompanyName string `json:"company_name"`

	// MarketCap is the intraday market capitalization.
	MarketCap float64 `json:"market_cap,omitempty"`

	// EventName is the name of the earnings event.
	EventName string `json:"event_name,omitempty"`

	// EventTime is the event start datetime.
	EventTime *time.Time `json:"event_time,omitempty"`

	// Timing indicates if the event is before/after market (BMO/AMC).
	Timing string `json:"timing,omitempty"`

	// EPSEstimate is the estimated earnings per share.
	EPSEstimate float64 `json:"eps_estimate,omitempty"`

	// EPSActual is the actual reported earnings per share.
	EPSActual float64 `json:"eps_actual,omitempty"`

	// SurprisePercent is the earnings surprise percentage.
	SurprisePercent float64 `json:"surprise_percent,omitempty"`
}

// IPOEvent represents an IPO calendar event.
type IPOEvent struct {
	// Symbol is the ticker symbol.
	Symbol string `json:"symbol"`

	// CompanyName is the company's short name.
	CompanyName string `json:"company_name"`

	// Exchange is the exchange short name.
	Exchange string `json:"exchange,omitempty"`

	// FilingDate is the SEC filing date.
	FilingDate *time.Time `json:"filing_date,omitempty"`

	// Date is the IPO date.
	Date *time.Time `json:"date,omitempty"`

	// AmendedDate is the amended filing date.
	AmendedDate *time.Time `json:"amended_date,omitempty"`

	// PriceFrom is the lower end of the price range.
	PriceFrom float64 `json:"price_from,omitempty"`

	// PriceTo is the upper end of the price range.
	PriceTo float64 `json:"price_to,omitempty"`

	// OfferPrice is the final offer price.
	OfferPrice float64 `json:"offer_price,omitempty"`

	// Currency is the currency name.
	Currency string `json:"currency,omitempty"`

	// Shares is the number of shares offered.
	Shares int64 `json:"shares,omitempty"`

	// DealType is the type of deal.
	DealType string `json:"deal_type,omitempty"`
}

// EconomicEvent represents an economic calendar event.
type EconomicEvent struct {
	// Event is the economic release name.
	Event string `json:"event"`

	// Region is the country code.
	Region string `json:"region,omitempty"`

	// EventTime is the event start datetime.
	EventTime *time.Time `json:"event_time,omitempty"`

	// Period is the reporting period.
	Period string `json:"period,omitempty"`

	// Actual is the actual released value.
	Actual float64 `json:"actual,omitempty"`

	// Expected is the consensus estimate.
	Expected float64 `json:"expected,omitempty"`

	// Last is the prior release actual value.
	Last float64 `json:"last,omitempty"`

	// Revised is the revised value from original report.
	Revised float64 `json:"revised,omitempty"`
}

// CalendarSplitEvent represents a stock split calendar event.
type CalendarSplitEvent struct {
	// Symbol is the ticker symbol.
	Symbol string `json:"symbol"`

	// CompanyName is the company's short name.
	CompanyName string `json:"company_name"`

	// PayableDate is the split payable date.
	PayableDate *time.Time `json:"payable_date,omitempty"`

	// Optionable indicates if the stock is optionable.
	Optionable bool `json:"optionable"`

	// OldShareWorth is the old share worth.
	OldShareWorth float64 `json:"old_share_worth,omitempty"`

	// NewShareWorth is the new share worth after split.
	NewShareWorth float64 `json:"new_share_worth,omitempty"`

	// Ratio represents the split ratio (e.g., "2:1").
	Ratio string `json:"ratio,omitempty"`
}

// CalendarResponse represents the raw API response for calendar data.
type CalendarResponse struct {
	Finance struct {
		Result []struct {
			Documents []struct {
				Columns []struct {
					Label string `json:"label"`
					Type  string `json:"type"`
				} `json:"columns"`
				Rows [][]interface{} `json:"rows"`
			} `json:"documents"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error,omitempty"`
	} `json:"finance"`
}

// CalendarOptions contains options for calendar queries.
type CalendarOptions struct {
	// Start is the start date for the calendar range.
	Start time.Time

	// End is the end date for the calendar range.
	End time.Time

	// Limit is the maximum number of results (max 100).
	Limit int

	// Offset is the pagination offset.
	Offset int
}

// DefaultCalendarOptions returns default calendar options.
// Default range is today to 7 days from now, limit 12.
func DefaultCalendarOptions() CalendarOptions {
	now := time.Now()
	return CalendarOptions{
		Start:  now,
		End:    now.AddDate(0, 0, 7),
		Limit:  12,
		Offset: 0,
	}
}
