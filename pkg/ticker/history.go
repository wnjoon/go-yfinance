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
func (t *Ticker) History(params models.HistoryParams) ([]models.Bar, error) {
	// Apply defaults
	if params.Period == "" && params.Start == nil && params.End == nil {
		params.Period = "1mo"
	}
	if params.Interval == "" {
		params.Interval = "1d"
	}

	// Build URL parameters
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

	// Parse OHLCV data
	bars, err := t.parseChartData(&result, params.AutoAdjust, params.Actions)
	if err != nil {
		return nil, err
	}

	// Remove NaN rows unless KeepNA is true
	if !params.KeepNA {
		bars = filterValidBars(bars)
	}

	return bars, nil
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

	// Parse dividends and splits
	dividends := make(map[int64]float64)
	splits := make(map[int64]float64)

	if includeActions && result.Events != nil {
		if result.Events.Dividends != nil {
			for _, div := range result.Events.Dividends {
				dividends[div.Date] = div.Amount
			}
		}
		if result.Events.Splits != nil {
			for _, split := range result.Events.Splits {
				splits[split.Date] = split.Numerator / split.Denominator
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
		if split, ok := splits[ts]; ok {
			bar.Splits = split
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
func (t *Ticker) HistoryPeriod(period string) ([]models.Bar, error) {
	return t.History(models.HistoryParams{Period: period, Interval: "1d", AutoAdjust: true})
}

// HistoryRange fetches history for a specific date range.
func (t *Ticker) HistoryRange(start, end time.Time, interval string) ([]models.Bar, error) {
	return t.History(models.HistoryParams{
		Start:      &start,
		End:        &end,
		Interval:   interval,
		AutoAdjust: true,
	})
}

// Dividends returns the dividend history for the ticker.
func (t *Ticker) Dividends() ([]models.Dividend, error) {
	bars, err := t.History(models.HistoryParams{
		Period:   "max",
		Interval: "1d",
		Actions:  true,
	})
	if err != nil {
		return nil, err
	}

	var dividends []models.Dividend
	for _, bar := range bars {
		if bar.Dividends > 0 {
			dividends = append(dividends, models.Dividend{
				Date:   bar.Date,
				Amount: bar.Dividends,
			})
		}
	}

	return dividends, nil
}

// Splits returns the stock split history for the ticker.
func (t *Ticker) Splits() ([]models.Split, error) {
	bars, err := t.History(models.HistoryParams{
		Period:   "max",
		Interval: "1d",
		Actions:  true,
	})
	if err != nil {
		return nil, err
	}

	var splits []models.Split
	for _, bar := range bars {
		if bar.Splits != 0 && bar.Splits != 1 {
			split := models.Split{
				Date:        bar.Date,
				Numerator:   bar.Splits,
				Denominator: 1,
			}
			if bar.Splits > 1 {
				split.Ratio = fmt.Sprintf("%.0f:1", bar.Splits)
			} else {
				split.Ratio = fmt.Sprintf("1:%.0f", 1/bar.Splits)
			}
			splits = append(splits, split)
		}
	}

	return splits, nil
}

// Actions returns both dividends and splits for the ticker.
func (t *Ticker) Actions() (*models.Actions, error) {
	bars, err := t.History(models.HistoryParams{
		Period:   "max",
		Interval: "1d",
		Actions:  true,
	})
	if err != nil {
		return nil, err
	}

	actions := &models.Actions{}

	for _, bar := range bars {
		if bar.Dividends > 0 {
			actions.Dividends = append(actions.Dividends, models.Dividend{
				Date:   bar.Date,
				Amount: bar.Dividends,
			})
		}
		if bar.Splits != 0 && bar.Splits != 1 {
			split := models.Split{
				Date:        bar.Date,
				Numerator:   bar.Splits,
				Denominator: 1,
			}
			if bar.Splits > 1 {
				split.Ratio = fmt.Sprintf("%.0f:1", bar.Splits)
			} else {
				split.Ratio = fmt.Sprintf("1:%.0f", 1/bar.Splits)
			}
			actions.Splits = append(actions.Splits, split)
		}
	}

	// Sort by date
	sort.Slice(actions.Dividends, func(i, j int) bool {
		return actions.Dividends[i].Date.Before(actions.Dividends[j].Date)
	})
	sort.Slice(actions.Splits, func(i, j int) bool {
		return actions.Splits[i].Date.Before(actions.Splits[j].Date)
	})

	return actions, nil
}
