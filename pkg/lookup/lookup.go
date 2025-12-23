package lookup

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"sync"

	"github.com/wnjoon/go-yfinance/internal/endpoints"
	"github.com/wnjoon/go-yfinance/pkg/client"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

// Lookup provides Yahoo Finance ticker lookup functionality.
//
// Lookup allows searching for financial instruments by query string,
// with optional filtering by instrument type.
type Lookup struct {
	query string

	client     *client.Client
	ownsClient bool

	// Cache for lookup results
	mu    sync.RWMutex
	cache map[string]*models.LookupResult
}

// Option is a function that configures a Lookup instance.
type Option func(*Lookup)

// WithClient sets a custom HTTP client for the Lookup instance.
func WithClient(c *client.Client) Option {
	return func(l *Lookup) {
		l.client = c
		l.ownsClient = false
	}
}

// New creates a new Lookup instance for the given query.
//
// The query can be a ticker symbol, company name, or partial match.
//
// Example:
//
//	l, err := lookup.New("AAPL")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer l.Close()
//
//	results, err := l.All(10)
//	for _, doc := range results {
//	    fmt.Printf("%s: %s\n", doc.Symbol, doc.Name)
//	}
func New(query string, opts ...Option) (*Lookup, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	l := &Lookup{
		query:      query,
		ownsClient: true,
		cache:      make(map[string]*models.LookupResult),
	}

	for _, opt := range opts {
		opt(l)
	}

	if l.client == nil {
		c, err := client.New()
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}
		l.client = c
	}

	return l, nil
}

// Close releases resources used by the Lookup instance.
func (l *Lookup) Close() {
	if l.ownsClient && l.client != nil {
		l.client.Close()
	}
}

// Query returns the search query string.
func (l *Lookup) Query() string {
	return l.query
}

// fetch performs the actual API request to Yahoo Finance lookup endpoint.
func (l *Lookup) fetch(lookupType models.LookupType, count int) (*models.LookupResult, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s:%d", lookupType, count)
	l.mu.RLock()
	if cached, ok := l.cache[cacheKey]; ok {
		l.mu.RUnlock()
		return cached, nil
	}
	l.mu.RUnlock()

	// Build request parameters
	params := url.Values{}
	params.Set("query", l.query)
	params.Set("type", string(lookupType))
	params.Set("start", "0")
	params.Set("count", strconv.Itoa(count))
	params.Set("formatted", "false")
	params.Set("fetchPricingData", "true")
	params.Set("lang", "en-US")
	params.Set("region", "US")

	resp, err := l.client.Get(endpoints.LookupURL, params)
	if err != nil {
		return nil, fmt.Errorf("lookup request failed: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		return nil, client.HTTPStatusToError(resp.StatusCode, resp.Body)
	}

	// Parse response
	var rawResp models.LookupResponse
	if err := json.Unmarshal([]byte(resp.Body), &rawResp); err != nil {
		return nil, fmt.Errorf("failed to parse lookup response: %w", err)
	}

	// Check for API error
	if rawResp.Finance.Error != nil {
		return nil, fmt.Errorf("lookup API error: %s - %s",
			rawResp.Finance.Error.Code, rawResp.Finance.Error.Description)
	}

	// Parse results
	result := l.parseResponse(&rawResp)

	// Cache the result
	l.mu.Lock()
	l.cache[cacheKey] = result
	l.mu.Unlock()

	return result, nil
}

// parseResponse converts the raw API response to LookupResult.
func (l *Lookup) parseResponse(raw *models.LookupResponse) *models.LookupResult {
	result := &models.LookupResult{
		Documents: make([]models.LookupDocument, 0),
	}

	if len(raw.Finance.Result) == 0 {
		return result
	}

	docs := raw.Finance.Result[0].Documents
	result.Count = len(docs)

	for _, doc := range docs {
		document := models.LookupDocument{
			Symbol:                     getString(doc, "symbol"),
			Name:                       getString(doc, "name"),
			ShortName:                  getString(doc, "shortName"),
			Exchange:                   getString(doc, "exchange"),
			ExchangeDisplay:            getString(doc, "exchDisp"),
			QuoteType:                  getString(doc, "quoteType"),
			TypeDisplay:                getString(doc, "typeDisp"),
			Industry:                   getString(doc, "industry"),
			Sector:                     getString(doc, "sector"),
			Score:                      getFloat(doc, "score"),
			RegularMarketPrice:         getFloat(doc, "regularMarketPrice"),
			RegularMarketChange:        getFloat(doc, "regularMarketChange"),
			RegularMarketChangePercent: getFloat(doc, "regularMarketChangePercent"),
			RegularMarketPreviousClose: getFloat(doc, "regularMarketPreviousClose"),
			RegularMarketOpen:          getFloat(doc, "regularMarketOpen"),
			RegularMarketDayHigh:       getFloat(doc, "regularMarketDayHigh"),
			RegularMarketDayLow:        getFloat(doc, "regularMarketDayLow"),
			RegularMarketVolume:        getInt64(doc, "regularMarketVolume"),
			MarketCap:                  getInt64(doc, "marketCap"),
			FiftyTwoWeekHigh:           getFloat(doc, "fiftyTwoWeekHigh"),
			FiftyTwoWeekLow:            getFloat(doc, "fiftyTwoWeekLow"),
			Currency:                   getString(doc, "currency"),
			MarketState:                getString(doc, "marketState"),
		}

		// Use shortName if name is empty
		if document.Name == "" && document.ShortName != "" {
			document.Name = document.ShortName
		}

		result.Documents = append(result.Documents, document)
	}

	return result
}

// All returns all types of financial instruments matching the query.
//
// Parameters:
//   - count: Maximum number of results to return (default 25 if <= 0)
//
// Example:
//
//	results, err := l.All(10)
//	for _, doc := range results {
//	    fmt.Printf("%s: %s (%s)\n", doc.Symbol, doc.Name, doc.QuoteType)
//	}
func (l *Lookup) All(count int) ([]models.LookupDocument, error) {
	if count <= 0 {
		count = 25
	}
	result, err := l.fetch(models.LookupTypeAll, count)
	if err != nil {
		return nil, err
	}
	return result.Documents, nil
}

// Stock returns equity/stock instruments matching the query.
//
// Parameters:
//   - count: Maximum number of results to return (default 25 if <= 0)
//
// Example:
//
//	stocks, err := l.Stock(10)
//	for _, doc := range stocks {
//	    fmt.Printf("%s: %s\n", doc.Symbol, doc.Name)
//	}
func (l *Lookup) Stock(count int) ([]models.LookupDocument, error) {
	if count <= 0 {
		count = 25
	}
	result, err := l.fetch(models.LookupTypeEquity, count)
	if err != nil {
		return nil, err
	}
	return result.Documents, nil
}

// MutualFund returns mutual fund instruments matching the query.
//
// Parameters:
//   - count: Maximum number of results to return (default 25 if <= 0)
//
// Example:
//
//	funds, err := l.MutualFund(10)
//	for _, doc := range funds {
//	    fmt.Printf("%s: %s\n", doc.Symbol, doc.Name)
//	}
func (l *Lookup) MutualFund(count int) ([]models.LookupDocument, error) {
	if count <= 0 {
		count = 25
	}
	result, err := l.fetch(models.LookupTypeMutualFund, count)
	if err != nil {
		return nil, err
	}
	return result.Documents, nil
}

// ETF returns ETF instruments matching the query.
//
// Parameters:
//   - count: Maximum number of results to return (default 25 if <= 0)
//
// Example:
//
//	etfs, err := l.ETF(10)
//	for _, doc := range etfs {
//	    fmt.Printf("%s: %s\n", doc.Symbol, doc.Name)
//	}
func (l *Lookup) ETF(count int) ([]models.LookupDocument, error) {
	if count <= 0 {
		count = 25
	}
	result, err := l.fetch(models.LookupTypeETF, count)
	if err != nil {
		return nil, err
	}
	return result.Documents, nil
}

// Index returns index instruments matching the query.
//
// Parameters:
//   - count: Maximum number of results to return (default 25 if <= 0)
//
// Example:
//
//	indices, err := l.Index(10)
//	for _, doc := range indices {
//	    fmt.Printf("%s: %s\n", doc.Symbol, doc.Name)
//	}
func (l *Lookup) Index(count int) ([]models.LookupDocument, error) {
	if count <= 0 {
		count = 25
	}
	result, err := l.fetch(models.LookupTypeIndex, count)
	if err != nil {
		return nil, err
	}
	return result.Documents, nil
}

// Future returns futures instruments matching the query.
//
// Parameters:
//   - count: Maximum number of results to return (default 25 if <= 0)
//
// Example:
//
//	futures, err := l.Future(10)
//	for _, doc := range futures {
//	    fmt.Printf("%s: %s\n", doc.Symbol, doc.Name)
//	}
func (l *Lookup) Future(count int) ([]models.LookupDocument, error) {
	if count <= 0 {
		count = 25
	}
	result, err := l.fetch(models.LookupTypeFuture, count)
	if err != nil {
		return nil, err
	}
	return result.Documents, nil
}

// Currency returns currency instruments matching the query.
//
// Parameters:
//   - count: Maximum number of results to return (default 25 if <= 0)
//
// Example:
//
//	currencies, err := l.Currency(10)
//	for _, doc := range currencies {
//	    fmt.Printf("%s: %s\n", doc.Symbol, doc.Name)
//	}
func (l *Lookup) Currency(count int) ([]models.LookupDocument, error) {
	if count <= 0 {
		count = 25
	}
	result, err := l.fetch(models.LookupTypeCurrency, count)
	if err != nil {
		return nil, err
	}
	return result.Documents, nil
}

// Cryptocurrency returns cryptocurrency instruments matching the query.
//
// Parameters:
//   - count: Maximum number of results to return (default 25 if <= 0)
//
// Example:
//
//	cryptos, err := l.Cryptocurrency(10)
//	for _, doc := range cryptos {
//	    fmt.Printf("%s: %s\n", doc.Symbol, doc.Name)
//	}
func (l *Lookup) Cryptocurrency(count int) ([]models.LookupDocument, error) {
	if count <= 0 {
		count = 25
	}
	result, err := l.fetch(models.LookupTypeCryptocurrency, count)
	if err != nil {
		return nil, err
	}
	return result.Documents, nil
}

// ClearCache clears the cached lookup results.
func (l *Lookup) ClearCache() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cache = make(map[string]*models.LookupResult)
}

// Helper functions for parsing map values

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getFloat(m map[string]interface{}, key string) float64 {
	switch v := m[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return 0
}

func getInt64(m map[string]interface{}, key string) int64 {
	switch v := m[key].(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	}
	return 0
}
