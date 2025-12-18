package models

// ChartResponse represents the response from Yahoo Finance chart API.
type ChartResponse struct {
	Chart struct {
		Result []ChartResult `json:"result"`
		Error  *ChartError   `json:"error"`
	} `json:"chart"`
}

// ChartResult represents a single chart result.
type ChartResult struct {
	Meta       ChartMeta         `json:"meta"`
	Timestamp  []int64           `json:"timestamp"`
	Events     *ChartEvents      `json:"events,omitempty"`
	Indicators ChartIndicators   `json:"indicators"`
}

// ChartError represents an error from the chart API.
type ChartError struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

// ChartEvents contains dividend and split events.
type ChartEvents struct {
	Dividends    map[string]DividendEvent    `json:"dividends,omitempty"`
	Splits       map[string]SplitEvent       `json:"splits,omitempty"`
	CapitalGains map[string]CapitalGainEvent `json:"capitalGains,omitempty"`
}

// DividendEvent represents a dividend event in chart response.
type DividendEvent struct {
	Amount float64 `json:"amount"`
	Date   int64   `json:"date"`
}

// SplitEvent represents a stock split event in chart response.
type SplitEvent struct {
	Date        int64   `json:"date"`
	Numerator   float64 `json:"numerator"`
	Denominator float64 `json:"denominator"`
	SplitRatio  string  `json:"splitRatio"`
}

// CapitalGainEvent represents a capital gain distribution event.
type CapitalGainEvent struct {
	Amount float64 `json:"amount"`
	Date   int64   `json:"date"`
}

// ChartIndicators contains OHLCV data.
type ChartIndicators struct {
	Quote    []ChartQuote    `json:"quote"`
	AdjClose []ChartAdjClose `json:"adjclose,omitempty"`
}

// ChartQuote contains OHLCV arrays.
type ChartQuote struct {
	Open   []*float64 `json:"open"`
	High   []*float64 `json:"high"`
	Low    []*float64 `json:"low"`
	Close  []*float64 `json:"close"`
	Volume []*int64   `json:"volume"`
}

// ChartAdjClose contains adjusted close prices.
type ChartAdjClose struct {
	AdjClose []*float64 `json:"adjclose"`
}

// QuoteSummaryResponse represents the response from quoteSummary API.
type QuoteSummaryResponse struct {
	QuoteSummary struct {
		Result []QuoteSummaryResult `json:"result"`
		Error  *QuoteSummaryError   `json:"error"`
	} `json:"quoteSummary"`
}

// QuoteSummaryResult contains all module data.
type QuoteSummaryResult struct {
	AssetProfile         map[string]interface{} `json:"assetProfile,omitempty"`
	SummaryDetail        map[string]interface{} `json:"summaryDetail,omitempty"`
	DefaultKeyStatistics map[string]interface{} `json:"defaultKeyStatistics,omitempty"`
	FinancialData        map[string]interface{} `json:"financialData,omitempty"`
	QuoteType            map[string]interface{} `json:"quoteType,omitempty"`
	Price                map[string]interface{} `json:"price,omitempty"`
	CalendarEvents       map[string]interface{} `json:"calendarEvents,omitempty"`
}

// QuoteSummaryError represents an error from quoteSummary API.
type QuoteSummaryError struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

// QuoteResponse represents the response from quote API (v7).
type QuoteResponse struct {
	QuoteResponse struct {
		Result []QuoteResult `json:"result"`
		Error  *QuoteError   `json:"error"`
	} `json:"quoteResponse"`
}

// QuoteResult represents a single quote result.
type QuoteResult struct {
	Language                     string  `json:"language"`
	Region                       string  `json:"region"`
	QuoteType                    string  `json:"quoteType"`
	TypeDisp                     string  `json:"typeDisp"`
	QuoteSourceName              string  `json:"quoteSourceName"`
	Triggerable                  bool    `json:"triggerable"`
	CustomPriceAlertConfidence   string  `json:"customPriceAlertConfidence"`
	Currency                     string  `json:"currency"`
	Exchange                     string  `json:"exchange"`
	ShortName                    string  `json:"shortName"`
	LongName                     string  `json:"longName"`
	MessageBoardID               string  `json:"messageBoardId"`
	ExchangeTimezoneName         string  `json:"exchangeTimezoneName"`
	ExchangeTimezoneShortName    string  `json:"exchangeTimezoneShortName"`
	GMTOffSetMilliseconds        int64   `json:"gmtOffSetMilliseconds"`
	Market                       string  `json:"market"`
	EsgPopulated                 bool    `json:"esgPopulated"`
	MarketState                  string  `json:"marketState"`
	RegularMarketChangePercent   float64 `json:"regularMarketChangePercent"`
	RegularMarketPrice           float64 `json:"regularMarketPrice"`
	RegularMarketChange          float64 `json:"regularMarketChange"`
	RegularMarketTime            int64   `json:"regularMarketTime"`
	RegularMarketDayHigh         float64 `json:"regularMarketDayHigh"`
	RegularMarketDayRange        string  `json:"regularMarketDayRange"`
	RegularMarketDayLow          float64 `json:"regularMarketDayLow"`
	RegularMarketVolume          int64   `json:"regularMarketVolume"`
	RegularMarketPreviousClose   float64 `json:"regularMarketPreviousClose"`
	RegularMarketOpen            float64 `json:"regularMarketOpen"`
	Bid                          float64 `json:"bid"`
	Ask                          float64 `json:"ask"`
	BidSize                      int64   `json:"bidSize"`
	AskSize                      int64   `json:"askSize"`
	FullExchangeName             string  `json:"fullExchangeName"`
	FinancialCurrency            string  `json:"financialCurrency"`
	SourceInterval               int     `json:"sourceInterval"`
	ExchangeDataDelayedBy        int     `json:"exchangeDataDelayedBy"`
	Tradeable                    bool    `json:"tradeable"`
	CryptoTradeable              bool    `json:"cryptoTradeable"`
	FirstTradeDateMilliseconds   int64   `json:"firstTradeDateMilliseconds"`
	PriceHint                    int     `json:"priceHint"`
	PostMarketChangePercent      float64 `json:"postMarketChangePercent"`
	PostMarketTime               int64   `json:"postMarketTime"`
	PostMarketPrice              float64 `json:"postMarketPrice"`
	PostMarketChange             float64 `json:"postMarketChange"`
	PreMarketChangePercent       float64 `json:"preMarketChangePercent"`
	PreMarketTime                int64   `json:"preMarketTime"`
	PreMarketPrice               float64 `json:"preMarketPrice"`
	PreMarketChange              float64 `json:"preMarketChange"`
	FiftyTwoWeekLowChange        float64 `json:"fiftyTwoWeekLowChange"`
	FiftyTwoWeekLowChangePercent float64 `json:"fiftyTwoWeekLowChangePercent"`
	FiftyTwoWeekRange            string  `json:"fiftyTwoWeekRange"`
	FiftyTwoWeekHighChange       float64 `json:"fiftyTwoWeekHighChange"`
	FiftyTwoWeekHighChangePercent float64 `json:"fiftyTwoWeekHighChangePercent"`
	FiftyTwoWeekLow              float64 `json:"fiftyTwoWeekLow"`
	FiftyTwoWeekHigh             float64 `json:"fiftyTwoWeekHigh"`
	FiftyTwoWeekChangePercent    float64 `json:"fiftyTwoWeekChangePercent"`
	DividendDate                 int64   `json:"dividendDate"`
	EarningsTimestamp            int64   `json:"earningsTimestamp"`
	EarningsTimestampStart       int64   `json:"earningsTimestampStart"`
	EarningsTimestampEnd         int64   `json:"earningsTimestampEnd"`
	TrailingAnnualDividendRate   float64 `json:"trailingAnnualDividendRate"`
	TrailingPE                   float64 `json:"trailingPE"`
	TrailingAnnualDividendYield  float64 `json:"trailingAnnualDividendYield"`
	EpsTrailingTwelveMonths      float64 `json:"epsTrailingTwelveMonths"`
	EpsForward                   float64 `json:"epsForward"`
	EpsCurrentYear               float64 `json:"epsCurrentYear"`
	PriceEpsCurrentYear          float64 `json:"priceEpsCurrentYear"`
	SharesOutstanding            int64   `json:"sharesOutstanding"`
	BookValue                    float64 `json:"bookValue"`
	FiftyDayAverage              float64 `json:"fiftyDayAverage"`
	FiftyDayAverageChange        float64 `json:"fiftyDayAverageChange"`
	FiftyDayAverageChangePercent float64 `json:"fiftyDayAverageChangePercent"`
	TwoHundredDayAverage         float64 `json:"twoHundredDayAverage"`
	TwoHundredDayAverageChange   float64 `json:"twoHundredDayAverageChange"`
	TwoHundredDayAverageChangePercent float64 `json:"twoHundredDayAverageChangePercent"`
	MarketCap                    int64   `json:"marketCap"`
	ForwardPE                    float64 `json:"forwardPE"`
	PriceToBook                  float64 `json:"priceToBook"`
	AverageDailyVolume10Day      int64   `json:"averageDailyVolume10Day"`
	AverageDailyVolume3Month     int64   `json:"averageDailyVolume3Month"`
	DisplayName                  string  `json:"displayName"`
	Symbol                       string  `json:"symbol"`
}

// QuoteError represents an error from quote API.
type QuoteError struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}
