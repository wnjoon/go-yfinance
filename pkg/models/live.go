package models

import "time"

// PricingData represents real-time pricing data from Yahoo Finance WebSocket.
//
// This structure contains live market data including price, volume, bid/ask,
// and various market indicators.
type PricingData struct {
	// ID is the ticker symbol.
	ID string `json:"id,omitempty"`

	// Price is the current price.
	Price float32 `json:"price,omitempty"`

	// Time is the quote timestamp (Unix timestamp).
	Time int64 `json:"time,omitempty"`

	// Currency is the trading currency (e.g., "USD").
	Currency string `json:"currency,omitempty"`

	// Exchange is the exchange name (e.g., "NMS").
	Exchange string `json:"exchange,omitempty"`

	// QuoteType indicates the type of quote (1=equity, 2=option, etc.).
	QuoteType int32 `json:"quote_type,omitempty"`

	// MarketHours indicates market state (0=pre, 1=regular, 2=post, 3=closed).
	MarketHours int32 `json:"market_hours,omitempty"`

	// ChangePercent is the percentage change from previous close.
	ChangePercent float32 `json:"change_percent,omitempty"`

	// DayVolume is the trading volume for the day.
	DayVolume int64 `json:"day_volume,omitempty"`

	// DayHigh is the day's high price.
	DayHigh float32 `json:"day_high,omitempty"`

	// DayLow is the day's low price.
	DayLow float32 `json:"day_low,omitempty"`

	// Change is the absolute change from previous close.
	Change float32 `json:"change,omitempty"`

	// ShortName is the company short name.
	ShortName string `json:"short_name,omitempty"`

	// ExpireDate is the option expiration date (Unix timestamp).
	ExpireDate int64 `json:"expire_date,omitempty"`

	// OpenPrice is the day's opening price.
	OpenPrice float32 `json:"open_price,omitempty"`

	// PreviousClose is the previous day's closing price.
	PreviousClose float32 `json:"previous_close,omitempty"`

	// StrikePrice is the option strike price.
	StrikePrice float32 `json:"strike_price,omitempty"`

	// UnderlyingSymbol is the underlying symbol for options.
	UnderlyingSymbol string `json:"underlying_symbol,omitempty"`

	// OpenInterest is the option open interest.
	OpenInterest int64 `json:"open_interest,omitempty"`

	// OptionsType indicates call (0) or put (1).
	OptionsType int64 `json:"options_type,omitempty"`

	// MiniOption indicates if this is a mini option.
	MiniOption int64 `json:"mini_option,omitempty"`

	// LastSize is the last trade size.
	LastSize int64 `json:"last_size,omitempty"`

	// Bid is the current bid price.
	Bid float32 `json:"bid,omitempty"`

	// BidSize is the bid size.
	BidSize int64 `json:"bid_size,omitempty"`

	// Ask is the current ask price.
	Ask float32 `json:"ask,omitempty"`

	// AskSize is the ask size.
	AskSize int64 `json:"ask_size,omitempty"`

	// PriceHint is the decimal precision hint.
	PriceHint int64 `json:"price_hint,omitempty"`

	// Vol24Hr is 24-hour volume (for crypto).
	Vol24Hr int64 `json:"vol_24hr,omitempty"`

	// VolAllCurrencies is volume in all currencies (for crypto).
	VolAllCurrencies int64 `json:"vol_all_currencies,omitempty"`

	// FromCurrency is the source currency (for crypto).
	FromCurrency string `json:"from_currency,omitempty"`

	// LastMarket is the last market (for crypto).
	LastMarket string `json:"last_market,omitempty"`

	// CirculatingSupply is the circulating supply (for crypto).
	CirculatingSupply float64 `json:"circulating_supply,omitempty"`

	// MarketCap is the market capitalization.
	MarketCap float64 `json:"market_cap,omitempty"`
}

// Timestamp returns the quote time as time.Time.
func (p *PricingData) Timestamp() time.Time {
	return time.Unix(p.Time, 0)
}

// ExpireTime returns the option expiration date as time.Time.
func (p *PricingData) ExpireTime() time.Time {
	if p.ExpireDate == 0 {
		return time.Time{}
	}
	return time.Unix(p.ExpireDate, 0)
}

// IsRegularMarket returns true if trading during regular market hours.
func (p *PricingData) IsRegularMarket() bool {
	return p.MarketHours == 1
}

// IsPreMarket returns true if trading during pre-market hours.
func (p *PricingData) IsPreMarket() bool {
	return p.MarketHours == 0
}

// IsPostMarket returns true if trading during post-market hours.
func (p *PricingData) IsPostMarket() bool {
	return p.MarketHours == 2
}

// MarketState represents the market hours state.
type MarketState int32

const (
	// MarketStatePreMarket indicates pre-market hours.
	MarketStatePreMarket MarketState = 0
	// MarketStateRegular indicates regular trading hours.
	MarketStateRegular MarketState = 1
	// MarketStatePostMarket indicates post-market hours.
	MarketStatePostMarket MarketState = 2
	// MarketStateClosed indicates market is closed.
	MarketStateClosed MarketState = 3
)

// String returns the string representation of MarketState.
func (m MarketState) String() string {
	switch m {
	case MarketStatePreMarket:
		return "PRE_MARKET"
	case MarketStateRegular:
		return "REGULAR"
	case MarketStatePostMarket:
		return "POST_MARKET"
	case MarketStateClosed:
		return "CLOSED"
	default:
		return "UNKNOWN"
	}
}
