package models

// ScreenerResult represents the result from a stock screener query.
//
// Example:
//
//	result, err := screener.Screen(screener.DayGainers, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Found %d stocks\n", result.Total)
//	for _, quote := range result.Quotes {
//	    fmt.Printf("%s: %.2f%%\n", quote.Symbol, quote.PercentChange)
//	}
type ScreenerResult struct {
	// Total is the total number of matching results.
	Total int `json:"total"`

	// Count is the number of results returned in this batch.
	Count int `json:"count"`

	// Offset is the offset used for pagination.
	Offset int `json:"offset"`

	// Quotes contains the matching stocks.
	Quotes []ScreenerQuote `json:"quotes"`
}

// ScreenerQuote represents a single stock from screener results.
type ScreenerQuote struct {
	// Symbol is the ticker symbol.
	Symbol string `json:"symbol"`

	// ShortName is the short company name.
	ShortName string `json:"shortName,omitempty"`

	// LongName is the full company name.
	LongName string `json:"longName,omitempty"`

	// Exchange is the exchange where the stock is traded.
	Exchange string `json:"exchange,omitempty"`

	// ExchangeDisp is the display name of the exchange.
	ExchangeDisp string `json:"fullExchangeName,omitempty"`

	// QuoteType is the type of asset.
	QuoteType string `json:"quoteType,omitempty"`

	// Region is the geographic region.
	Region string `json:"region,omitempty"`

	// Sector is the sector of the company.
	Sector string `json:"sector,omitempty"`

	// Industry is the industry of the company.
	Industry string `json:"industry,omitempty"`

	// Currency is the trading currency.
	Currency string `json:"currency,omitempty"`

	// MarketState is the current market state (PRE, REGULAR, POST, CLOSED).
	MarketState string `json:"marketState,omitempty"`

	// RegularMarketPrice is the current/last regular market price.
	RegularMarketPrice float64 `json:"regularMarketPrice,omitempty"`

	// RegularMarketChange is the price change.
	RegularMarketChange float64 `json:"regularMarketChange,omitempty"`

	// RegularMarketChangePercent is the percentage change.
	RegularMarketChangePercent float64 `json:"regularMarketChangePercent,omitempty"`

	// RegularMarketVolume is the trading volume.
	RegularMarketVolume int64 `json:"regularMarketVolume,omitempty"`

	// RegularMarketDayHigh is the day's high price.
	RegularMarketDayHigh float64 `json:"regularMarketDayHigh,omitempty"`

	// RegularMarketDayLow is the day's low price.
	RegularMarketDayLow float64 `json:"regularMarketDayLow,omitempty"`

	// RegularMarketOpen is the opening price.
	RegularMarketOpen float64 `json:"regularMarketOpen,omitempty"`

	// RegularMarketPreviousClose is the previous close price.
	RegularMarketPreviousClose float64 `json:"regularMarketPreviousClose,omitempty"`

	// MarketCap is the market capitalization.
	MarketCap int64 `json:"marketCap,omitempty"`

	// FiftyTwoWeekHigh is the 52-week high price.
	FiftyTwoWeekHigh float64 `json:"fiftyTwoWeekHigh,omitempty"`

	// FiftyTwoWeekLow is the 52-week low price.
	FiftyTwoWeekLow float64 `json:"fiftyTwoWeekLow,omitempty"`

	// FiftyTwoWeekChange is the 52-week change percentage.
	FiftyTwoWeekChange float64 `json:"fiftyTwoWeekChangePercent,omitempty"`

	// FiftyDayAverage is the 50-day average price.
	FiftyDayAverage float64 `json:"fiftyDayAverage,omitempty"`

	// TwoHundredDayAverage is the 200-day average price.
	TwoHundredDayAverage float64 `json:"twoHundredDayAverage,omitempty"`

	// AverageVolume is the average trading volume.
	AverageVolume int64 `json:"averageDailyVolume3Month,omitempty"`

	// TrailingPE is the trailing P/E ratio.
	TrailingPE float64 `json:"trailingPE,omitempty"`

	// ForwardPE is the forward P/E ratio.
	ForwardPE float64 `json:"forwardPE,omitempty"`

	// PriceToBook is the price to book ratio.
	PriceToBook float64 `json:"priceToBook,omitempty"`

	// DividendYield is the dividend yield.
	DividendYield float64 `json:"dividendYield,omitempty"`

	// TrailingEPS is the trailing earnings per share.
	TrailingEPS float64 `json:"epsTrailingTwelveMonths,omitempty"`

	// BookValue is the book value per share.
	BookValue float64 `json:"bookValue,omitempty"`
}

// PredefinedScreener represents a predefined screener query name.
type PredefinedScreener string

const (
	// Equity Screeners
	ScreenerAggressiveSmallCaps   PredefinedScreener = "aggressive_small_caps"
	ScreenerDayGainers            PredefinedScreener = "day_gainers"
	ScreenerDayLosers             PredefinedScreener = "day_losers"
	ScreenerGrowthTech            PredefinedScreener = "growth_technology_stocks"
	ScreenerMostActives           PredefinedScreener = "most_actives"
	ScreenerMostShorted           PredefinedScreener = "most_shorted_stocks"
	ScreenerSmallCapGainers       PredefinedScreener = "small_cap_gainers"
	ScreenerUndervaluedGrowth     PredefinedScreener = "undervalued_growth_stocks"
	ScreenerUndervaluedLargeCaps  PredefinedScreener = "undervalued_large_caps"

	// Fund Screeners
	ScreenerConservativeForeign   PredefinedScreener = "conservative_foreign_funds"
	ScreenerHighYieldBond         PredefinedScreener = "high_yield_bond"
	ScreenerPortfolioAnchors      PredefinedScreener = "portfolio_anchors"
	ScreenerSolidLargeGrowth      PredefinedScreener = "solid_large_growth_funds"
	ScreenerSolidMidcapGrowth     PredefinedScreener = "solid_midcap_growth_funds"
	ScreenerTopMutualFunds        PredefinedScreener = "top_mutual_funds"
)

// AllPredefinedScreeners returns all available predefined screener names.
func AllPredefinedScreeners() []PredefinedScreener {
	return []PredefinedScreener{
		ScreenerAggressiveSmallCaps,
		ScreenerDayGainers,
		ScreenerDayLosers,
		ScreenerGrowthTech,
		ScreenerMostActives,
		ScreenerMostShorted,
		ScreenerSmallCapGainers,
		ScreenerUndervaluedGrowth,
		ScreenerUndervaluedLargeCaps,
		ScreenerConservativeForeign,
		ScreenerHighYieldBond,
		ScreenerPortfolioAnchors,
		ScreenerSolidLargeGrowth,
		ScreenerSolidMidcapGrowth,
		ScreenerTopMutualFunds,
	}
}

// ScreenerParams represents parameters for the Screen function.
type ScreenerParams struct {
	// Offset is the result offset for pagination (default 0).
	Offset int

	// Count is the number of results to return (default 25, max 250).
	Count int

	// SortField is the field to sort by (default "ticker").
	SortField string

	// SortAsc sorts in ascending order if true (default false/descending).
	SortAsc bool
}

// DefaultScreenerParams returns default screener parameters.
func DefaultScreenerParams() ScreenerParams {
	return ScreenerParams{
		Offset:    0,
		Count:     25,
		SortField: "ticker",
		SortAsc:   false,
	}
}

// QueryOperator represents an operator for custom screener queries.
type QueryOperator string

const (
	// Comparison operators
	OpEQ   QueryOperator = "eq"   // Equals
	OpGT   QueryOperator = "gt"   // Greater than
	OpLT   QueryOperator = "lt"   // Less than
	OpGTE  QueryOperator = "gte"  // Greater than or equal
	OpLTE  QueryOperator = "lte"  // Less than or equal
	OpBTWN QueryOperator = "btwn" // Between

	// Logical operators
	OpAND QueryOperator = "and" // All conditions must match
	OpOR  QueryOperator = "or"  // Any condition can match
)

// ScreenerQuery represents a custom screener query.
type ScreenerQuery struct {
	// Operator is the query operator (eq, gt, lt, and, or, etc.).
	Operator QueryOperator `json:"operator"`

	// Operands contains the operands for this query.
	// For comparison operators: [fieldName, value] or [fieldName, min, max] for btwn
	// For logical operators: nested ScreenerQuery objects
	Operands []interface{} `json:"operands"`
}

// NewEquityQuery creates a new equity screener query.
//
// Example:
//
//	// Find US tech stocks with price > $10
//	q := models.NewEquityQuery(models.OpAND, []interface{}{
//	    models.NewEquityQuery(models.OpEQ, []interface{}{"region", "us"}),
//	    models.NewEquityQuery(models.OpEQ, []interface{}{"sector", "Technology"}),
//	    models.NewEquityQuery(models.OpGT, []interface{}{"eodprice", 10}),
//	})
func NewEquityQuery(op QueryOperator, operands []interface{}) *ScreenerQuery {
	return &ScreenerQuery{
		Operator: op,
		Operands: operands,
	}
}

// ScreenerResponse represents the raw API response from Yahoo Finance screener.
type ScreenerResponse struct {
	Finance struct {
		Result []struct {
			Total  int                      `json:"total"`
			Count  int                      `json:"count"`
			Quotes []map[string]interface{} `json:"quotes"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"finance"`
}
