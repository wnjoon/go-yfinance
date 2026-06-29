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

// financialsCache stores cached financial statements.
type financialsCache struct {
	incomeAnnual      *models.FinancialStatement
	incomeQuarterly   *models.FinancialStatement
	balanceAnnual     *models.FinancialStatement
	balanceQuarterly  *models.FinancialStatement
	cashFlowAnnual    *models.FinancialStatement
	cashFlowQuarterly *models.FinancialStatement
}

const financialsChunkKeys = 60

type financialsPayloadGetter func(apiURL string, params url.Values) (string, error)

// IncomeStatement returns the income statement data.
//
// Parameters:
//   - freq: "annual", "yearly", or "quarterly" (default: "annual")
//
// Note: "yearly" is accepted as an alias for "annual" for Python yfinance compatibility.
//
// The returned [models.FinancialStatement] contains fields like TotalRevenue, GrossProfit,
// OperatingIncome, NetIncome, EBITDA, BasicEPS, and DilutedEPS.
//
// Example:
//
//	income, err := ticker.IncomeStatement("annual")
//	if revenue, ok := income.GetLatest("TotalRevenue"); ok {
//	    fmt.Printf("Revenue: %.2f\n", revenue)
//	}
func (t *Ticker) IncomeStatement(freq string) (*models.FinancialStatement, error) {
	freq = normalizeFrequency(freq)

	// Check cache
	if t.financialsCache != nil {
		if freq == "annual" && t.financialsCache.incomeAnnual != nil {
			return t.financialsCache.incomeAnnual, nil
		}
		if freq == "quarterly" && t.financialsCache.incomeQuarterly != nil {
			return t.financialsCache.incomeQuarterly, nil
		}
	}

	stmt, err := t.fetchFinancials("income", freq)
	if err != nil {
		return nil, err
	}

	// Cache result
	t.initFinancialsCache()
	if freq == "annual" {
		t.financialsCache.incomeAnnual = stmt
	} else {
		t.financialsCache.incomeQuarterly = stmt
	}

	return stmt, nil
}

// BalanceSheet returns the balance sheet data.
//
// Parameters:
//   - freq: "annual", "yearly", or "quarterly" (default: "annual")
//
// Note: "yearly" is accepted as an alias for "annual" for Python yfinance compatibility.
//
// The returned [models.FinancialStatement] contains fields like TotalAssets, TotalLiabilities,
// TotalEquity, CashAndCashEquivalents, TotalDebt, and WorkingCapital.
//
// Example:
//
//	balance, err := ticker.BalanceSheet("quarterly")
//	if assets, ok := balance.GetLatest("TotalAssets"); ok {
//	    fmt.Printf("Total Assets: %.2f\n", assets)
//	}
func (t *Ticker) BalanceSheet(freq string) (*models.FinancialStatement, error) {
	freq = normalizeFrequency(freq)

	// Check cache
	if t.financialsCache != nil {
		if freq == "annual" && t.financialsCache.balanceAnnual != nil {
			return t.financialsCache.balanceAnnual, nil
		}
		if freq == "quarterly" && t.financialsCache.balanceQuarterly != nil {
			return t.financialsCache.balanceQuarterly, nil
		}
	}

	stmt, err := t.fetchFinancials("balance-sheet", freq)
	if err != nil {
		return nil, err
	}

	// Cache result
	t.initFinancialsCache()
	if freq == "annual" {
		t.financialsCache.balanceAnnual = stmt
	} else {
		t.financialsCache.balanceQuarterly = stmt
	}

	return stmt, nil
}

// CashFlow returns the cash flow statement data.
//
// Parameters:
//   - freq: "annual", "yearly", or "quarterly" (default: "annual")
//
// Note: "yearly" is accepted as an alias for "annual" for Python yfinance compatibility.
//
// The returned [models.FinancialStatement] contains fields like OperatingCashFlow,
// InvestingCashFlow, FinancingCashFlow, FreeCashFlow, and CapitalExpenditure.
//
// Example:
//
//	cashFlow, err := ticker.CashFlow("annual")
//	if fcf, ok := cashFlow.GetLatest("FreeCashFlow"); ok {
//	    fmt.Printf("Free Cash Flow: %.2f\n", fcf)
//	}
func (t *Ticker) CashFlow(freq string) (*models.FinancialStatement, error) {
	freq = normalizeFrequency(freq)

	// Check cache
	if t.financialsCache != nil {
		if freq == "annual" && t.financialsCache.cashFlowAnnual != nil {
			return t.financialsCache.cashFlowAnnual, nil
		}
		if freq == "quarterly" && t.financialsCache.cashFlowQuarterly != nil {
			return t.financialsCache.cashFlowQuarterly, nil
		}
	}

	stmt, err := t.fetchFinancials("cash-flow", freq)
	if err != nil {
		return nil, err
	}

	// Cache result
	t.initFinancialsCache()
	if freq == "annual" {
		t.financialsCache.cashFlowAnnual = stmt
	} else {
		t.financialsCache.cashFlowQuarterly = stmt
	}

	return stmt, nil
}

// initFinancialsCache initializes the financials cache if nil.
func (t *Ticker) initFinancialsCache() {
	if t.financialsCache == nil {
		t.financialsCache = &financialsCache{}
	}
}

// normalizeFrequency normalizes frequency parameter to match API expectations.
// Accepts "yearly" as alias for "annual" for Python yfinance compatibility.
func normalizeFrequency(freq string) string {
	switch freq {
	case "yearly":
		return "annual"
	case "":
		return "annual"
	default:
		return freq
	}
}

// fetchFinancials fetches financial data from the timeseries API.
func (t *Ticker) fetchFinancials(statementType, freq string) (*models.FinancialStatement, error) {
	return t.fetchFinancialsWithGetter(statementType, freq, t.fetchFinancialsBody)
}

func (t *Ticker) fetchFinancialsWithGetter(statementType, freq string, getter financialsPayloadGetter) (*models.FinancialStatement, error) {
	keys, prefix, err := financialKeysAndPrefix(statementType, freq)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("%s/%s", endpoints.FundamentalsURL, t.symbol)
	baseParams, err := t.financialsBaseParams()
	if err != nil {
		return nil, err
	}
	return t.fetchFinancialsWithParams(apiURL, baseParams, prefix, keys, getter)
}

func (t *Ticker) fetchFinancialsWithParams(apiURL string, baseParams url.Values, prefix string, keys []string, getter financialsPayloadGetter) (*models.FinancialStatement, error) {
	var result []interface{}
	var err error
	if t.financialsChunkedEnabled() {
		result, err = t.fetchFinancialsChunked(apiURL, baseParams, prefix, keys, getter)
		if err != nil {
			t.setFinancialsChunked(false)
			return nil, err
		}
		return t.parseFinancialsResult(result, prefix), nil
	}

	result, err = t.fetchFinancialsSingle(apiURL, baseParams, prefix, keys, getter)
	if err == nil {
		return t.parseFinancialsResult(result, prefix), nil
	}

	t.setFinancialsChunked(true)
	result, chunkErr := t.fetchFinancialsChunked(apiURL, baseParams, prefix, keys, getter)
	if chunkErr != nil {
		t.setFinancialsChunked(false)
		return nil, chunkErr
	}
	return t.parseFinancialsResult(result, prefix), nil
}

func financialKeysAndPrefix(statementType, freq string) ([]string, string, error) {
	var keys []string
	switch statementType {
	case "income":
		keys = endpoints.IncomeStatementKeys
	case "balance-sheet":
		keys = endpoints.BalanceSheetKeys
	case "cash-flow":
		keys = endpoints.CashFlowKeys
	default:
		return nil, "", fmt.Errorf("invalid statement type: %s", statementType)
	}

	var prefix string
	switch freq {
	case "annual":
		prefix = "annual"
	case "quarterly":
		prefix = "quarterly"
	case "trailing":
		prefix = "trailing"
	default:
		return nil, "", fmt.Errorf("invalid frequency: %s", freq)
	}
	return keys, prefix, nil
}

func (t *Ticker) financialsBaseParams() (url.Values, error) {
	params := url.Values{}
	params.Set("symbol", t.symbol)

	// Set time range (from 2016 to now, same as Python yfinance)
	start := time.Date(2016, 12, 31, 0, 0, 0, 0, time.UTC)
	end := time.Now()
	params.Set("period1", fmt.Sprintf("%d", start.Unix()))
	params.Set("period2", fmt.Sprintf("%d", end.Unix()))

	// Add crumb
	params, err := t.auth.AddCrumbToParams(params)
	if err != nil {
		return nil, fmt.Errorf("failed to add crumb: %w", err)
	}
	return params, nil
}

func (t *Ticker) fetchFinancialsSingle(apiURL string, baseParams url.Values, prefix string, keys []string, getter financialsPayloadGetter) ([]interface{}, error) {
	params := cloneURLValues(baseParams)
	params.Set("type", strings.Join(prefixedFinancialKeys(prefix, keys), ","))

	body, err := getter(apiURL, params)
	if err != nil {
		return nil, err
	}
	return financialsResultItems(body)
}

func (t *Ticker) fetchFinancialsChunked(apiURL string, baseParams url.Values, prefix string, keys []string, getter financialsPayloadGetter) ([]interface{}, error) {
	result := make([]interface{}, 0, len(keys))
	for i := 0; i < len(keys); i += financialsChunkKeys {
		end := i + financialsChunkKeys
		if end > len(keys) {
			end = len(keys)
		}

		params := cloneURLValues(baseParams)
		params.Set("type", strings.Join(prefixedFinancialKeys(prefix, keys[i:end]), ","))
		body, err := getter(apiURL, params)
		if err != nil {
			return nil, err
		}
		chunkResult, err := financialsResultItems(body)
		if err != nil {
			return nil, err
		}
		result = append(result, chunkResult...)
	}
	return result, nil
}

func (t *Ticker) fetchFinancialsBody(apiURL string, params url.Values) (string, error) {
	resp, err := t.client.Get(apiURL, params)
	if err != nil {
		return "", fmt.Errorf("failed to fetch financials: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("API error: status %d", resp.StatusCode)
	}

	return resp.Body, nil
}

func prefixedFinancialKeys(prefix string, keys []string) []string {
	typeParams := make([]string, len(keys))
	for i, key := range keys {
		typeParams[i] = prefix + key
	}
	return typeParams
}

func cloneURLValues(values url.Values) url.Values {
	cloned := make(url.Values, len(values))
	for key, vals := range values {
		cloned[key] = append([]string{}, vals...)
	}
	return cloned
}

func financialsResultItems(body string) ([]interface{}, error) {
	var rawResp map[string]interface{}
	if err := json.Unmarshal([]byte(body), &rawResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	timeseries, ok := rawResp["timeseries"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response: missing timeseries")
	}

	if errObj, ok := timeseries["error"]; ok && errObj != nil {
		return nil, fmt.Errorf("API error in response")
	}

	result, ok := timeseries["result"].([]interface{})
	if !ok || len(result) == 0 {
		return nil, fmt.Errorf("invalid response: missing result")
	}

	return result, nil
}

func (t *Ticker) parseFinancialsResult(result []interface{}, prefix string) *models.FinancialStatement {
	stmt := models.NewFinancialStatement()
	allDates := make(map[time.Time]bool)

	for _, item := range result {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// Find the data field (it's named with the full key like "annualTotalRevenue")
		for key, value := range itemMap {
			if key == "meta" || key == "timestamp" {
				continue
			}

			// Extract field name by removing prefix
			fieldName := strings.TrimPrefix(key, prefix)
			if fieldName == key {
				continue // Prefix not found, skip
			}

			dataPoints, ok := value.([]interface{})
			if !ok {
				continue
			}

			items := parseFinancialItems(dataPoints, stmt, allDates)

			if len(items) > 0 {
				// Sort items by date ascending
				sort.Slice(items, func(i, j int) bool {
					return items[i].AsOfDate.Before(items[j].AsOfDate)
				})
				stmt.Data[fieldName] = items
			}
		}
	}

	// Build sorted dates list
	dates := make([]time.Time, 0, len(allDates))
	for d := range allDates {
		dates = append(dates, d)
	}
	sort.Slice(dates, func(i, j int) bool {
		return dates[i].Before(dates[j])
	})
	stmt.Dates = dates

	return stmt
}

func (t *Ticker) financialsChunkedEnabled() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.financialsChunked
}

func (t *Ticker) setFinancialsChunked(enabled bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.financialsChunked = enabled
}

func parseFinancialItems(dataPoints []interface{}, stmt *models.FinancialStatement, allDates map[time.Time]bool) []models.FinancialItem {
	items := make([]models.FinancialItem, 0, len(dataPoints))
	for _, dp := range dataPoints {
		item, ok := parseFinancialItem(dp)
		if !ok {
			continue
		}
		allDates[item.AsOfDate] = true
		if stmt.Currency == "" && item.CurrencyCode != "" {
			stmt.Currency = item.CurrencyCode
		}
		items = append(items, item)
	}
	return items
}

func parseFinancialItem(dp interface{}) (models.FinancialItem, bool) {
	dpMap, ok := dp.(map[string]interface{})
	if !ok {
		return models.FinancialItem{}, false
	}

	asOfDate, _ := dpMap["asOfDate"].(string)
	date, err := time.Parse("2006-01-02", asOfDate)
	if err != nil {
		return models.FinancialItem{}, false
	}

	currencyCode, _ := dpMap["currencyCode"].(string)
	periodType, _ := dpMap["periodType"].(string)
	rawValue, fmtValue := reportedFinancialValue(dpMap)
	return models.FinancialItem{
		AsOfDate:     date,
		CurrencyCode: currencyCode,
		PeriodType:   periodType,
		Value:        rawValue,
		Formatted:    fmtValue,
	}, true
}

func reportedFinancialValue(dpMap map[string]interface{}) (float64, string) {
	reportedValue, _ := dpMap["reportedValue"].(map[string]interface{})
	if reportedValue == nil {
		return 0, ""
	}
	rawValue, _ := reportedValue["raw"].(float64)
	fmtValue, _ := reportedValue["fmt"].(string)
	return rawValue, fmtValue
}

// FinancialsJSON returns raw JSON for debugging.
func (t *Ticker) FinancialsJSON(statementType, freq string) ([]byte, error) {
	stmt, err := t.fetchFinancialsRaw(statementType, freq)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(stmt, "", "  ")
}

// fetchFinancialsRaw fetches raw financials response.
func (t *Ticker) fetchFinancialsRaw(statementType, freq string) (map[string]interface{}, error) {
	var keys []string
	switch statementType {
	case "income":
		keys = endpoints.IncomeStatementKeys
	case "balance-sheet":
		keys = endpoints.BalanceSheetKeys
	case "cash-flow":
		keys = endpoints.CashFlowKeys
	default:
		return nil, fmt.Errorf("invalid statement type: %s", statementType)
	}

	var prefix string
	switch freq {
	case "annual", "":
		prefix = "annual"
	case "quarterly":
		prefix = "quarterly"
	default:
		return nil, fmt.Errorf("invalid frequency: %s", freq)
	}

	typeParams := make([]string, len(keys))
	for i, key := range keys {
		typeParams[i] = prefix + key
	}

	apiURL := fmt.Sprintf("%s/%s", endpoints.FundamentalsURL, t.symbol)

	params := url.Values{}
	params.Set("symbol", t.symbol)
	params.Set("type", strings.Join(typeParams, ","))

	start := time.Date(2016, 12, 31, 0, 0, 0, 0, time.UTC)
	end := time.Now()
	params.Set("period1", fmt.Sprintf("%d", start.Unix()))
	params.Set("period2", fmt.Sprintf("%d", end.Unix()))

	params, err := t.auth.AddCrumbToParams(params)
	if err != nil {
		return nil, fmt.Errorf("failed to add crumb: %w", err)
	}

	resp, err := t.client.Get(apiURL, params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch financials: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(resp.Body), &result); err != nil {
		return nil, err
	}

	return result, nil
}
