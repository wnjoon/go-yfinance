// Package ticker provides the main Ticker interface for accessing Yahoo Finance data.
package ticker

import (
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/wnjoon/go-yfinance/internal/endpoints"
	"github.com/wnjoon/go-yfinance/pkg/client"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

// Ticker represents a single stock/ETF/fund ticker.
type Ticker struct {
	symbol string

	// HTTP client and authentication
	client *client.Client
	auth   *client.AuthManager

	// Cached data
	mu           sync.RWMutex
	infoCache    *models.Info
	quoteCache   *models.Quote
	historyMeta  *models.ChartMeta
	optionsCache *optionsCache

	// Ownership tracking for cleanup
	ownsClient bool
}

// Option is a function that configures a Ticker.
type Option func(*Ticker)

// WithClient sets a custom client for the Ticker.
func WithClient(c *client.Client) Option {
	return func(t *Ticker) {
		t.client = c
		t.ownsClient = false
	}
}

// New creates a new Ticker for the given symbol.
func New(symbol string, opts ...Option) (*Ticker, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol cannot be empty")
	}

	t := &Ticker{
		symbol:     strings.ToUpper(symbol),
		ownsClient: true,
	}

	for _, opt := range opts {
		opt(t)
	}

	// Create default client if not provided
	if t.client == nil {
		var err error
		t.client, err = client.New()
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}
	}

	t.auth = client.NewAuthManager(t.client)

	return t, nil
}

// Symbol returns the ticker symbol.
func (t *Ticker) Symbol() string {
	return t.symbol
}

// Close releases resources used by the Ticker.
// If the client was created by the Ticker, it will be closed.
func (t *Ticker) Close() {
	if t.ownsClient && t.client != nil {
		t.client.Close()
	}
}

// getWithCrumb performs a GET request with crumb authentication.
func (t *Ticker) getWithCrumb(rawURL string, params url.Values) (*client.Response, error) {
	params, err := t.auth.AddCrumbToParams(params)
	if err != nil {
		return nil, fmt.Errorf("failed to get crumb: %w", err)
	}

	resp, err := t.client.Get(rawURL, params)
	if err != nil {
		return nil, err
	}

	// Check for rate limiting
	if resp.StatusCode == 429 {
		return nil, client.WrapRateLimitError()
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		return nil, client.HTTPStatusToError(resp.StatusCode, resp.Body)
	}

	return resp, nil
}

// chartURL returns the URL for the chart API.
func (t *Ticker) chartURL() string {
	return fmt.Sprintf("%s/%s", endpoints.ChartURL, t.symbol)
}

// quoteSummaryURL returns the URL for the quoteSummary API.
func (t *Ticker) quoteSummaryURL() string {
	return fmt.Sprintf("%s/%s", endpoints.QuoteSummaryURL, t.symbol)
}

// quoteURL returns the URL for the quote API.
func (t *Ticker) quoteURL() string {
	return endpoints.QuoteURL
}

// ClearCache clears all cached data.
func (t *Ticker) ClearCache() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.infoCache = nil
	t.quoteCache = nil
	t.historyMeta = nil
	t.optionsCache = nil
}

// GetHistoryMetadata returns the cached history metadata.
func (t *Ticker) GetHistoryMetadata() *models.ChartMeta {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.historyMeta
}

// setHistoryMetadata sets the history metadata cache.
func (t *Ticker) setHistoryMetadata(meta *models.ChartMeta) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.historyMeta = meta
}
