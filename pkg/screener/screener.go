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

// maxScreenerResults is Yahoo's per-request limit for count and size.
const maxScreenerResults = 250

// httpDoer is the request transport used by Screener. It exists so tests can
// capture outgoing requests without network access.
type httpDoer interface {
	Get(rawURL string, params url.Values) (*client.Response, error)
	PostJSON(rawURL string, params url.Values, body []byte) (*client.Response, error)
}

// crumbDoer sends screener requests with Yahoo cookie + crumb authentication.
// Yahoo rejects crumb-less screener POSTs with "Unauthorized - Invalid Crumb",
// and Python yfinance sends both screener requests through its crumb-aware
// YfData session.
type crumbDoer struct {
	auth *client.AuthManager
}

func (d crumbDoer) Get(rawURL string, params url.Values) (*client.Response, error) {
	return d.auth.GetWithCrumb(rawURL, params)
}

func (d crumbDoer) PostJSON(rawURL string, params url.Values, body []byte) (*client.Response, error) {
	return d.auth.PostJSONWithCrumb(rawURL, params, body)
}

// Screener provides Yahoo Finance stock screener functionality.
type Screener struct {
	client     *client.Client
	auth       *client.AuthManager
	ownsClient bool

	// http performs requests; nil means crumb-authenticated requests via auth.
	http httpDoer
}

// doer returns the request transport for this Screener.
func (s *Screener) doer() httpDoer {
	if s.http != nil {
		return s.http
	}
	return crumbDoer{auth: s.auth}
}

// baseParams returns the query parameters Python yfinance sends with every
// screener request.
func baseParams() url.Values {
	return url.Values{
		"corsDomain": {"finance.yahoo.com"},
		"formatted":  {"false"},
		"lang":       {"en-US"},
		"region":     {"US"},
	}
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
	s.auth = client.NewAuthManager(s.client)

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
// Without an offset, Screen calls Yahoo's predefined GET endpoint
// (/v1/finance/screener/predefined/saved?scrIds=<name>, as Python yfinance
// does) with the count, offset, and sort parameters. That endpoint ignores offset, so when
// params.Offset > 0 and the screener has an entry in PredefinedScreenerQueries,
// Screen switches to the custom-query POST endpoint using the predefined query
// (matching Python yfinance's screen()). In that case the predefined sort field
// and direction are used unless params.SortField is set to something other than
// "" or "ticker" (or params.SortAsc is true), and the number of results is sent
// as Yahoo's "size" field: params.Size if set, otherwise params.Count.
//
// Python switches endpoints whenever offset is given, including offset=0. Go
// cannot distinguish an explicit zero from an unset int, and an offset of 0
// returns the first page on either endpoint, so Go switches only for Offset > 0.
//
// If params.Size is set on the GET path it takes precedence over params.Count,
// mirroring Python's deprecated handling of size for predefined screens.
// Count and Size must not exceed 250 (Yahoo's limit). The caller's params are
// never modified.
//
// Requests are sent with Yahoo cookie + crumb authentication. HTTP failures
// return the pkg/client error types (for example client.IsRateLimitError for
// HTTP 429 and client.IsNotFoundError for an unknown screener name).
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
	p := models.DefaultScreenerParams()
	if params != nil {
		// Copy so the caller's params are never mutated.
		p = *params
	}

	if err := validateLimits(&p); err != nil {
		return nil, err
	}

	// If offset is specified, switch to POST endpoint with predefined query body
	// (Yahoo's predefined GET endpoint ignores offset)
	if p.Offset > 0 {
		if predefined, ok := PredefinedScreenerQueries[string(screener)]; ok {
			return s.postQuery(buildPredefinedOffsetBody(predefined, &p), p.Offset)
		}
	}

	resp, err := s.doer().Get(endpoints.ScreenerPredefinedURL, buildPredefinedParams(screener, &p))
	if err != nil {
		return nil, fmt.Errorf("screener request failed: %w", err)
	}

	return s.parseHTTPResponse(resp, p.Offset)
}

// ScreenWithQuery uses a custom query to find matching stocks or funds.
// Accepts *models.EquityQuery, *models.FundQuery, and *models.ETFQuery.
// The quoteType is automatically determined from the query type:
// "EQUITY" for EquityQuery, "MUTUALFUND" for FundQuery, "ETF" for ETFQuery.
//
// The request body always includes params.Count as "count"; params.Size is
// sent as "size" only when it is greater than zero, matching Python yfinance.
// Count and Size must not exceed 250 (Yahoo's limit).
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

	p := models.DefaultScreenerParams()
	if params != nil {
		p = *params
	}

	if err := validateLimits(&p); err != nil {
		return nil, err
	}

	return s.postQuery(buildCustomQueryBody(query, &p), p.Offset)
}

// validateLimits enforces Yahoo's 250-result cap on count and size.
func validateLimits(p *models.ScreenerParams) error {
	if p.Count > maxScreenerResults {
		return fmt.Errorf("yahoo limits query count to %d, reduce count", maxScreenerResults)
	}
	if p.Size > maxScreenerResults {
		return fmt.Errorf("yahoo limits query size to %d, reduce size", maxScreenerResults)
	}
	return nil
}

// sortType converts an ascending flag to Yahoo's sortType value.
func sortType(asc bool) string {
	if asc {
		return "ASC"
	}
	return "DESC"
}

// buildPredefinedParams builds the query string for the predefined GET endpoint.
func buildPredefinedParams(screener models.PredefinedScreener, p *models.ScreenerParams) url.Values {
	count := p.Count
	if p.Size > 0 {
		// Python treats size as a deprecated alias for count here.
		count = p.Size
	}

	urlParams := baseParams()
	urlParams.Set("scrIds", string(screener))
	urlParams.Set("count", strconv.Itoa(count))
	urlParams.Set("offset", strconv.Itoa(p.Offset))
	if p.SortField != "" {
		urlParams.Set("sortField", p.SortField)
	}
	if p.SortAsc {
		urlParams.Set("sortAsc", "true")
	}
	return urlParams
}

// buildCustomQueryBody builds the POST body for a custom query.
func buildCustomQueryBody(query models.ScreenerQueryBuilder, p *models.ScreenerParams) map[string]any {
	body := map[string]any{
		"offset":     p.Offset,
		"count":      p.Count,
		"sortField":  p.SortField,
		"sortType":   sortType(p.SortAsc),
		"userId":     p.UserID,
		"userIdType": p.UserIDType,
		"quoteType":  query.QuoteType(),
		"query":      query.ToDict(),
	}
	if p.Size > 0 {
		body["size"] = p.Size
	}
	return body
}

// buildPredefinedOffsetBody builds the POST body used when a predefined
// screener is switched to the custom-query endpoint to honour an offset.
//
// The POST endpoint takes "size", not "count" (Python yfinance #3006), so the
// requested count is sent as size. If Size is set explicitly it is used as size
// and a non-zero Count is passed through as count, as Python does.
func buildPredefinedOffsetBody(predefined PredefinedQuery, p *models.ScreenerParams) map[string]any {
	sortField := p.SortField
	if sortField == "" || sortField == "ticker" {
		sortField = predefined.SortField
	}
	sortAsc := p.SortAsc || predefined.SortAsc

	body := map[string]any{
		"offset":     p.Offset,
		"sortField":  sortField,
		"sortType":   sortType(sortAsc),
		"userId":     p.UserID,
		"userIdType": p.UserIDType,
		"quoteType":  predefined.Query.QuoteType(),
		"query":      predefined.Query.ToDict(),
	}

	switch {
	case p.Size > 0:
		body["size"] = p.Size
		if p.Count > 0 {
			body["count"] = p.Count
		}
	case p.Count > 0:
		body["size"] = p.Count
	}
	return body
}

// postQuery sends a screener POST body and parses the response.
func (s *Screener) postQuery(body map[string]any, offset int) (*models.ScreenerResult, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	resp, err := s.doer().PostJSON(endpoints.ScreenerURL, baseParams(), bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("screener request failed: %w", err)
	}

	return s.parseHTTPResponse(resp, offset)
}

// parseHTTPResponse checks the HTTP status and parses the response body.
//
// Yahoo returns JSON error envelopes ({"finance":{"error":...}}) with 4xx
// statuses, and HTML pages for some failures. A JSON error keeps its Yahoo
// code and description in the message; for HTTP >= 400 the error also wraps
// the matching pkg/client error type so callers can use errors.Is.
func (s *Screener) parseHTTPResponse(resp *client.Response, offset int) (*models.ScreenerResult, error) {
	if resp == nil {
		return nil, fmt.Errorf("screener request failed: %w", client.ErrInvalidResponse)
	}
	if resp.StatusCode < 400 {
		return s.parseResponse(resp.Body, offset)
	}

	statusErr := client.HTTPStatusToError(resp.StatusCode, truncateBody(resp.Body))
	var rawResp models.ScreenerResponse
	if err := json.Unmarshal([]byte(resp.Body), &rawResp); err == nil && rawResp.Finance.Error != nil {
		return nil, fmt.Errorf("screener API error: %s - %s: %w",
			rawResp.Finance.Error.Code, rawResp.Finance.Error.Description, statusErr)
	}
	return nil, fmt.Errorf("screener request failed: %w", statusErr)
}

// truncateBody shortens a response body for inclusion in an error message.
func truncateBody(body string) string {
	const limit = 200
	if len(body) > limit {
		return body[:limit] + "..."
	}
	return body
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
			CategoryName:                  getString(q, "categoryName"),
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
