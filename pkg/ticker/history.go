package ticker

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/client"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

// History fetches historical OHLCV data for the ticker.
//
// The method returns a slice of [models.Bar] containing Open, High, Low, Close,
// Adjusted Close, and Volume data for each period.
//
// Parameters can be configured via [models.HistoryParams]:
//   - Period: Time range (1d, 5d, 1mo, 3mo, 6mo, 1y, 2y, 5y, 10y, ytd, max)
//   - Interval: Data granularity (1m, 2m, 5m, 15m, 30m, 60m, 90m, 1h, 1d, 5d, 1wk, 1mo, 3mo)
//   - Start/End: Specific date range (overrides Period)
//   - PrePost: Include pre/post market data
//   - AutoAdjust: Adjust prices for splits/dividends
//   - Actions: Include dividend and split data in bars
//
// Example:
//
//	bars, err := ticker.History(models.HistoryParams{
//	    Period:   "1mo",
//	    Interval: "1d",
//	})
func (t *Ticker) History(params models.HistoryParams) ([]models.Bar, error) {
	params = normalizeHistoryParams(params)

	result, err := t.fetchChartResult(params)
	if err != nil {
		return nil, err
	}

	// Parse OHLCV data
	bars, err := t.parseChartData(result, params.AutoAdjust, params.Actions)
	if err != nil {
		return nil, err
	}

	// Remove NaN rows unless KeepNA is true
	if !params.KeepNA {
		bars = filterValidBars(bars)
	}

	return bars, nil
}

func normalizeHistoryParams(params models.HistoryParams) models.HistoryParams {
	if params.Period == "" && params.Start == nil && params.End == nil {
		params.Period = "1mo"
	}
	if params.Interval == "" {
		params.Interval = "1d"
	}

	return params
}

func buildHistoryURLParams(params models.HistoryParams) url.Values {
	urlParams := url.Values{}

	if params.Period != "" && params.Start == nil && params.End == nil {
		// Use range parameter
		urlParams.Set("range", params.Period)
	} else {
		// Use period1/period2 parameters
		var start, end int64

		if params.Start != nil {
			start = params.Start.Unix()
		} else {
			// Default: 1 month ago
			start = time.Now().AddDate(0, -1, 0).Unix()
		}

		if params.End != nil {
			end = params.End.Unix()
		} else {
			end = time.Now().Unix()
		}

		urlParams.Set("period1", fmt.Sprintf("%d", start))
		urlParams.Set("period2", fmt.Sprintf("%d", end))
	}

	urlParams.Set("interval", params.Interval)
	urlParams.Set("includePrePost", fmt.Sprintf("%t", params.PrePost))
	urlParams.Set("events", "div,splits,capitalGains")

	return urlParams
}

func (t *Ticker) fetchChartResult(params models.HistoryParams) (*models.ChartResult, error) {
	params = normalizeHistoryParams(params)
	urlParams := buildHistoryURLParams(params)

	resp, err := t.getWithCrumb(t.chartURL(), urlParams)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch history: %w", err)
	}

	var chartResp models.ChartResponse
	if err := json.Unmarshal([]byte(resp.Body), &chartResp); err != nil {
		return nil, client.WrapInvalidResponseError(err)
	}

	if chartResp.Chart.Error != nil {
		return nil, fmt.Errorf("API error: %s", chartResp.Chart.Error.Description)
	}

	if len(chartResp.Chart.Result) == 0 {
		return nil, client.WrapNotFoundError(t.symbol)
	}

	result := chartResp.Chart.Result[0]

	// Cache metadata
	t.setHistoryMetadata(&result.Meta)

	return &result, nil
}

// parseChartData converts chart API response to Bar slice.
func (t *Ticker) parseChartData(result *models.ChartResult, autoAdjust bool, includeActions bool) ([]models.Bar, error) {
	if len(result.Timestamp) == 0 {
		return []models.Bar{}, nil
	}

	if len(result.Indicators.Quote) == 0 {
		return nil, fmt.Errorf("no quote data in response")
	}

	quote := result.Indicators.Quote[0]
	timestamps := result.Timestamp

	// Get adjusted close if available
	var adjClose []*float64
	if len(result.Indicators.AdjClose) > 0 {
		adjClose = result.Indicators.AdjClose[0].AdjClose
	}

	// Parse dividends, splits, and capital gains
	dividends := make(map[int64]float64)
	dividendCurrencies := make(map[int64]string)
	splits := make(map[int64]float64)
	capitalGains := make(map[int64]float64)

	if includeActions && result.Events != nil {
		if result.Events.Dividends != nil {
			for _, div := range result.Events.Dividends {
				dividends[div.Date] = div.Amount
				if div.Currency != "" {
					dividendCurrencies[div.Date] = div.Currency
				}
			}
		}
		if result.Events.Splits != nil {
			for _, split := range result.Events.Splits {
				if split.Denominator != 0 {
					splits[split.Date] = split.Numerator / split.Denominator
				}
			}
		}
		if result.Events.CapitalGains != nil {
			for _, cg := range result.Events.CapitalGains {
				capitalGains[cg.Date] = cg.Amount
			}
		}
	}

	bars := make([]models.Bar, 0, len(timestamps))

	for i, ts := range timestamps {
		bar := models.Bar{
			Date: time.Unix(ts, 0).UTC(),
		}

		// Handle nil values (gaps in data)
		if i < len(quote.Open) && quote.Open[i] != nil {
			bar.Open = *quote.Open[i]
		}
		if i < len(quote.High) && quote.High[i] != nil {
			bar.High = *quote.High[i]
		}
		if i < len(quote.Low) && quote.Low[i] != nil {
			bar.Low = *quote.Low[i]
		}
		if i < len(quote.Close) && quote.Close[i] != nil {
			bar.Close = *quote.Close[i]
		}
		if i < len(quote.Volume) && quote.Volume[i] != nil {
			bar.Volume = *quote.Volume[i]
		}

		// Adjusted close
		if adjClose != nil && i < len(adjClose) && adjClose[i] != nil {
			bar.AdjClose = *adjClose[i]
		} else {
			bar.AdjClose = bar.Close
		}

		// Add actions if available
		if div, ok := dividends[ts]; ok {
			bar.Dividends = div
		}
		if currency, ok := dividendCurrencies[ts]; ok {
			bar.DividendCurrency = currency
		}
		if split, ok := splits[ts]; ok {
			bar.Splits = split
		}
		if cg, ok := capitalGains[ts]; ok {
			bar.CapitalGains = cg
		}

		// Auto-adjust OHLC if requested
		if autoAdjust && bar.Close != 0 && bar.AdjClose != 0 {
			ratio := bar.AdjClose / bar.Close
			bar.Open *= ratio
			bar.High *= ratio
			bar.Low *= ratio
			bar.Close = bar.AdjClose
		}

		bars = append(bars, bar)
	}

	return bars, nil
}

// filterValidBars removes bars with zero/NaN values.
func filterValidBars(bars []models.Bar) []models.Bar {
	result := make([]models.Bar, 0, len(bars))
	for _, bar := range bars {
		// Skip bars where all OHLC values are zero
		if bar.Open == 0 && bar.High == 0 && bar.Low == 0 && bar.Close == 0 {
			continue
		}
		result = append(result, bar)
	}
	return result
}

// HistoryPeriod is a convenience method to fetch history with just a period string.
//
// Valid periods: 1d, 5d, 1mo, 3mo, 6mo, 1y, 2y, 5y, 10y, ytd, max
//
// Uses daily interval with auto-adjustment enabled.
func (t *Ticker) HistoryPeriod(period string) ([]models.Bar, error) {
	return t.History(models.HistoryParams{Period: period, Interval: "1d", AutoAdjust: true})
}

// HistoryRange fetches history for a specific date range.
//
// Parameters:
//   - start: Start date (inclusive)
//   - end: End date (exclusive)
//   - interval: Data granularity (1m, 2m, 5m, 15m, 30m, 60m, 90m, 1h, 1d, 5d, 1wk, 1mo, 3mo)
//
// Uses auto-adjustment enabled by default.
func (t *Ticker) HistoryRange(start, end time.Time, interval string) ([]models.Bar, error) {
	return t.History(models.HistoryParams{
		Start:      &start,
		End:        &end,
		Interval:   interval,
		AutoAdjust: true,
	})
}

// Dividends returns the dividend history for the ticker.
//
// Returns all historical dividend payments with dates and amounts.
func (t *Ticker) Dividends() ([]models.Dividend, error) {
	result, err := t.fetchChartResult(models.HistoryParams{
		Period:   "max",
		Interval: "1d",
		Actions:  true,
	})
	if err != nil {
		return nil, err
	}

	return parseDividendEvents(result), nil
}

// Splits returns the stock split history for the ticker.
//
// Returns all historical stock splits with dates and ratios.
func (t *Ticker) Splits() ([]models.Split, error) {
	result, err := t.fetchChartResult(models.HistoryParams{
		Period:   "max",
		Interval: "1d",
		Actions:  true,
	})
	if err != nil {
		return nil, err
	}

	return parseSplitEvents(result), nil
}

// CapitalGains returns capital gain distributions for the ticker.
func (t *Ticker) CapitalGains() ([]models.CapitalGain, error) {
	result, err := t.fetchChartResult(models.HistoryParams{
		Period:   "max",
		Interval: "1d",
		Actions:  true,
	})
	if err != nil {
		return nil, err
	}

	return parseCapitalGainEvents(result), nil
}

// Actions returns dividends, splits, and capital gains for the ticker.
//
// This is a convenience method that combines the action event series into a
// single response.
func (t *Ticker) Actions() (*models.Actions, error) {
	result, err := t.fetchChartResult(models.HistoryParams{
		Period:   "max",
		Interval: "1d",
		Actions:  true,
	})
	if err != nil {
		return nil, err
	}

	return &models.Actions{
		Dividends:    parseDividendEvents(result),
		Splits:       parseSplitEvents(result),
		CapitalGains: parseCapitalGainEvents(result),
	}, nil
}

func parseDividendEvents(result *models.ChartResult) []models.Dividend {
	if result == nil || result.Events == nil || result.Events.Dividends == nil {
		return nil
	}

	dividends := make([]models.Dividend, 0, len(result.Events.Dividends))
	for _, div := range result.Events.Dividends {
		if div.Amount <= 0 {
			continue
		}
		dividends = append(dividends, models.Dividend{
			Date:     time.Unix(div.Date, 0).UTC(),
			Amount:   div.Amount,
			Currency: div.Currency,
		})
	}

	sort.Slice(dividends, func(i, j int) bool {
		return dividends[i].Date.Before(dividends[j].Date)
	})

	return dividends
}

func parseSplitEvents(result *models.ChartResult) []models.Split {
	if result == nil || result.Events == nil || result.Events.Splits == nil {
		return nil
	}

	splits := make([]models.Split, 0, len(result.Events.Splits))
	for _, event := range result.Events.Splits {
		if event.Numerator == 0 || event.Denominator == 0 {
			continue
		}
		split := models.Split{
			Date:        time.Unix(event.Date, 0).UTC(),
			Numerator:   event.Numerator,
			Denominator: event.Denominator,
			Ratio:       event.SplitRatio,
		}
		if split.Ratio == "" {
			split.Ratio = fmt.Sprintf("%.0f:%.0f", event.Numerator, event.Denominator)
		}
		splits = append(splits, split)
	}

	sort.Slice(splits, func(i, j int) bool {
		return splits[i].Date.Before(splits[j].Date)
	})

	return splits
}

func parseCapitalGainEvents(result *models.ChartResult) []models.CapitalGain {
	if result == nil || result.Events == nil || result.Events.CapitalGains == nil {
		return nil
	}

	capitalGains := make([]models.CapitalGain, 0, len(result.Events.CapitalGains))
	for _, event := range result.Events.CapitalGains {
		if event.Amount <= 0 {
			continue
		}
		capitalGains = append(capitalGains, models.CapitalGain{
			Date:   time.Unix(event.Date, 0).UTC(),
			Amount: event.Amount,
		})
	}

	sort.Slice(capitalGains, func(i, j int) bool {
		return capitalGains[i].Date.Before(capitalGains[j].Date)
	})

	return capitalGains
}
