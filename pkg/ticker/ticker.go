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
	mu                    sync.RWMutex
	infoCache             *models.Info
	quoteCache            *models.Quote
	historyMeta           *models.ChartMeta
	tradingPeriodsLoading bool
	tradingPeriodsLoadSeq uint64
	tradingPeriodsLastErr error
	cacheGeneration       uint64
	tradingPeriodsCond    *sync.Cond
	tradingPeriodsFetcher func() (*models.ChartMeta, error)
	optionsCache          *optionsCache
	financialsCache       *financialsCache
	financialsChunked     bool
	analysisCache         *analysisCache
	valuationCache        map[string]*models.ValuationMeasures
	holdersCache          *holdersCache
	calendarCache         *models.Calendar
	newsCache             []models.NewsArticle
	newsCacheCount        int
	newsCacheTab          models.NewsTab

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
	t.tradingPeriodsCond = sync.NewCond(&t.mu)

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
	t.cacheGeneration++
	t.tradingPeriodsLastErr = nil
	t.optionsCache = nil
	t.financialsCache = nil
	t.analysisCache = nil
	t.valuationCache = nil
	t.holdersCache = nil
	t.calendarCache = nil
	t.newsCache = nil
	t.newsCacheCount = 0
	t.newsCacheTab = ""
}

// GetHistoryMetadata returns the cached history metadata.
func (t *Ticker) GetHistoryMetadata() *models.ChartMeta {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return cloneChartMeta(t.historyMeta)
}

// setHistoryMetadata sets the history metadata cache.
func (t *Ticker) setHistoryMetadata(meta *models.ChartMeta) {
	t.mu.Lock()
	defer t.mu.Unlock()
	clone := cloneChartMeta(meta)
	if clone != nil && !clone.HasTradingPeriods() && t.historyMeta != nil && t.historyMeta.HasTradingPeriods() {
		merged := clone.WithTradingPeriods(cloneTradingPeriods(t.historyMeta.TradingPeriods))
		clone = &merged
	}
	t.historyMeta = clone
}

func cloneChartMeta(meta *models.ChartMeta) *models.ChartMeta {
	if meta == nil {
		return nil
	}
	clone := *meta
	clone.ValidRanges = append([]string(nil), meta.ValidRanges...)
	clone.TradingPeriods = cloneTradingPeriods(meta.TradingPeriods)
	return &clone
}

func cloneTradingPeriods(periods []models.TradingPeriod) []models.TradingPeriod {
	if periods == nil {
		return nil
	}
	clone := make([]models.TradingPeriod, len(periods))
	for i, period := range periods {
		clone[i] = period
		clone[i].PreStart = cloneInt64(period.PreStart)
		clone[i].PreEnd = cloneInt64(period.PreEnd)
		clone[i].PostStart = cloneInt64(period.PostStart)
		clone[i].PostEnd = cloneInt64(period.PostEnd)
	}
	return clone
}

func cloneInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}
