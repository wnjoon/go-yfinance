// Package screener provides Yahoo Finance stock screener functionality.
//
// # Overview
//
// The screener package allows you to filter stocks based on various criteria
// using Yahoo Finance's predefined screeners or custom queries.
//
// # Predefined Screeners
//
// Yahoo Finance provides several predefined screeners:
//
//   - day_gainers: Stocks with highest percentage gain today
//   - day_losers: Stocks with highest percentage loss today
//   - most_actives: Stocks with highest trading volume
//   - aggressive_small_caps: Aggressive small cap stocks
//   - growth_technology_stocks: Growth technology stocks
//   - undervalued_growth_stocks: Undervalued growth stocks
//   - undervalued_large_caps: Undervalued large cap stocks
//   - most_shorted_stocks: Most shorted stocks
//   - small_cap_gainers: Small cap gainers
//
// # Basic Usage
//
//	s, err := screener.New()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer s.Close()
//
//	// Use predefined screener
//	result, err := s.Screen(models.ScreenerDayGainers, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, quote := range result.Quotes {
//	    fmt.Printf("%s: %.2f%%\n", quote.Symbol, quote.RegularMarketChangePercent)
//	}
//
// # Custom Queries
//
//	// Find US tech stocks with price > $50
//	query := models.NewEquityQuery(models.OpAND, []interface{}{
//	    models.NewEquityQuery(models.OpEQ, []interface{}{"region", "us"}),
//	    models.NewEquityQuery(models.OpEQ, []interface{}{"sector", "Technology"}),
//	    models.NewEquityQuery(models.OpGT, []interface{}{"intradayprice", 50}),
//	})
//	result, err := s.ScreenWithQuery(query, nil)
//
// # Thread Safety
//
// All Screener methods are safe for concurrent use from multiple goroutines.
package screener

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/wnjoon/go-yfinance/internal/endpoints"
	"github.com/wnjoon/go-yfinance/pkg/client"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

// Screener provides Yahoo Finance stock screener functionality.
type Screener struct {
	client     *client.Client
	ownsClient bool
}

// Option is a function that configures a Screener instance.
type Option func(*Screener)

// WithClient sets a custom HTTP client.
func WithClient(c *client.Client) Option {
	return func(s *Screener) {
		s.client = c
		s.ownsClient = false
	}
}

// New creates a new Screener instance.
//
// Example:
//
//	s, err := screener.New()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer s.Close()
func New(opts ...Option) (*Screener, error) {
	s := &Screener{
		ownsClient: true,
	}

	for _, opt := range opts {
		opt(s)
	}

	if s.client == nil {
		c, err := client.New()
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}
		s.client = c
	}

	return s, nil
}

// Close releases resources used by the Screener instance.
func (s *Screener) Close() {
	if s.ownsClient && s.client != nil {
		s.client.Close()
	}
}

// Screen uses a predefined screener to find matching stocks.
//
// Example:
//
//	result, err := s.Screen(models.ScreenerDayGainers, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, quote := range result.Quotes {
//	    fmt.Printf("%s: %.2f%%\n", quote.Symbol, quote.RegularMarketChangePercent)
//	}
func (s *Screener) Screen(screener models.PredefinedScreener, params *models.ScreenerParams) (*models.ScreenerResult, error) {
	if params == nil {
		defaultParams := models.DefaultScreenerParams()
		params = &defaultParams
	}

	if params.Count > 250 {
		return nil, fmt.Errorf("yahoo limits query count to 250, reduce count")
	}

	// If offset is specified, switch to POST endpoint with predefined query body
	// (Yahoo's predefined GET endpoint ignores offset)
	if params.Offset > 0 {
		predefined, ok := PredefinedScreenerQueries[string(screener)]
		if ok {
			// Use predefined's sort settings as defaults
			if params.SortField == "" || params.SortField == "ticker" {
				params.SortField = predefined.SortField
			}
			if !params.SortAsc {
				params.SortAsc = predefined.SortAsc
			}
			return s.ScreenWithQuery(predefined.Query, params)
		}
	}

	// Build URL for predefined screener
	screenerURL := fmt.Sprintf("%s/predefined/%s", endpoints.ScreenerURL, string(screener))

	// Build query parameters
	urlParams := url.Values{}
	urlParams.Set("count", strconv.Itoa(params.Count))
	urlParams.Set("offset", strconv.Itoa(params.Offset))
	if params.SortField != "" {
		urlParams.Set("sortField", params.SortField)
	}
	if params.SortAsc {
		urlParams.Set("sortAsc", "true")
	}

	resp, err := s.client.Get(screenerURL, urlParams)
	if err != nil {
		return nil, fmt.Errorf("screener request failed: %w", err)
	}

	return s.parseResponse(resp.Body, params.Offset)
}

// ScreenWithQuery uses a custom query to find matching stocks or funds.
// Accepts both *models.EquityQuery and *models.FundQuery.
// The quoteType is automatically determined from the query type:
// "EQUITY" for EquityQuery, "MUTUALFUND" for FundQuery.
//
// Example:
//
//	// Find US stocks with market cap > $10B
//	q1, _ := models.NewEquityQuery("eq", []interface{}{"region", "us"})
//	q2, _ := models.NewEquityQuery("gt", []interface{}{"intradaymarketcap", 10000000000})
//	query, _ := models.NewEquityQuery("and", []interface{}{q1, q2})
//	result, err := s.ScreenWithQuery(query, nil)
func (s *Screener) ScreenWithQuery(query models.ScreenerQueryBuilder, params *models.ScreenerParams) (*models.ScreenerResult, error) {
	if query == nil {
		return nil, fmt.Errorf("query is required")
	}

	if params == nil {
		defaultParams := models.DefaultScreenerParams()
		params = &defaultParams
	}

	if params.Count > 250 {
		return nil, fmt.Errorf("yahoo limits query count to 250, reduce count")
	}

	// Determine sort type
	sortType := "DESC"
	if params.SortAsc {
		sortType = "ASC"
	}

	// Build request body matching Python's format
	body := map[string]any{
		"offset":     params.Offset,
		"count":      params.Count,
		"sortField":  params.SortField,
		"sortType":   sortType,
		"userId":     params.UserID,
		"userIdType": params.UserIDType,
		"quoteType":  query.QuoteType(),
		"query":      query.ToDict(),
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	resp, err := s.client.PostJSON(endpoints.ScreenerURL, nil, bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("screener request failed: %w", err)
	}

	return s.parseResponse(resp.Body, params.Offset)
}

// DayGainers returns stocks with the highest percentage gain today.
//
// Example:
//
//	result, err := s.DayGainers(10)
//	for _, quote := range result.Quotes {
//	    fmt.Printf("%s: +%.2f%%\n", quote.Symbol, quote.RegularMarketChangePercent)
//	}
func (s *Screener) DayGainers(count int) (*models.ScreenerResult, error) {
	params := &models.ScreenerParams{Count: count}
	return s.Screen(models.ScreenerDayGainers, params)
}

// DayLosers returns stocks with the highest percentage loss today.
//
// Example:
//
//	result, err := s.DayLosers(10)
//	for _, quote := range result.Quotes {
//	    fmt.Printf("%s: %.2f%%\n", quote.Symbol, quote.RegularMarketChangePercent)
//	}
func (s *Screener) DayLosers(count int) (*models.ScreenerResult, error) {
	params := &models.ScreenerParams{Count: count}
	return s.Screen(models.ScreenerDayLosers, params)
}

// MostActives returns stocks with the highest trading volume today.
//
// Example:
//
//	result, err := s.MostActives(10)
//	for _, quote := range result.Quotes {
//	    fmt.Printf("%s: %d shares\n", quote.Symbol, quote.RegularMarketVolume)
//	}
func (s *Screener) MostActives(count int) (*models.ScreenerResult, error) {
	params := &models.ScreenerParams{Count: count}
	return s.Screen(models.ScreenerMostActives, params)
}

// parseResponse parses the raw API response into ScreenerResult.
func (s *Screener) parseResponse(body string, offset int) (*models.ScreenerResult, error) {
	var rawResp models.ScreenerResponse
	if err := json.Unmarshal([]byte(body), &rawResp); err != nil {
		return nil, fmt.Errorf("failed to parse screener response: %w", err)
	}

	// Check for API error
	if rawResp.Finance.Error != nil {
		return nil, fmt.Errorf("screener API error: %s - %s",
			rawResp.Finance.Error.Code, rawResp.Finance.Error.Description)
	}

	// Check if we have results
	if len(rawResp.Finance.Result) == 0 {
		return &models.ScreenerResult{
			Offset: offset,
		}, nil
	}

	result := rawResp.Finance.Result[0]
	screenerResult := &models.ScreenerResult{
		Total:  result.Total,
		Count:  result.Count,
		Offset: offset,
	}

	// Parse quotes
	for _, q := range result.Quotes {
		quote := models.ScreenerQuote{
			Symbol:                     getString(q, "symbol"),
			ShortName:                  getString(q, "shortName"),
			LongName:                   getString(q, "longName"),
			Exchange:                   getString(q, "exchange"),
			ExchangeDisp:               getString(q, "fullExchangeName"),
			QuoteType:                  getString(q, "quoteType"),
			Region:                     getString(q, "region"),
			Sector:                     getString(q, "sector"),
			Industry:                   getString(q, "industry"),
			Currency:                   getString(q, "currency"),
			MarketState:                getString(q, "marketState"),
			RegularMarketPrice:         getFloat(q, "regularMarketPrice"),
			RegularMarketChange:        getFloat(q, "regularMarketChange"),
			RegularMarketChangePercent: getFloat(q, "regularMarketChangePercent"),
			RegularMarketVolume:        getInt64(q, "regularMarketVolume"),
			RegularMarketDayHigh:       getFloat(q, "regularMarketDayHigh"),
			RegularMarketDayLow:        getFloat(q, "regularMarketDayLow"),
			RegularMarketOpen:          getFloat(q, "regularMarketOpen"),
			RegularMarketPreviousClose: getFloat(q, "regularMarketPreviousClose"),
			MarketCap:                  getInt64(q, "marketCap"),
			FiftyTwoWeekHigh:           getFloat(q, "fiftyTwoWeekHigh"),
			FiftyTwoWeekLow:            getFloat(q, "fiftyTwoWeekLow"),
			FiftyTwoWeekChange:         getFloat(q, "fiftyTwoWeekChangePercent"),
			FiftyDayAverage:            getFloat(q, "fiftyDayAverage"),
			TwoHundredDayAverage:       getFloat(q, "twoHundredDayAverage"),
			AverageVolume:              getInt64(q, "averageDailyVolume3Month"),
			TrailingPE:                 getFloat(q, "trailingPE"),
			ForwardPE:                  getFloat(q, "forwardPE"),
			PriceToBook:                getFloat(q, "priceToBook"),
			DividendYield:              getFloat(q, "dividendYield"),
			TrailingEPS:                getFloat(q, "epsTrailingTwelveMonths"),
			BookValue:                  getFloat(q, "bookValue"),
			// Fund-specific fields
			FundNetAssets:                 getFloat(q, "fundNetAssets"),
			CategoryName:                 getString(q, "categoryName"),
			PerformanceRatingOverall:      getInt(q, "performanceRatingOverall"),
			RiskRatingOverall:             getInt(q, "riskRatingOverall"),
			InitialInvestment:             getFloat(q, "initialInvestment"),
			AnnualReturnNavY1CategoryRank: getFloat(q, "annualReturnNavY1CategoryRank"),
		}
		screenerResult.Quotes = append(screenerResult.Quotes, quote)
	}

	return screenerResult, nil
}

// Helper functions
func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getFloat(m map[string]any, key string) float64 {
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

func getInt(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	}
	return 0
}

func getInt64(m map[string]any, key string) int64 {
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
