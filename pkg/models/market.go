package models

import "time"

// MarketStatus represents the status and trading hours of a market.
//
// This includes opening/closing times, timezone information,
// and current market state.
//
// Example:
//
//	m, err := market.New("us_market")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	status, err := m.Status()
//	fmt.Printf("Market opens at: %s\n", status.Open.Format("15:04"))
type MarketStatus struct {
	// ID is the market identifier (e.g., "us_market").
	ID string `json:"id,omitempty"`

	// Open is the market opening time.
	Open *time.Time `json:"open,omitempty"`

	// Close is the market closing time.
	Close *time.Time `json:"close,omitempty"`

	// Timezone contains timezone information.
	Timezone *MarketTimezone `json:"timezone,omitempty"`

	// State is the current market state (e.g., "REGULAR", "CLOSED", "PRE", "POST").
	State string `json:"state,omitempty"`

	// OpenStr is the raw open time string from API.
	OpenStr string `json:"-"`

	// CloseStr is the raw close time string from API.
	CloseStr string `json:"-"`
}

// MarketTimezone represents timezone information for a market.
type MarketTimezone struct {
	// GMTOffset is the GMT offset in milliseconds.
	GMTOffset int64 `json:"gmtoffset,omitempty"`

	// Short is the short timezone name (e.g., "EST", "PST").
	Short string `json:"short,omitempty"`

	// Long is the full timezone name (e.g., "Eastern Standard Time").
	Long string `json:"long,omitempty"`
}

// MarketSummaryItem represents a single market index or asset in the summary.
//
// Each item represents an index like S&P 500, Dow Jones, etc.
type MarketSummaryItem struct {
	// Exchange is the exchange code (e.g., "SNP", "DJI").
	Exchange string `json:"exchange"`

	// Symbol is the ticker symbol.
	Symbol string `json:"symbol"`

	// ShortName is the short name of the index.
	ShortName string `json:"shortName"`

	// FullExchangeName is the full name of the exchange.
	FullExchangeName string `json:"fullExchangeName,omitempty"`

	// MarketState is the current market state.
	MarketState string `json:"marketState,omitempty"`

	// RegularMarketPrice is the current price.
	RegularMarketPrice float64 `json:"regularMarketPrice"`

	// RegularMarketChange is the price change.
	RegularMarketChange float64 `json:"regularMarketChange"`

	// RegularMarketChangePercent is the percentage change.
	RegularMarketChangePercent float64 `json:"regularMarketChangePercent"`

	// RegularMarketPreviousClose is the previous closing price.
	RegularMarketPreviousClose float64 `json:"regularMarketPreviousClose,omitempty"`

	// RegularMarketTime is the time of the last regular market trade.
	RegularMarketTime int64 `json:"regularMarketTime,omitempty"`

	// QuoteType is the type of quote (e.g., "INDEX", "EQUITY").
	QuoteType string `json:"quoteType,omitempty"`

	// SourceInterval is the data update interval in seconds.
	SourceInterval int `json:"sourceInterval,omitempty"`

	// ExchangeDataDelayedBy is the delay in seconds.
	ExchangeDataDelayedBy int `json:"exchangeDataDelayedBy,omitempty"`
}

// MarketSummary represents the summary of all market indices.
//
// This provides an overview of major market indices like S&P 500,
// Dow Jones, NASDAQ, etc.
//
// Example:
//
//	m, err := market.New("us_market")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	summary, err := m.Summary()
//	for exchange, item := range summary {
//	    fmt.Printf("%s: %.2f (%.2f%%)\n",
//	        item.ShortName, item.RegularMarketPrice, item.RegularMarketChangePercent)
//	}
type MarketSummary map[string]MarketSummaryItem

// MarketSummaryResponse represents the raw API response for market summary.
type MarketSummaryResponse struct {
	MarketSummaryResponse struct {
		Result []map[string]interface{} `json:"result"`
		Error  *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error,omitempty"`
	} `json:"marketSummaryResponse"`
}

// MarketTimeResponse represents the raw API response for market time.
type MarketTimeResponse struct {
	Finance struct {
		MarketTimes []struct {
			ID         string `json:"id"`
			MarketTime []struct {
				ID       string                   `json:"id"`
				Open     string                   `json:"open"`
				Close    string                   `json:"close"`
				Timezone []map[string]interface{} `json:"timezone"`
				Time     string                   `json:"time,omitempty"`
			} `json:"marketTime"`
		} `json:"marketTimes"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error,omitempty"`
	} `json:"finance"`
}

// MarketRegion represents Yahoo market regions accepted by the market summary endpoint.
type MarketRegion string

const (
	// MarketRegionUS represents U.S. markets.
	MarketRegionUS MarketRegion = "US"

	// MarketRegionGB represents Great Britain markets.
	MarketRegionGB MarketRegion = "GB"

	// MarketRegionAsia represents Asian markets.
	MarketRegionAsia MarketRegion = "ASIA"

	// MarketRegionEurope represents European markets.
	MarketRegionEurope MarketRegion = "EUROPE"

	// MarketRegionRates represents rates markets.
	MarketRegionRates MarketRegion = "RATES"

	// MarketRegionCommodities represents commodities markets.
	MarketRegionCommodities MarketRegion = "COMMODITIES"

	// MarketRegionCurrencies represents currencies markets.
	MarketRegionCurrencies MarketRegion = "CURRENCIES"

	// MarketRegionCryptocurrencies represents cryptocurrency markets.
	MarketRegionCryptocurrencies MarketRegion = "CRYPTOCURRENCIES"
)

// PredefinedMarket represents legacy market identifiers.
// New code can use MarketRegion with market.NewWithRegion for Python v1.4.0 parity.
type PredefinedMarket string

const (
	// MarketUS represents the US market.
	MarketUS PredefinedMarket = "us_market"

	// MarketGB represents the UK market.
	MarketGB PredefinedMarket = "gb_market"

	// MarketDE represents the German market.
	MarketDE PredefinedMarket = "de_market"

	// MarketFR represents the French market.
	MarketFR PredefinedMarket = "fr_market"

	// MarketJP represents the Japanese market.
	MarketJP PredefinedMarket = "jp_market"

	// MarketHK represents the Hong Kong market.
	MarketHK PredefinedMarket = "hk_market"

	// MarketCN represents the Chinese market.
	MarketCN PredefinedMarket = "cn_market"

	// MarketCA represents the Canadian market.
	MarketCA PredefinedMarket = "ca_market"

	// MarketAU represents the Australian market.
	MarketAU PredefinedMarket = "au_market"

	// MarketIN represents the Indian market.
	MarketIN PredefinedMarket = "in_market"

	// MarketKR represents the Korean market.
	MarketKR PredefinedMarket = "kr_market"

	// MarketBR represents the Brazilian market.
	MarketBR PredefinedMarket = "br_market"
)
