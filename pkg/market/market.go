package market

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/wnjoon/go-yfinance/internal/endpoints"
	"github.com/wnjoon/go-yfinance/pkg/client"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

// Market provides access to market status and summary information.
//
// Market allows retrieving market opening/closing times, timezone information,
// and summary data for major market indices.
type Market struct {
	market string

	client     *client.Client
	ownsClient bool

	// Cached data
	mu            sync.RWMutex
	statusCache   *models.MarketStatus
	summaryCache  models.MarketSummary
	lastFetchTime time.Time
}

// Option is a function that configures a Market instance.
type Option func(*Market)

// WithClient sets a custom HTTP client for the Market instance.
func WithClient(c *client.Client) Option {
	return func(m *Market) {
		m.client = c
		m.ownsClient = false
	}
}

// New creates a new Market instance for the given market identifier.
//
// Common market identifiers:
//   - "us_market" - United States
//   - "gb_market" - United Kingdom
//   - "de_market" - Germany
//   - "jp_market" - Japan
//   - "hk_market" - Hong Kong
//   - "cn_market" - China
//
// Example:
//
//	m, err := market.New("us_market")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer m.Close()
//
//	status, err := m.Status()
//	fmt.Printf("Market opens at: %s\n", status.Open.Format("15:04"))
func New(market string, opts ...Option) (*Market, error) {
	if market == "" {
		return nil, fmt.Errorf("market identifier cannot be empty")
	}

	m := &Market{
		market:     market,
		ownsClient: true,
	}

	for _, opt := range opts {
		opt(m)
	}

	if m.client == nil {
		c, err := client.New()
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}
		m.client = c
	}

	return m, nil
}

// NewWithPredefined creates a new Market instance using a predefined market constant.
//
// Example:
//
//	m, err := market.NewWithPredefined(models.MarketUS)
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewWithPredefined(market models.PredefinedMarket, opts ...Option) (*Market, error) {
	return New(string(market), opts...)
}

// Close releases resources used by the Market instance.
func (m *Market) Close() {
	if m.ownsClient && m.client != nil {
		m.client.Close()
	}
}

// Market returns the market identifier string.
func (m *Market) Market() string {
	return m.market
}

// fetchData fetches both status and summary data from Yahoo Finance.
func (m *Market) fetchData() error {
	m.mu.RLock()
	if m.statusCache != nil && m.summaryCache != nil {
		m.mu.RUnlock()
		return nil
	}
	m.mu.RUnlock()

	// Fetch summary
	// Extract region from market identifier (e.g., "us_market" -> "US")
	region := extractRegion(m.market)

	summaryParams := url.Values{}
	summaryParams.Set("fields", "shortName,regularMarketPrice,regularMarketChange,regularMarketChangePercent")
	summaryParams.Set("formatted", "false")
	summaryParams.Set("lang", "en-US")
	summaryParams.Set("region", region)

	summaryResp, err := m.client.Get(endpoints.MarketSummaryURL, summaryParams)
	if err != nil {
		return fmt.Errorf("failed to fetch market summary: %w", err)
	}

	if summaryResp.StatusCode >= 400 {
		return client.HTTPStatusToError(summaryResp.StatusCode, summaryResp.Body)
	}

	// Fetch status/time
	statusParams := url.Values{}
	statusParams.Set("formatted", "true")
	statusParams.Set("key", "finance")
	statusParams.Set("lang", "en-US")
	statusParams.Set("region", region)

	statusResp, err := m.client.Get(endpoints.MarketTimeURL, statusParams)
	if err != nil {
		return fmt.Errorf("failed to fetch market time: %w", err)
	}

	if statusResp.StatusCode >= 400 {
		return client.HTTPStatusToError(statusResp.StatusCode, statusResp.Body)
	}

	// Parse summary
	var summaryRaw models.MarketSummaryResponse
	if err := json.Unmarshal([]byte(summaryResp.Body), &summaryRaw); err != nil {
		return fmt.Errorf("failed to parse market summary: %w", err)
	}

	if summaryRaw.MarketSummaryResponse.Error != nil {
		return fmt.Errorf("market summary API error: %s - %s",
			summaryRaw.MarketSummaryResponse.Error.Code,
			summaryRaw.MarketSummaryResponse.Error.Description)
	}

	summary := m.parseSummary(summaryRaw.MarketSummaryResponse.Result)

	// Parse status
	var statusRaw models.MarketTimeResponse
	if err := json.Unmarshal([]byte(statusResp.Body), &statusRaw); err != nil {
		return fmt.Errorf("failed to parse market time: %w", err)
	}

	if statusRaw.Finance.Error != nil {
		return fmt.Errorf("market time API error: %s - %s",
			statusRaw.Finance.Error.Code,
			statusRaw.Finance.Error.Description)
	}

	status, err := m.parseStatus(&statusRaw)
	if err != nil {
		return fmt.Errorf("failed to parse market status: %w", err)
	}

	// Update cache
	m.mu.Lock()
	m.statusCache = status
	m.summaryCache = summary
	m.lastFetchTime = time.Now()
	m.mu.Unlock()

	return nil
}

// parseSummary converts raw API response to MarketSummary.
func (m *Market) parseSummary(results []map[string]interface{}) models.MarketSummary {
	summary := make(models.MarketSummary)

	for _, result := range results {
		exchange := getString(result, "exchange")
		if exchange == "" {
			continue
		}

		item := models.MarketSummaryItem{
			Exchange:                   exchange,
			Symbol:                     getString(result, "symbol"),
			ShortName:                  getString(result, "shortName"),
			FullExchangeName:           getString(result, "fullExchangeName"),
			MarketState:                getString(result, "marketState"),
			RegularMarketPrice:         getFloat(result, "regularMarketPrice"),
			RegularMarketChange:        getFloat(result, "regularMarketChange"),
			RegularMarketChangePercent: getFloat(result, "regularMarketChangePercent"),
			RegularMarketPreviousClose: getFloat(result, "regularMarketPreviousClose"),
			RegularMarketTime:          getInt64(result, "regularMarketTime"),
			QuoteType:                  getString(result, "quoteType"),
			SourceInterval:             int(getFloat(result, "sourceInterval")),
			ExchangeDataDelayedBy:      int(getFloat(result, "exchangeDataDelayedBy")),
		}

		summary[exchange] = item
	}

	return summary
}

// parseStatus converts raw API response to MarketStatus.
func (m *Market) parseStatus(raw *models.MarketTimeResponse) (*models.MarketStatus, error) {
	if len(raw.Finance.MarketTimes) == 0 {
		return nil, fmt.Errorf("no market times in response")
	}

	marketTimes := raw.Finance.MarketTimes[0]
	if len(marketTimes.MarketTime) == 0 {
		return nil, fmt.Errorf("no market time data in response")
	}

	mt := marketTimes.MarketTime[0]

	status := &models.MarketStatus{
		ID:       mt.ID,
		OpenStr:  mt.Open,
		CloseStr: mt.Close,
	}

	// Parse timezone
	if len(mt.Timezone) > 0 {
		tz := mt.Timezone[0]
		status.Timezone = &models.MarketTimezone{
			GMTOffset: getInt64(tz, "gmtoffset"),
			Short:     getString(tz, "short"),
			Long:      getString(tz, "long"),
		}
	}

	// Parse open time
	if mt.Open != "" {
		openTime, err := time.Parse(time.RFC3339, mt.Open)
		if err == nil {
			status.Open = &openTime
		}
	}

	// Parse close time
	if mt.Close != "" {
		closeTime, err := time.Parse(time.RFC3339, mt.Close)
		if err == nil {
			status.Close = &closeTime
		}
	}

	return status, nil
}

// Status returns the current market status including opening/closing times.
//
// Example:
//
//	status, err := m.Status()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if status.Open != nil {
//	    fmt.Printf("Opens at: %s\n", status.Open.Format("15:04"))
//	}
//	if status.Timezone != nil {
//	    fmt.Printf("Timezone: %s\n", status.Timezone.Short)
//	}
func (m *Market) Status() (*models.MarketStatus, error) {
	if err := m.fetchData(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.statusCache, nil
}

// Summary returns the market summary with major indices data.
//
// The result is a map where keys are exchange codes (e.g., "SNP", "DJI")
// and values contain price and change information.
//
// Example:
//
//	summary, err := m.Summary()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for exchange, item := range summary {
//	    fmt.Printf("%s (%s): %.2f (%.2f%%)\n",
//	        item.ShortName, exchange,
//	        item.RegularMarketPrice,
//	        item.RegularMarketChangePercent)
//	}
func (m *Market) Summary() (models.MarketSummary, error) {
	if err := m.fetchData(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.summaryCache, nil
}

// ClearCache clears the cached market data.
// The next call to Status() or Summary() will fetch fresh data.
func (m *Market) ClearCache() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statusCache = nil
	m.summaryCache = nil
}

// IsOpen returns true if the market is currently open.
// This is determined by checking if the current time is between open and close times.
func (m *Market) IsOpen() (bool, error) {
	status, err := m.Status()
	if err != nil {
		return false, err
	}

	if status.Open == nil || status.Close == nil {
		return false, nil
	}

	now := time.Now()

	// Simple check - this may need timezone adjustment
	return now.After(*status.Open) && now.Before(*status.Close), nil
}

// extractRegion extracts the region code from market identifier.
// e.g., "us_market" -> "US", "gb_market" -> "GB"
func extractRegion(market string) string {
	regionMap := map[string]string{
		"us_market": "US",
		"gb_market": "GB",
		"de_market": "DE",
		"fr_market": "FR",
		"jp_market": "JP",
		"hk_market": "HK",
		"cn_market": "CN",
		"ca_market": "CA",
		"au_market": "AU",
		"in_market": "IN",
		"kr_market": "KR",
		"br_market": "BR",
	}

	if region, ok := regionMap[market]; ok {
		return region
	}

	// Fallback: try to extract from pattern "{code}_market"
	if len(market) >= 3 && market[2] == '_' {
		return strings.ToUpper(market[:2])
	}

	return "US" // Default to US
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
