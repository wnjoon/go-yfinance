package models

import "time"

// Bar represents a single OHLCV bar (candlestick).
type Bar struct {
	Date      time.Time `json:"date"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	AdjClose  float64   `json:"adjClose"`
	Volume    int64     `json:"volume"`
	Dividends float64   `json:"dividends,omitempty"`
	Splits    float64   `json:"splits,omitempty"`
}

// History represents historical price data.
type History struct {
	Symbol   string `json:"symbol"`
	Currency string `json:"currency"`
	Bars     []Bar  `json:"bars"`
}

// HistoryParams represents parameters for fetching historical data.
type HistoryParams struct {
	// Period: 1d, 5d, 1mo, 3mo, 6mo, 1y, 2y, 5y, 10y, ytd, max
	Period string `json:"period,omitempty"`

	// Interval: 1m, 2m, 5m, 15m, 30m, 60m, 90m, 1h, 1d, 5d, 1wk, 1mo, 3mo
	Interval string `json:"interval,omitempty"`

	// Start date (YYYY-MM-DD or time.Time)
	Start *time.Time `json:"start,omitempty"`

	// End date (YYYY-MM-DD or time.Time)
	End *time.Time `json:"end,omitempty"`

	// Include pre/post market data
	PrePost bool `json:"prepost,omitempty"`

	// Automatically adjust OHLC for splits/dividends
	AutoAdjust bool `json:"autoAdjust,omitempty"`

	// Include dividend and split events
	Actions bool `json:"actions,omitempty"`

	// Repair bad data (100x errors, missing data)
	Repair bool `json:"repair,omitempty"`

	// Keep NaN rows
	KeepNA bool `json:"keepna,omitempty"`
}

// DefaultHistoryParams returns default history parameters.
func DefaultHistoryParams() HistoryParams {
	return HistoryParams{
		Period:     "1mo",
		Interval:   "1d",
		PrePost:    false,
		AutoAdjust: true,
		Actions:    true,
		Repair:     false,
		KeepNA:     false,
	}
}

// ChartMeta represents metadata from chart API response.
type ChartMeta struct {
	Currency             string   `json:"currency"`
	Symbol               string   `json:"symbol"`
	ExchangeName         string   `json:"exchangeName"`
	ExchangeTimezoneName string   `json:"exchangeTimezoneName"`
	InstrumentType       string   `json:"instrumentType"`
	FirstTradeDate       int64    `json:"firstTradeDate"`
	RegularMarketTime    int64    `json:"regularMarketTime"`
	GMTOffset            int      `json:"gmtoffset"`
	Timezone             string   `json:"timezone"`
	RegularMarketPrice   float64  `json:"regularMarketPrice"`
	ChartPreviousClose   float64  `json:"chartPreviousClose"`
	PreviousClose        float64  `json:"previousClose"`
	Scale                int      `json:"scale"`
	PriceHint            int      `json:"priceHint"`
	DataGranularity      string   `json:"dataGranularity"`
	Range                string   `json:"range"`
	ValidRanges          []string `json:"validRanges"`
}

// Dividend represents a dividend payment.
type Dividend struct {
	Date   time.Time `json:"date"`
	Amount float64   `json:"amount"`
}

// Split represents a stock split.
type Split struct {
	Date       time.Time `json:"date"`
	Numerator  float64   `json:"numerator"`
	Denominator float64   `json:"denominator"`
	Ratio      string    `json:"ratio"` // e.g., "4:1"
}

// Actions represents dividend and split actions.
type Actions struct {
	Dividends []Dividend `json:"dividends,omitempty"`
	Splits    []Split    `json:"splits,omitempty"`
}

// ValidPeriods returns all valid period values.
func ValidPeriods() []string {
	return []string{"1d", "5d", "1mo", "3mo", "6mo", "1y", "2y", "5y", "10y", "ytd", "max"}
}

// ValidIntervals returns all valid interval values.
func ValidIntervals() []string {
	return []string{"1m", "2m", "5m", "15m", "30m", "60m", "90m", "1h", "1d", "5d", "1wk", "1mo", "3mo"}
}

// IsValidPeriod checks if a period string is valid.
func IsValidPeriod(period string) bool {
	for _, p := range ValidPeriods() {
		if p == period {
			return true
		}
	}
	return false
}

// IsValidInterval checks if an interval string is valid.
func IsValidInterval(interval string) bool {
	for _, i := range ValidIntervals() {
		if i == interval {
			return true
		}
	}
	return false
}
