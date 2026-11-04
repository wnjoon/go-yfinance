package models

import "time"

// DownloadParams represents parameters for downloading multiple tickers.
//
// Example:
//
//	params := models.DownloadParams{
//	    Symbols:  []string{"AAPL", "MSFT", "GOOGL"},
//	    Period:   "1mo",
//	    Interval: "1d",
//	}
type DownloadParams struct {
	// Symbols is the list of ticker symbols to download.
	Symbols []string

	// Period is the data period (1d, 5d, 1mo, 3mo, 6mo, 1y, 2y, 5y, 10y, ytd, max).
	// Either use Period or Start/End.
	Period string

	// Interval is the data interval (1m, 2m, 5m, 15m, 30m, 60m, 90m, 1h, 1d, 5d, 1wk, 1mo, 3mo).
	Interval string

	// Start is the start date.
	Start *time.Time

	// End is the end date.
	End *time.Time

	// PrePost includes pre and post market data.
	PrePost bool

	// Actions includes dividend and stock split data.
	Actions bool

	// AutoAdjust adjusts OHLC for splits and dividends.
	AutoAdjust bool

	// Threads is the number of concurrent downloads.
	// 0 or 1 means sequential, >1 means parallel.
	Threads int

	// Timeout is the request timeout in seconds.
	Timeout int
}

// DefaultDownloadParams returns default download parameters.
func DefaultDownloadParams() DownloadParams {
	return DownloadParams{
		Period:     "1mo",
		Interval:   "1d",
		AutoAdjust: true,
		Threads:    0, // Sequential by default for Go
		Timeout:    10,
	}
}

// MultiTickerResult represents the result of downloading multiple tickers.
type MultiTickerResult struct {
	// Data contains the history data for each ticker.
	// Key is the ticker symbol.
	Data map[string][]Bar

	// Errors contains any errors that occurred during download.
	// Key is the ticker symbol.
	Errors map[string]error

	// Symbols is the list of successfully downloaded symbols.
	Symbols []string
}

// Get returns the history data for a specific ticker.
func (r *MultiTickerResult) Get(symbol string) []Bar {
	if r.Data == nil {
		return nil
	}
	return r.Data[symbol]
}

// HasErrors returns true if any ticker had an error.
func (r *MultiTickerResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// SuccessCount returns the number of successfully downloaded tickers.
func (r *MultiTickerResult) SuccessCount() int {
	return len(r.Symbols)
}

// ErrorCount returns the number of tickers that had errors.
func (r *MultiTickerResult) ErrorCount() int {
	return len(r.Errors)
}

// TickersInfoResult is the result of fetching Info for every symbol of a
// multi.Tickers collection (Python yfinance Tickers.info / get_info).
//
// Every requested symbol appears in exactly one of Data or Errors.
type TickersInfoResult struct {
	// Data maps each successfully fetched symbol to its Info.
	Data map[string]*Info

	// Errors maps each failed or skipped symbol to its error.
	Errors map[string]error

	// Symbols lists the successfully fetched symbols in Tickers order.
	Symbols []string
}

// Get returns the Info for a symbol, or nil if it was not fetched.
func (r *TickersInfoResult) Get(symbol string) *Info {
	if r == nil || r.Data == nil {
		return nil
	}
	return r.Data[symbol]
}

// HasErrors returns true if any symbol failed or was skipped.
func (r *TickersInfoResult) HasErrors() bool {
	return r != nil && len(r.Errors) > 0
}

// SuccessCount returns the number of successfully fetched symbols.
func (r *TickersInfoResult) SuccessCount() int {
	if r == nil {
		return 0
	}
	return len(r.Symbols)
}

// ErrorCount returns the number of failed or skipped symbols.
func (r *TickersInfoResult) ErrorCount() int {
	if r == nil {
		return 0
	}
	return len(r.Errors)
}

// TickersCalendarResult is the result of fetching Calendar for every symbol
// of a multi.Tickers collection (Python yfinance Tickers.calendar /
// get_calendar).
//
// Python returns one DataFrame row per ticker with a NaN row for failures;
// Go returns one Calendar per successful symbol and the error for the rest.
// Every requested symbol appears in exactly one of Data or Errors.
type TickersCalendarResult struct {
	// Data maps each successfully fetched symbol to its Calendar.
	Data map[string]*Calendar

	// Errors maps each failed or skipped symbol to its error.
	Errors map[string]error

	// Symbols lists the successfully fetched symbols in Tickers order.
	Symbols []string
}

// Get returns the Calendar for a symbol, or nil if it was not fetched.
func (r *TickersCalendarResult) Get(symbol string) *Calendar {
	if r == nil || r.Data == nil {
		return nil
	}
	return r.Data[symbol]
}

// HasErrors returns true if any symbol failed or was skipped.
func (r *TickersCalendarResult) HasErrors() bool {
	return r != nil && len(r.Errors) > 0
}

// SuccessCount returns the number of successfully fetched symbols.
func (r *TickersCalendarResult) SuccessCount() int {
	if r == nil {
		return 0
	}
	return len(r.Symbols)
}

// ErrorCount returns the number of failed or skipped symbols.
func (r *TickersCalendarResult) ErrorCount() int {
	if r == nil {
		return 0
	}
	return len(r.Errors)
}
