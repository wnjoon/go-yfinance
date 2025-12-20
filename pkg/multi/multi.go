// Package multi provides functionality for downloading data for multiple tickers.
//
// # Overview
//
// The multi package allows you to efficiently download historical data for
// multiple stock symbols at once, with optional parallel processing.
//
// # Basic Usage
//
//	result, err := multi.Download([]string{"AAPL", "MSFT", "GOOGL"}, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for symbol, bars := range result.Data {
//	    fmt.Printf("%s: %d bars\n", symbol, len(bars))
//	}
//
// # With Parameters
//
//	params := &models.DownloadParams{
//	    Period:   "3mo",
//	    Interval: "1d",
//	    Threads:  4,
//	}
//	result, err := multi.Download([]string{"AAPL", "MSFT"}, params)
//
// # Tickers Class
//
//	tickers, err := multi.NewTickers([]string{"AAPL", "MSFT", "GOOGL"})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer tickers.Close()
//
//	// Access individual ticker
//	aapl := tickers.Get("AAPL")
//	info, _ := aapl.Info()
//
//	// Download history for all
//	result, _ := tickers.History(nil)
//
// # Thread Safety
//
// All multi package functions are safe for concurrent use.
package multi

import (
	"fmt"
	"strings"
	"sync"

	"github.com/wnjoon/go-yfinance/pkg/client"
	"github.com/wnjoon/go-yfinance/pkg/models"
	"github.com/wnjoon/go-yfinance/pkg/ticker"
)

// Tickers represents a collection of multiple ticker symbols.
type Tickers struct {
	symbols    []string
	tickers    map[string]*ticker.Ticker
	client     *client.Client
	ownsClient bool
	mu         sync.RWMutex
}

// Option is a function that configures Tickers.
type Option func(*Tickers)

// WithClient sets a custom HTTP client.
func WithClient(c *client.Client) Option {
	return func(t *Tickers) {
		t.client = c
		t.ownsClient = false
	}
}

// NewTickers creates a new Tickers instance for multiple symbols.
//
// Symbols can be provided as a slice of strings.
//
// Example:
//
//	tickers, err := multi.NewTickers([]string{"AAPL", "MSFT", "GOOGL"})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer tickers.Close()
func NewTickers(symbols []string, opts ...Option) (*Tickers, error) {
	if len(symbols) == 0 {
		return nil, fmt.Errorf("at least one symbol is required")
	}

	t := &Tickers{
		symbols:    make([]string, 0, len(symbols)),
		tickers:    make(map[string]*ticker.Ticker),
		ownsClient: true,
	}

	for _, opt := range opts {
		opt(t)
	}

	// Create shared client if not provided
	if t.client == nil {
		c, err := client.New()
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}
		t.client = c
	}

	// Normalize and store symbols
	for _, sym := range symbols {
		sym = strings.TrimSpace(strings.ToUpper(sym))
		if sym == "" {
			continue
		}
		t.symbols = append(t.symbols, sym)

		// Create ticker with shared client
		tkr, err := ticker.New(sym, ticker.WithClient(t.client))
		if err != nil {
			// Clean up already created tickers
			t.Close()
			return nil, fmt.Errorf("failed to create ticker for %s: %w", sym, err)
		}
		t.tickers[sym] = tkr
	}

	if len(t.symbols) == 0 {
		return nil, fmt.Errorf("no valid symbols provided")
	}

	return t, nil
}

// NewTickersFromString creates Tickers from a space or comma separated string.
//
// Example:
//
//	tickers, err := multi.NewTickersFromString("AAPL MSFT GOOGL")
//	// or
//	tickers, err := multi.NewTickersFromString("AAPL,MSFT,GOOGL")
func NewTickersFromString(tickerStr string, opts ...Option) (*Tickers, error) {
	// Replace commas with spaces and split
	tickerStr = strings.ReplaceAll(tickerStr, ",", " ")
	symbols := strings.Fields(tickerStr)
	return NewTickers(symbols, opts...)
}

// Close releases all resources used by Tickers.
func (t *Tickers) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Close all individual tickers (they don't own the client)
	for _, tkr := range t.tickers {
		tkr.Close()
	}

	// Close the shared client if we own it
	if t.ownsClient && t.client != nil {
		t.client.Close()
	}
}

// Symbols returns the list of ticker symbols.
func (t *Tickers) Symbols() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return append([]string{}, t.symbols...)
}

// Get returns the Ticker instance for a specific symbol.
//
// Returns nil if the symbol is not found.
func (t *Tickers) Get(symbol string) *ticker.Ticker {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.tickers[strings.ToUpper(symbol)]
}

// Count returns the number of tickers.
func (t *Tickers) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.symbols)
}

// History downloads historical data for all tickers.
//
// Example:
//
//	result, err := tickers.History(&models.DownloadParams{
//	    Period:   "1mo",
//	    Interval: "1d",
//	})
func (t *Tickers) History(params *models.DownloadParams) (*models.MultiTickerResult, error) {
	if params == nil {
		defaultParams := models.DefaultDownloadParams()
		params = &defaultParams
	}
	params.Symbols = t.Symbols()
	return download(t.tickers, params)
}

// Download downloads historical data for all tickers with default parameters.
func (t *Tickers) Download() (*models.MultiTickerResult, error) {
	return t.History(nil)
}

// download performs the actual download for multiple tickers.
func download(tickers map[string]*ticker.Ticker, params *models.DownloadParams) (*models.MultiTickerResult, error) {
	result := &models.MultiTickerResult{
		Data:    make(map[string][]models.Bar),
		Errors:  make(map[string]error),
		Symbols: make([]string, 0),
	}

	if len(params.Symbols) == 0 {
		return result, nil
	}

	// Build history params
	histParams := models.HistoryParams{
		Period:     params.Period,
		Interval:   params.Interval,
		PrePost:    params.PrePost,
		AutoAdjust: params.AutoAdjust,
	}

	histParams.Start = params.Start
	histParams.End = params.End

	// Determine concurrency
	threads := params.Threads
	if threads <= 0 {
		threads = 1 // Sequential
	}

	if threads == 1 {
		// Sequential download
		for _, symbol := range params.Symbols {
			tkr := tickers[symbol]
			if tkr == nil {
				result.Errors[symbol] = fmt.Errorf("ticker not found: %s", symbol)
				continue
			}

			bars, err := tkr.History(histParams)
			if err != nil {
				result.Errors[symbol] = err
			} else {
				result.Data[symbol] = bars
				result.Symbols = append(result.Symbols, symbol)
			}
		}
	} else {
		// Parallel download with worker pool
		type downloadResult struct {
			symbol string
			bars   []models.Bar
			err    error
		}

		symbolChan := make(chan string, len(params.Symbols))
		resultChan := make(chan downloadResult, len(params.Symbols))

		// Start workers
		var wg sync.WaitGroup
		workerCount := threads
		if workerCount > len(params.Symbols) {
			workerCount = len(params.Symbols)
		}

		for i := 0; i < workerCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for symbol := range symbolChan {
					tkr := tickers[symbol]
					if tkr == nil {
						resultChan <- downloadResult{
							symbol: symbol,
							err:    fmt.Errorf("ticker not found: %s", symbol),
						}
						continue
					}

					bars, err := tkr.History(histParams)
					resultChan <- downloadResult{
						symbol: symbol,
						bars:   bars,
						err:    err,
					}
				}
			}()
		}

		// Send symbols to workers
		for _, symbol := range params.Symbols {
			symbolChan <- symbol
		}
		close(symbolChan)

		// Wait for workers to finish
		go func() {
			wg.Wait()
			close(resultChan)
		}()

		// Collect results
		for res := range resultChan {
			if res.err != nil {
				result.Errors[res.symbol] = res.err
			} else {
				result.Data[res.symbol] = res.bars
				result.Symbols = append(result.Symbols, res.symbol)
			}
		}
	}

	return result, nil
}

// Download is a convenience function to download data for multiple tickers.
//
// Example:
//
//	result, err := multi.Download([]string{"AAPL", "MSFT"}, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for symbol, bars := range result.Data {
//	    fmt.Printf("%s: %d bars\n", symbol, len(bars))
//	}
func Download(symbols []string, params *models.DownloadParams) (*models.MultiTickerResult, error) {
	tickers, err := NewTickers(symbols)
	if err != nil {
		return nil, err
	}
	defer tickers.Close()

	return tickers.History(params)
}

// DownloadString is like Download but accepts a space/comma separated string.
//
// Example:
//
//	result, err := multi.DownloadString("AAPL MSFT GOOGL", nil)
func DownloadString(tickerStr string, params *models.DownloadParams) (*models.MultiTickerResult, error) {
	tickers, err := NewTickersFromString(tickerStr)
	if err != nil {
		return nil, err
	}
	defer tickers.Close()

	return tickers.History(params)
}
