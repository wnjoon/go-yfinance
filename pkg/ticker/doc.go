// Package ticker provides the main interface for accessing Yahoo Finance data.
//
// # Overview
//
// The ticker package is the primary entry point for go-yfinance. It provides
// a [Ticker] type that represents a single stock, ETF, or fund, and offers
// methods to fetch various types of financial data.
//
// # Quick Start
//
//	import "github.com/wnjoon/go-yfinance/pkg/ticker"
//
//	// Create a ticker
//	t, err := ticker.New("AAPL")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer t.Close()
//
//	// Get current quote
//	quote, err := t.Quote()
//
//	// Get historical data
//	history, err := t.History(models.HistoryParams{
//	    Period:   "1mo",
//	    Interval: "1d",
//	})
//
// # Available Data
//
// The Ticker type provides methods for:
//
//   - [Ticker.Quote]: Real-time quote data
//   - [Ticker.History]: Historical OHLCV data
//   - [Ticker.Info]: Company information and key statistics
//   - [Ticker.FastInfo]: Quick-access subset of info data
//   - [Ticker.Dividends]: Dividend history
//   - [Ticker.Splits]: Stock split history
//   - [Ticker.Actions]: Combined dividends and splits
//   - [Ticker.Options]: Available option expiration dates
//   - [Ticker.OptionChain]: Full option chain data
//   - [Ticker.IncomeStatement]: Income statement data
//   - [Ticker.BalanceSheet]: Balance sheet data
//   - [Ticker.CashFlow]: Cash flow statement data
//   - [Ticker.Recommendations]: Analyst recommendations
//   - [Ticker.AnalystPriceTargets]: Analyst price targets
//   - [Ticker.EarningsEstimate]: Earnings estimates
//   - [Ticker.RevenueEstimate]: Revenue estimates
//   - [Ticker.EPSTrend]: EPS trend data
//   - [Ticker.EPSRevisions]: EPS revision data
//   - [Ticker.EarningsHistory]: Historical earnings data
//   - [Ticker.GrowthEstimates]: Growth estimates
//
// # Caching
//
// The Ticker automatically caches API responses to minimize redundant requests.
// Use [Ticker.ClearCache] to force a refresh of cached data.
//
// # Thread Safety
//
// All Ticker methods are safe for concurrent use from multiple goroutines.
package ticker
