package ticker

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/client"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

// Quote fetches the current quote for the ticker.
func (t *Ticker) Quote() (*models.Quote, error) {
	params := url.Values{}
	params.Set("symbols", t.symbol)
	params.Set("formatted", "false")

	resp, err := t.getWithCrumb(t.quoteURL(), params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quote: %w", err)
	}

	var quoteResp models.QuoteResponse
	if err := json.Unmarshal([]byte(resp.Body), &quoteResp); err != nil {
		return nil, client.WrapInvalidResponseError(err)
	}

	if quoteResp.QuoteResponse.Error != nil {
		return nil, fmt.Errorf("API error: %s", quoteResp.QuoteResponse.Error.Description)
	}

	if len(quoteResp.QuoteResponse.Result) == 0 {
		return nil, client.WrapNotFoundError(t.symbol)
	}

	result := quoteResp.QuoteResponse.Result[0]

	quote := &models.Quote{
		Symbol:                     result.Symbol,
		ShortName:                  result.ShortName,
		LongName:                   result.LongName,
		QuoteType:                  result.QuoteType,
		Exchange:                   result.Exchange,
		ExchangeName:               result.FullExchangeName,
		ExchangeTimezoneName:       result.ExchangeTimezoneName,
		Currency:                   result.Currency,
		RegularMarketPrice:         result.RegularMarketPrice,
		RegularMarketChange:        result.RegularMarketChange,
		RegularMarketChangePercent: result.RegularMarketChangePercent,
		RegularMarketDayHigh:       result.RegularMarketDayHigh,
		RegularMarketDayLow:        result.RegularMarketDayLow,
		RegularMarketOpen:          result.RegularMarketOpen,
		RegularMarketPreviousClose: result.RegularMarketPreviousClose,
		RegularMarketVolume:        result.RegularMarketVolume,
		RegularMarketTime:          time.Unix(result.RegularMarketTime, 0),
		PreMarketPrice:             result.PreMarketPrice,
		PreMarketChange:            result.PreMarketChange,
		PreMarketChangePercent:     result.PreMarketChangePercent,
		PreMarketTime:              time.Unix(result.PreMarketTime, 0),
		PostMarketPrice:            result.PostMarketPrice,
		PostMarketChange:           result.PostMarketChange,
		PostMarketChangePercent:    result.PostMarketChangePercent,
		PostMarketTime:             time.Unix(result.PostMarketTime, 0),
		FiftyTwoWeekHigh:           result.FiftyTwoWeekHigh,
		FiftyTwoWeekLow:            result.FiftyTwoWeekLow,
		FiftyTwoWeekChangePerc:     result.FiftyTwoWeekChangePercent,
		FiftyDayAverage:            result.FiftyDayAverage,
		FiftyDayAverageChange:      result.FiftyDayAverageChange,
		FiftyDayAverageChangePerc:  result.FiftyDayAverageChangePercent,
		TwoHundredDayAverage:       result.TwoHundredDayAverage,
		TwoHundredDayAverageChg:    result.TwoHundredDayAverageChange,
		TwoHundredDayAverageChgPc:  result.TwoHundredDayAverageChangePercent,
		AverageDailyVolume3Month:   result.AverageDailyVolume3Month,
		AverageDailyVolume10Day:    result.AverageDailyVolume10Day,
		MarketCap:                  result.MarketCap,
		SharesOutstanding:          result.SharesOutstanding,
		TrailingAnnualDividendRate: result.TrailingAnnualDividendRate,
		TrailingAnnualDividendYield: result.TrailingAnnualDividendYield,
		DividendDate:               result.DividendDate,
		TrailingPE:                 result.TrailingPE,
		ForwardPE:                  result.ForwardPE,
		EpsTrailingTwelveMonths:    result.EpsTrailingTwelveMonths,
		EpsForward:                 result.EpsForward,
		EpsCurrentYear:             result.EpsCurrentYear,
		BookValue:                  result.BookValue,
		PriceToBook:                result.PriceToBook,
		Bid:                        result.Bid,
		BidSize:                    result.BidSize,
		Ask:                        result.Ask,
		AskSize:                    result.AskSize,
		MarketState:                result.MarketState,
	}

	// Cache the quote
	t.mu.Lock()
	t.quoteCache = quote
	t.mu.Unlock()

	return quote, nil
}

// FastInfo returns a FastInfo struct with commonly used data.
// This fetches data from the history endpoint which can be faster for some fields.
func (t *Ticker) FastInfo() (*models.FastInfo, error) {
	// First, ensure we have history metadata
	if t.GetHistoryMetadata() == nil {
		_, err := t.History(models.HistoryParams{Period: "5d", Interval: "1d"})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch history for fast info: %w", err)
		}
	}

	meta := t.GetHistoryMetadata()
	if meta == nil {
		return nil, fmt.Errorf("no metadata available")
	}

	// Get latest quote data
	quote, err := t.Quote()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quote: %w", err)
	}

	return &models.FastInfo{
		Currency:                   meta.Currency,
		QuoteType:                  meta.InstrumentType,
		Exchange:                   meta.ExchangeName,
		Timezone:                   meta.ExchangeTimezoneName,
		Shares:                     quote.SharesOutstanding,
		MarketCap:                  float64(quote.MarketCap),
		LastPrice:                  quote.RegularMarketPrice,
		PreviousClose:              quote.RegularMarketPreviousClose,
		Open:                       quote.RegularMarketOpen,
		DayHigh:                    quote.RegularMarketDayHigh,
		DayLow:                     quote.RegularMarketDayLow,
		RegularMarketPreviousClose: quote.RegularMarketPreviousClose,
		LastVolume:                 quote.RegularMarketVolume,
		FiftyDayAverage:            quote.FiftyDayAverage,
		TwoHundredDayAverage:       quote.TwoHundredDayAverage,
		TenDayAverageVolume:        quote.AverageDailyVolume10Day,
		ThreeMonthAverageVolume:    quote.AverageDailyVolume3Month,
		YearHigh:                   quote.FiftyTwoWeekHigh,
		YearLow:                    quote.FiftyTwoWeekLow,
		YearChange:                 quote.FiftyTwoWeekChangePerc / 100, // Convert from percentage
	}, nil
}
