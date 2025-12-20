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
