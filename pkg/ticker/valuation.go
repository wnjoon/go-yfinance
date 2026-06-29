package ticker

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/wnjoon/go-yfinance/internal/endpoints"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

var valuationMeasureLabels = map[string]string{
	"MarketCap":                    "Market Cap",
	"EnterpriseValue":              "Enterprise Value",
	"PeRatio":                      "Trailing P/E",
	"ForwardPeRatio":               "Forward P/E",
	"PegRatio":                     "PEG Ratio (5yr expected)",
	"PsRatio":                      "Price/Sales",
	"PbRatio":                      "Price/Book",
	"EnterprisesValueRevenueRatio": "Enterprise Value/Revenue",
	"EnterprisesValueEBITDARatio":  "Enterprise Value/EBITDA",
}

var valuationMeasureKeys = []string{
	"MarketCap",
	"EnterpriseValue",
	"PeRatio",
	"ForwardPeRatio",
	"PegRatio",
	"PsRatio",
	"PbRatio",
	"EnterprisesValueRevenueRatio",
	"EnterprisesValueEBITDARatio",
}

var valuationMeasureOrder = []string{
	"Market Cap",
	"Enterprise Value",
	"Trailing P/E",
	"Forward P/E",
	"PEG Ratio (5yr expected)",
	"Price/Sales",
	"Price/Book",
	"Enterprise Value/Revenue",
	"Enterprise Value/EBITDA",
}

var valuationFreqPrefix = map[string]string{
	"quarterly": "quarterly",
	"monthly":   "monthly",
	"yearly":    "annual",
	"trailing":  "trailing",
}

// Valuation returns the valuation measures table using Python yfinance's
// default settings: quarterly period columns capped to the newest five periods.
func (t *Ticker) Valuation() (*models.ValuationMeasures, error) {
	periods := 5
	return t.GetValuationMeasures("quarterly", &periods)
}

// ValuationMeasures is an alias for Valuation.
func (t *Ticker) ValuationMeasures() (*models.ValuationMeasures, error) {
	return t.Valuation()
}

// GetValuationMeasures returns valuation measures from Yahoo's
// fundamentals-timeseries API.
//
// freq must be one of "quarterly", "monthly", "yearly", or "trailing".
// periods caps the number of period date columns returned, newest first.
// A nil periods pointer returns every available period column. Use 0 for
// the "Current" column only.
func (t *Ticker) GetValuationMeasures(freq string, periods *int) (*models.ValuationMeasures, error) {
	if periods != nil && *periods < 0 {
		return nil, fmt.Errorf("periods must be >= 0 or nil")
	}
	prefix, err := valuationPrefix(freq)
	if err != nil {
		return nil, err
	}

	t.mu.RLock()
	if t.valuationCache != nil && t.valuationCache[freq] != nil {
		cached := t.valuationCache[freq]
		t.mu.RUnlock()
		return sliceValuationPeriods(cached, periods), nil
	}
	t.mu.RUnlock()

	measures, err := t.fetchValuationMeasures(freq, prefix)
	if err != nil {
		return nil, err
	}

	t.mu.Lock()
	if t.valuationCache == nil {
		t.valuationCache = make(map[string]*models.ValuationMeasures)
	}
	t.valuationCache[freq] = measures
	t.mu.Unlock()

	return sliceValuationPeriods(measures, periods), nil
}

func (t *Ticker) fetchValuationMeasures(freq, prefix string) (*models.ValuationMeasures, error) {
	prefixes := []string{prefix}
	if prefix != "trailing" {
		prefixes = append(prefixes, "trailing")
		sort.Strings(prefixes)
	}

	types := make([]string, 0, len(valuationMeasureKeys)*len(prefixes))
	for _, key := range valuationMeasureKeys {
		for _, p := range prefixes {
			types = append(types, p+key)
		}
	}

	params := url.Values{}
	params.Set("symbol", t.symbol)
	params.Set("type", strings.Join(types, ","))
	params.Set("period1", fmt.Sprintf("%d", time.Date(2016, 12, 31, 0, 0, 0, 0, time.UTC).Unix()))
	params.Set("period2", fmt.Sprintf("%d", time.Now().UTC().Truncate(24*time.Hour).Add(24*time.Hour).Unix()))

	apiURL := fmt.Sprintf("%s/%s", endpoints.FundamentalsURL, url.PathEscape(t.symbol))
	resp, err := t.getWithCrumb(apiURL, params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch valuation measures: %w", err)
	}
	return parseValuationMeasuresTimeseries(resp.Body, prefix)
}

func valuationPrefix(freq string) (string, error) {
	prefix, ok := valuationFreqPrefix[freq]
	if !ok {
		return "", fmt.Errorf("freq must be one of quarterly, monthly, yearly, trailing, not %q", freq)
	}
	return prefix, nil
}

func parseValuationMeasuresTimeseries(body, prefix string) (*models.ValuationMeasures, error) {
	var rawResp map[string]interface{}
	if err := json.Unmarshal([]byte(body), &rawResp); err != nil {
		return nil, fmt.Errorf("failed to parse valuation measures response: %w", err)
	}

	timeseries, ok := rawResp["timeseries"].(map[string]interface{})
	if !ok {
		return &models.ValuationMeasures{}, nil
	}
	result, _ := timeseries["result"].([]interface{})
	if len(result) == 0 {
		return &models.ValuationMeasures{}, nil
	}

	period := make(map[string]map[time.Time]float64)
	trailing := make(map[string]map[time.Time]float64)

	for _, item := range result {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		for typeName, value := range itemMap {
			if typeName == "meta" || typeName == "timestamp" {
				continue
			}
			points, ok := value.([]interface{})
			if !ok {
				continue
			}

			base, target, ok := valuationTargetSeries(typeName, prefix, period, trailing)
			if !ok {
				continue
			}
			label, ok := valuationMeasureLabels[base]
			if !ok {
				continue
			}
			parseValuationPoints(points, label, target)
		}
	}

	if prefix == "trailing" {
		period = trailing
	}
	if len(period) == 0 && len(trailing) == 0 {
		return &models.ValuationMeasures{}, nil
	}

	return buildValuationMeasures(period, trailing), nil
}

func valuationTargetSeries(typeName, prefix string, period, trailing map[string]map[time.Time]float64) (string, map[string]map[time.Time]float64, bool) {
	if prefix != "trailing" && strings.HasPrefix(typeName, prefix) {
		return strings.TrimPrefix(typeName, prefix), period, true
	}
	if strings.HasPrefix(typeName, "trailing") {
		return strings.TrimPrefix(typeName, "trailing"), trailing, true
	}
	return "", nil, false
}

func parseValuationPoints(points []interface{}, label string, target map[string]map[time.Time]float64) {
	for _, point := range points {
		pointMap, ok := point.(map[string]interface{})
		if !ok {
			continue
		}
		asOf, _ := pointMap["asOfDate"].(string)
		if asOf == "" {
			continue
		}
		date, err := time.Parse("2006-01-02", asOf)
		if err != nil {
			continue
		}
		reported, _ := pointMap["reportedValue"].(map[string]interface{})
		rawValue, ok := reported["raw"].(float64)
		if !ok {
			continue
		}
		if target[label] == nil {
			target[label] = make(map[time.Time]float64)
		}
		target[label][date] = rawValue
	}
}

func buildValuationMeasures(period, trailing map[string]map[time.Time]float64) *models.ValuationMeasures {
	current := make(map[string]float64)
	for label, series := range trailing {
		if len(series) == 0 {
			continue
		}
		latest := latestDate(series)
		current[label] = series[latest]
	}

	dates := valuationDates(period)
	columns := make([]string, 0, len(dates)+1)
	columns = append(columns, "Current")
	for _, date := range dates {
		columns = append(columns, formatValuationDate(date))
	}

	rows := make([]models.ValuationMeasureRow, 0, len(valuationMeasureOrder))
	for _, label := range valuationMeasureOrder {
		row := models.ValuationMeasureRow{Name: label}
		if value, ok := current[label]; ok {
			row.SetValue("Current", floatPtr(value))
		} else {
			row.SetValue("Current", nil)
		}
		for _, date := range dates {
			column := formatValuationDate(date)
			if value, ok := period[label][date]; ok {
				row.SetValue(column, floatPtr(value))
			} else {
				row.SetValue(column, nil)
			}
		}
		rows = append(rows, row)
	}

	return &models.ValuationMeasures{Columns: columns, Rows: rows}
}

func valuationDates(period map[string]map[time.Time]float64) []time.Time {
	seen := make(map[time.Time]bool)
	for _, series := range period {
		for date := range series {
			seen[date] = true
		}
	}
	dates := make([]time.Time, 0, len(seen))
	for date := range seen {
		dates = append(dates, date)
	}
	sort.Slice(dates, func(i, j int) bool {
		return dates[i].After(dates[j])
	})
	return dates
}

func latestDate(series map[time.Time]float64) time.Time {
	var latest time.Time
	for date := range series {
		if latest.IsZero() || date.After(latest) {
			latest = date
		}
	}
	return latest
}

func formatValuationDate(date time.Time) string {
	return fmt.Sprintf("%d/%d/%d", int(date.Month()), date.Day(), date.Year())
}

func sliceValuationPeriods(measures *models.ValuationMeasures, periods *int) *models.ValuationMeasures {
	if measures == nil {
		return nil
	}
	if measures.Empty() || periods == nil {
		return cloneValuationMeasures(measures, measures.Columns)
	}

	limit := *periods
	columns := []string{"Current"}
	dateCols := make([]string, 0, len(measures.Columns))
	for _, column := range measures.Columns {
		if column != "Current" {
			dateCols = append(dateCols, column)
		}
	}
	if limit > len(dateCols) {
		limit = len(dateCols)
	}
	columns = append(columns, dateCols[:limit]...)
	return cloneValuationMeasures(measures, columns)
}

func cloneValuationMeasures(measures *models.ValuationMeasures, columns []string) *models.ValuationMeasures {
	clone := &models.ValuationMeasures{
		Columns: append([]string{}, columns...),
		Rows:    make([]models.ValuationMeasureRow, 0, len(measures.Rows)),
	}
	for _, row := range measures.Rows {
		clonedRow := models.ValuationMeasureRow{Name: row.Name}
		for _, column := range columns {
			if row.RawValues != nil {
				if value, ok := row.RawValues[column]; ok && value != nil {
					clonedRow.SetValue(column, floatPtr(*value))
					continue
				}
			}
			clonedRow.SetValue(column, nil)
		}
		clone.Rows = append(clone.Rows, clonedRow)
	}
	return clone
}

func floatPtr(value float64) *float64 {
	v := value
	return &v
}
