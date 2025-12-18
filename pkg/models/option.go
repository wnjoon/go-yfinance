// Package models provides data structures for Yahoo Finance API responses.
package models

import "time"

// Option represents a single option contract (call or put).
type Option struct {
	ContractSymbol    string  `json:"contractSymbol"`
	Strike            float64 `json:"strike"`
	Currency          string  `json:"currency"`
	LastPrice         float64 `json:"lastPrice"`
	Change            float64 `json:"change"`
	PercentChange     float64 `json:"percentChange"`
	Volume            int64   `json:"volume"`
	OpenInterest      int64   `json:"openInterest"`
	Bid               float64 `json:"bid"`
	Ask               float64 `json:"ask"`
	ContractSize      string  `json:"contractSize"`
	Expiration        int64   `json:"expiration"`
	LastTradeDate     int64   `json:"lastTradeDate"`
	ImpliedVolatility float64 `json:"impliedVolatility"`
	InTheMoney        bool    `json:"inTheMoney"`
}

// LastTradeDatetime returns the last trade date as time.Time.
func (o *Option) LastTradeDatetime() time.Time {
	return time.Unix(o.LastTradeDate, 0)
}

// ExpirationDatetime returns the expiration date as time.Time.
func (o *Option) ExpirationDatetime() time.Time {
	return time.Unix(o.Expiration, 0)
}

// OptionChain represents the complete option chain for a symbol.
type OptionChain struct {
	Calls      []Option       `json:"calls"`
	Puts       []Option       `json:"puts"`
	Underlying *Quote         `json:"underlying,omitempty"`
	Expiration time.Time      `json:"expiration"`
}

// OptionsData holds all expiration dates and the current option chain.
type OptionsData struct {
	ExpirationDates []time.Time  `json:"expirationDates"`
	Strikes         []float64    `json:"strikes"`
	HasMiniOptions  bool         `json:"hasMiniOptions"`
	OptionChain     *OptionChain `json:"optionChain,omitempty"`
}

// OptionChainResponse represents the API response for options data.
type OptionChainResponse struct {
	OptionChain struct {
		Result []struct {
			UnderlyingSymbol string    `json:"underlyingSymbol"`
			ExpirationDates  []int64   `json:"expirationDates"`
			Strikes          []float64 `json:"strikes"`
			HasMiniOptions   bool      `json:"hasMiniOptions"`
			Quote            Quote     `json:"quote"`
			Options          []struct {
				ExpirationDate int64    `json:"expirationDate"`
				HasMiniOptions bool     `json:"hasMiniOptions"`
				Calls          []Option `json:"calls"`
				Puts           []Option `json:"puts"`
			} `json:"options"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"optionChain"`
}
