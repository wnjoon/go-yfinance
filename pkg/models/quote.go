// Package models provides data structures for Yahoo Finance API responses.
package models

import "time"

// Quote represents current quote data for a ticker.
type Quote struct {
	// Symbol information
	Symbol    string `json:"symbol"`
	ShortName string `json:"shortName"`
	LongName  string `json:"longName"`
	QuoteType string `json:"quoteType"` // EQUITY, ETF, MUTUALFUND, etc.

	// Exchange information
	Exchange             string `json:"exchange"`
	ExchangeName         string `json:"exchangeName"`
	ExchangeTimezoneName string `json:"exchangeTimezoneName"`
	Currency             string `json:"currency"`

	// Regular market data
	RegularMarketPrice         float64   `json:"regularMarketPrice"`
	RegularMarketChange        float64   `json:"regularMarketChange"`
	RegularMarketChangePercent float64   `json:"regularMarketChangePercent"`
	RegularMarketDayHigh       float64   `json:"regularMarketDayHigh"`
	RegularMarketDayLow        float64   `json:"regularMarketDayLow"`
	RegularMarketOpen          float64   `json:"regularMarketOpen"`
	RegularMarketPreviousClose float64   `json:"regularMarketPreviousClose"`
	RegularMarketVolume        int64     `json:"regularMarketVolume"`
	RegularMarketTime          time.Time `json:"regularMarketTime"`

	// Pre/Post market data
	PreMarketPrice         float64   `json:"preMarketPrice,omitempty"`
	PreMarketChange        float64   `json:"preMarketChange,omitempty"`
	PreMarketChangePercent float64   `json:"preMarketChangePercent,omitempty"`
	PreMarketTime          time.Time `json:"preMarketTime,omitempty"`

	PostMarketPrice         float64   `json:"postMarketPrice,omitempty"`
	PostMarketChange        float64   `json:"postMarketChange,omitempty"`
	PostMarketChangePercent float64   `json:"postMarketChangePercent,omitempty"`
	PostMarketTime          time.Time `json:"postMarketTime,omitempty"`

	// 52-week range
	FiftyTwoWeekHigh       float64 `json:"fiftyTwoWeekHigh"`
	FiftyTwoWeekLow        float64 `json:"fiftyTwoWeekLow"`
	FiftyTwoWeekChange     float64 `json:"fiftyTwoWeekChange"`
	FiftyTwoWeekChangePerc float64 `json:"fiftyTwoWeekChangePercent"`

	// Moving averages
	FiftyDayAverage           float64 `json:"fiftyDayAverage"`
	FiftyDayAverageChange     float64 `json:"fiftyDayAverageChange"`
	FiftyDayAverageChangePerc float64 `json:"fiftyDayAverageChangePercent"`
	TwoHundredDayAverage      float64 `json:"twoHundredDayAverage"`
	TwoHundredDayAverageChg   float64 `json:"twoHundredDayAverageChange"`
	TwoHundredDayAverageChgPc float64 `json:"twoHundredDayAverageChangePercent"`

	// Volume averages
	AverageDailyVolume3Month int64 `json:"averageDailyVolume3Month"`
	AverageDailyVolume10Day  int64 `json:"averageDailyVolume10Day"`

	// Market cap and shares
	MarketCap        int64 `json:"marketCap"`
	SharesOutstanding int64 `json:"sharesOutstanding"`

	// Dividend info
	TrailingAnnualDividendRate  float64 `json:"trailingAnnualDividendRate,omitempty"`
	TrailingAnnualDividendYield float64 `json:"trailingAnnualDividendYield,omitempty"`
	DividendDate                int64   `json:"dividendDate,omitempty"`

	// Earnings info
	TrailingPE          float64 `json:"trailingPE,omitempty"`
	ForwardPE           float64 `json:"forwardPE,omitempty"`
	EpsTrailingTwelveMonths float64 `json:"epsTrailingTwelveMonths,omitempty"`
	EpsForward          float64 `json:"epsForward,omitempty"`
	EpsCurrentYear      float64 `json:"epsCurrentYear,omitempty"`

	// Book value
	BookValue         float64 `json:"bookValue,omitempty"`
	PriceToBook       float64 `json:"priceToBook,omitempty"`

	// Bid/Ask
	Bid     float64 `json:"bid,omitempty"`
	BidSize int64   `json:"bidSize,omitempty"`
	Ask     float64 `json:"ask,omitempty"`
	AskSize int64   `json:"askSize,omitempty"`

	// Market state
	MarketState string `json:"marketState"` // PRE, REGULAR, POST, CLOSED
}

// FastInfo represents a subset of quote data that can be fetched quickly.
type FastInfo struct {
	Currency                   string  `json:"currency"`
	QuoteType                  string  `json:"quoteType"`
	Exchange                   string  `json:"exchange"`
	Timezone                   string  `json:"timezone"`
	Shares                     int64   `json:"shares"`
	MarketCap                  float64 `json:"marketCap"`
	LastPrice                  float64 `json:"lastPrice"`
	PreviousClose              float64 `json:"previousClose"`
	Open                       float64 `json:"open"`
	DayHigh                    float64 `json:"dayHigh"`
	DayLow                     float64 `json:"dayLow"`
	RegularMarketPreviousClose float64 `json:"regularMarketPreviousClose"`
	LastVolume                 int64   `json:"lastVolume"`
	FiftyDayAverage            float64 `json:"fiftyDayAverage"`
	TwoHundredDayAverage       float64 `json:"twoHundredDayAverage"`
	TenDayAverageVolume        int64   `json:"tenDayAverageVolume"`
	ThreeMonthAverageVolume    int64   `json:"threeMonthAverageVolume"`
	YearHigh                   float64 `json:"yearHigh"`
	YearLow                    float64 `json:"yearLow"`
	YearChange                 float64 `json:"yearChange"`
}
