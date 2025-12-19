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
	incomeAnnual    *models.FinancialStatement
	incomeQuarterly *models.FinancialStatement
	balanceAnnual    *models.FinancialStatement
	balanceQuarterly *models.FinancialStatement
	cashFlowAnnual    *models.FinancialStatement
	cashFlowQuarterly *models.FinancialStatement
}

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
	// Get the appropriate keys
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

	// Convert frequency to API prefix
	var prefix string
	switch freq {
	case "annual":
		prefix = "annual"
	case "quarterly":
		prefix = "quarterly"
	case "trailing":
		prefix = "trailing"
	default:
		return nil, fmt.Errorf("invalid frequency: %s", freq)
	}

	// Build type parameter with prefix
	typeParams := make([]string, len(keys))
	for i, key := range keys {
		typeParams[i] = prefix + key
	}

	apiURL := fmt.Sprintf("%s/%s", endpoints.FundamentalsURL, t.symbol)

	params := url.Values{}
	params.Set("symbol", t.symbol)
	params.Set("type", strings.Join(typeParams, ","))

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

	resp, err := t.client.Get(apiURL, params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch financials: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: status %d", resp.StatusCode)
	}

	return t.parseFinancialsResponse(resp.Body, prefix)
}

// parseFinancialsResponse parses the timeseries API response into a FinancialStatement.
func (t *Ticker) parseFinancialsResponse(body, prefix string) (*models.FinancialStatement, error) {
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
	if !ok {
		return nil, fmt.Errorf("invalid response: missing result")
	}

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

			items := make([]models.FinancialItem, 0, len(dataPoints))
			for _, dp := range dataPoints {
				dpMap, ok := dp.(map[string]interface{})
				if !ok {
					continue
				}

				asOfDate, _ := dpMap["asOfDate"].(string)
				currencyCode, _ := dpMap["currencyCode"].(string)
				periodType, _ := dpMap["periodType"].(string)

				reportedValue, _ := dpMap["reportedValue"].(map[string]interface{})
				var rawValue float64
				var fmtValue string
				if reportedValue != nil {
					rawValue, _ = reportedValue["raw"].(float64)
					fmtValue, _ = reportedValue["fmt"].(string)
				}

				// Parse date
				date, err := time.Parse("2006-01-02", asOfDate)
				if err != nil {
					continue
				}

				allDates[date] = true
				if stmt.Currency == "" && currencyCode != "" {
					stmt.Currency = currencyCode
				}

				items = append(items, models.FinancialItem{
					AsOfDate:     date,
					CurrencyCode: currencyCode,
					PeriodType:   periodType,
					Value:        rawValue,
					Formatted:    fmtValue,
				})
			}

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

	return stmt, nil
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
