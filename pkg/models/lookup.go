package models

// LookupType represents the type of financial instrument for lookup queries.
type LookupType string

const (
	// LookupTypeAll returns all types of financial instruments.
	LookupTypeAll LookupType = "all"

	// LookupTypeEquity returns stock/equity instruments.
	LookupTypeEquity LookupType = "equity"

	// LookupTypeMutualFund returns mutual fund instruments.
	LookupTypeMutualFund LookupType = "mutualfund"

	// LookupTypeETF returns ETF instruments.
	LookupTypeETF LookupType = "etf"

	// LookupTypeIndex returns index instruments.
	LookupTypeIndex LookupType = "index"

	// LookupTypeFuture returns future instruments.
	LookupTypeFuture LookupType = "future"

	// LookupTypeCurrency returns currency instruments.
	LookupTypeCurrency LookupType = "currency"

	// LookupTypeCryptocurrency returns cryptocurrency instruments.
	LookupTypeCryptocurrency LookupType = "cryptocurrency"
)

// LookupResult represents the result of a lookup query.
//
// This contains a list of matching financial instruments based on the search query.
//
// Example:
//
//	l, err := lookup.New("AAPL")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	results, err := l.All(10)
//	for _, doc := range results {
//	    fmt.Printf("%s: %s (%s)\n", doc.Symbol, doc.Name, doc.Exchange)
//	}
type LookupResult struct {
	// Documents contains the list of matching financial instruments.
	Documents []LookupDocument `json:"documents"`

	// Count is the number of results returned.
	Count int `json:"count"`
}

// LookupDocument represents a single financial instrument from lookup results.
//
// This structure contains detailed information about a matching ticker symbol,
// including pricing data if requested.
type LookupDocument struct {
	// Symbol is the ticker symbol.
	Symbol string `json:"symbol"`

	// Name is the company/instrument name.
	Name string `json:"name"`

	// ShortName is the short name (alternative to Name).
	ShortName string `json:"shortName,omitempty"`

	// Exchange is the exchange where the instrument is traded.
	Exchange string `json:"exchange"`

	// ExchangeDisplay is the display name of the exchange.
	ExchangeDisplay string `json:"exchDisp,omitempty"`

	// QuoteType is the type of instrument (EQUITY, ETF, MUTUALFUND, etc.).
	QuoteType string `json:"quoteType"`

	// TypeDisplay is the display name of the instrument type.
	TypeDisplay string `json:"typeDisp,omitempty"`

	// Industry is the industry classification.
	Industry string `json:"industry,omitempty"`

	// Sector is the sector classification.
	Sector string `json:"sector,omitempty"`

	// Score is the relevance score of this result.
	Score float64 `json:"score,omitempty"`

	// Pricing data (when fetchPricingData=true)

	// RegularMarketPrice is the current market price.
	RegularMarketPrice float64 `json:"regularMarketPrice,omitempty"`

	// RegularMarketChange is the price change.
	RegularMarketChange float64 `json:"regularMarketChange,omitempty"`

	// RegularMarketChangePercent is the percentage price change.
	RegularMarketChangePercent float64 `json:"regularMarketChangePercent,omitempty"`

	// RegularMarketPreviousClose is the previous closing price.
	RegularMarketPreviousClose float64 `json:"regularMarketPreviousClose,omitempty"`

	// RegularMarketOpen is the opening price.
	RegularMarketOpen float64 `json:"regularMarketOpen,omitempty"`

	// RegularMarketDayHigh is the day's high price.
	RegularMarketDayHigh float64 `json:"regularMarketDayHigh,omitempty"`

	// RegularMarketDayLow is the day's low price.
	RegularMarketDayLow float64 `json:"regularMarketDayLow,omitempty"`

	// RegularMarketVolume is the trading volume.
	RegularMarketVolume int64 `json:"regularMarketVolume,omitempty"`

	// MarketCap is the market capitalization.
	MarketCap int64 `json:"marketCap,omitempty"`

	// FiftyTwoWeekHigh is the 52-week high price.
	FiftyTwoWeekHigh float64 `json:"fiftyTwoWeekHigh,omitempty"`

	// FiftyTwoWeekLow is the 52-week low price.
	FiftyTwoWeekLow float64 `json:"fiftyTwoWeekLow,omitempty"`

	// Currency is the trading currency.
	Currency string `json:"currency,omitempty"`

	// MarketState is the current market state (PRE, REGULAR, POST, CLOSED).
	MarketState string `json:"marketState,omitempty"`
}

// LookupParams represents parameters for the Lookup function.
type LookupParams struct {
	// Type is the instrument type filter (default "all").
	Type LookupType

	// Count is the maximum number of results to return (default 25).
	Count int

	// Start is the offset for pagination (default 0).
	Start int

	// FetchPricingData includes real-time pricing data in results (default true).
	FetchPricingData bool
}

// DefaultLookupParams returns default lookup parameters.
//
// Default values:
//   - Type: LookupTypeAll
//   - Count: 25
//   - Start: 0
//   - FetchPricingData: true
func DefaultLookupParams() LookupParams {
	return LookupParams{
		Type:             LookupTypeAll,
		Count:            25,
		Start:            0,
		FetchPricingData: true,
	}
}

// LookupResponse represents the raw API response from Yahoo Finance lookup.
type LookupResponse struct {
	Finance struct {
		Result []struct {
			Documents []map[string]interface{} `json:"documents"`
			Count     int                      `json:"count,omitempty"`
			Start     int                      `json:"start,omitempty"`
			Total     int                      `json:"total,omitempty"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error,omitempty"`
	} `json:"finance"`
}
