// Package search provides Yahoo Finance search functionality.
//
// # Overview
//
// The search package allows you to search for stock symbols, company names,
// and other financial assets on Yahoo Finance. It also retrieves related
// news articles, lists, and research reports.
//
// # Basic Usage
//
//	s, err := search.New()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer s.Close()
//
//	result, err := s.Search("AAPL")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, quote := range result.Quotes {
//	    fmt.Printf("%s: %s (%s)\n", quote.Symbol, quote.ShortName, quote.Exchange)
//	}
//
// # Search with Parameters
//
//	params := models.SearchParams{
//	    Query:            "Apple",
//	    MaxResults:       10,
//	    NewsCount:        5,
//	    EnableFuzzyQuery: true,
//	}
//	result, err := s.SearchWithParams(params)
//
// # Thread Safety
//
// All Search methods are safe for concurrent use from multiple goroutines.
package search

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

// Search provides Yahoo Finance search functionality.
type Search struct {
	client     *client.Client
	mu         sync.RWMutex
	ownsClient bool
}

// Option is a function that configures a Search instance.
type Option func(*Search)

// WithClient sets a custom HTTP client.
func WithClient(c *client.Client) Option {
	return func(s *Search) {
		s.client = c
		s.ownsClient = false
	}
}

// New creates a new Search instance.
//
// Example:
//
//	s, err := search.New()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer s.Close()
func New(opts ...Option) (*Search, error) {
	s := &Search{
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

// Close releases resources used by the Search instance.
func (s *Search) Close() {
	if s.ownsClient && s.client != nil {
		s.client.Close()
	}
}

// Search searches for symbols and returns matching quotes.
//
// This is a convenience method that uses default parameters.
//
// Example:
//
//	result, err := s.Search("AAPL")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, quote := range result.Quotes {
//	    fmt.Printf("%s: %s\n", quote.Symbol, quote.ShortName)
//	}
func (s *Search) Search(query string) (*models.SearchResult, error) {
	params := models.DefaultSearchParams()
	params.Query = query
	return s.SearchWithParams(params)
}

// SearchWithParams searches with custom parameters.
//
// Example:
//
//	params := models.SearchParams{
//	    Query:            "Apple",
//	    MaxResults:       10,
//	    NewsCount:        5,
//	    EnableFuzzyQuery: true,
//	}
//	result, err := s.SearchWithParams(params)
func (s *Search) SearchWithParams(params models.SearchParams) (*models.SearchResult, error) {
	if params.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	// Build URL parameters
	urlParams := url.Values{}
	urlParams.Set("q", params.Query)
	urlParams.Set("quotesCount", strconv.Itoa(params.MaxResults))
	urlParams.Set("newsCount", strconv.Itoa(params.NewsCount))
	urlParams.Set("listsCount", strconv.Itoa(params.ListsCount))
	urlParams.Set("quotesQueryId", "tss_match_phrase_query")
	urlParams.Set("newsQueryId", "news_cie_vespa")

	if params.EnableFuzzyQuery {
		urlParams.Set("enableFuzzyQuery", "true")
	}
	if params.IncludeResearch {
		urlParams.Set("enableResearchReports", "true")
	}
	if params.IncludeNav {
		urlParams.Set("enableNavLinks", "true")
	}

	resp, err := s.client.Get(endpoints.SearchURL, urlParams)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}

	var rawResp models.SearchResponse
	if err := json.Unmarshal([]byte(resp.Body), &rawResp); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	return s.parseSearchResult(&rawResp), nil
}

// Quotes returns only the quote results for a search query.
//
// Example:
//
//	quotes, err := s.Quotes("Apple", 10)
//	for _, q := range quotes {
//	    fmt.Printf("%s: %s\n", q.Symbol, q.ShortName)
//	}
func (s *Search) Quotes(query string, maxResults int) ([]models.SearchQuote, error) {
	params := models.SearchParams{
		Query:      query,
		MaxResults: maxResults,
		NewsCount:  0,
		ListsCount: 0,
	}
	result, err := s.SearchWithParams(params)
	if err != nil {
		return nil, err
	}
	return result.Quotes, nil
}

// News returns only the news results for a search query.
//
// Example:
//
//	news, err := s.News("Apple", 10)
//	for _, n := range news {
//	    fmt.Printf("%s: %s\n", n.Publisher, n.Title)
//	}
func (s *Search) News(query string, newsCount int) ([]models.SearchNews, error) {
	params := models.SearchParams{
		Query:      query,
		MaxResults: 0,
		NewsCount:  newsCount,
		ListsCount: 0,
	}
	result, err := s.SearchWithParams(params)
	if err != nil {
		return nil, err
	}
	return result.News, nil
}

// parseSearchResult converts raw API response to SearchResult.
func (s *Search) parseSearchResult(raw *models.SearchResponse) *models.SearchResult {
	result := &models.SearchResult{
		TotalCount: raw.Count,
	}

	// Parse quotes
	for _, q := range raw.Quotes {
		quote := models.SearchQuote{
			Symbol:         getString(q, "symbol"),
			ShortName:      getString(q, "shortname"),
			LongName:       getString(q, "longname"),
			Exchange:       getString(q, "exchange"),
			ExchangeDisp:   getString(q, "exchDisp"),
			QuoteType:      getString(q, "quoteType"),
			TypeDisp:       getString(q, "typeDisp"),
			Score:          getFloat(q, "score"),
			IsYahooFinance: getBool(q, "isYahooFinance"),
			Industry:       getString(q, "industry"),
			Sector:         getString(q, "sector"),
		}
		result.Quotes = append(result.Quotes, quote)
	}

	// Parse news
	for _, n := range raw.News {
		news := models.SearchNews{
			UUID:           getString(n, "uuid"),
			Title:          getString(n, "title"),
			Publisher:      getString(n, "publisher"),
			Link:           getString(n, "link"),
			PublishTime:    getInt64(n, "providerPublishTime"),
			Type:           getString(n, "type"),
			RelatedTickers: getStringSlice(n, "relatedTickers"),
		}

		// Parse thumbnail
		if thumbRaw, ok := n["thumbnail"].(map[string]interface{}); ok {
			if resolutions, ok := thumbRaw["resolutions"].([]interface{}); ok {
				thumb := &models.SearchThumbnail{}
				for _, r := range resolutions {
					if res, ok := r.(map[string]interface{}); ok {
						thumb.Resolutions = append(thumb.Resolutions, models.ThumbnailResolution{
							URL:    getString(res, "url"),
							Width:  int(getFloat(res, "width")),
							Height: int(getFloat(res, "height")),
							Tag:    getString(res, "tag"),
						})
					}
				}
				news.Thumbnail = thumb
			}
		}

		result.News = append(result.News, news)
	}

	// Parse lists
	for _, l := range raw.Lists {
		list := models.SearchList{
			ID:          getString(l, "id"),
			Name:        getString(l, "name"),
			Description: getString(l, "description"),
			SymbolCount: int(getFloat(l, "symbolCount")),
			URL:         getString(l, "url"),
		}
		result.Lists = append(result.Lists, list)
	}

	// Parse research
	for _, r := range raw.Research {
		research := models.SearchResearch{
			ReportID:    getString(r, "reportId"),
			Title:       getString(r, "title"),
			Provider:    getString(r, "provider"),
			Ticker:      getString(r, "ticker"),
			PublishDate: getString(r, "publishDate"),
		}
		result.Research = append(result.Research, research)
	}

	// Parse nav
	for _, n := range raw.Nav {
		nav := models.SearchNav{
			Name: getString(n, "name"),
			URL:  getString(n, "url"),
		}
		result.Nav = append(result.Nav, nav)
	}

	return result
}

// Helper functions
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

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func getStringSlice(m map[string]interface{}, key string) []string {
	if arr, ok := m[key].([]interface{}); ok {
		result := make([]string, 0, len(arr))
		for _, v := range arr {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}
