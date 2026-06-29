package ticker

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func TestFinancialStatementMethods(t *testing.T) {
	// Test FinancialStatement helper methods
	stmt := models.NewFinancialStatement()

	// Add test data
	date1 := time.Date(2023, 9, 30, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2024, 9, 30, 0, 0, 0, 0, time.UTC)

	stmt.Data["TotalRevenue"] = []models.FinancialItem{
		{AsOfDate: date1, Value: 383285000000, CurrencyCode: "USD", PeriodType: "12M"},
		{AsOfDate: date2, Value: 391035000000, CurrencyCode: "USD", PeriodType: "12M"},
	}
	stmt.Data["NetIncome"] = []models.FinancialItem{
		{AsOfDate: date1, Value: 96995000000, CurrencyCode: "USD", PeriodType: "12M"},
		{AsOfDate: date2, Value: 93736000000, CurrencyCode: "USD", PeriodType: "12M"},
	}
	stmt.Dates = []time.Time{date1, date2}
	stmt.Currency = "USD"

	// Test Get
	val, ok := stmt.Get("TotalRevenue", date2)
	if !ok {
		t.Error("Expected to find TotalRevenue for date2")
	}
	if val != 391035000000 {
		t.Errorf("Expected 391035000000, got %f", val)
	}

	// Test Get with non-existent date
	_, ok = stmt.Get("TotalRevenue", time.Date(2022, 9, 30, 0, 0, 0, 0, time.UTC))
	if ok {
		t.Error("Expected not to find TotalRevenue for 2022")
	}

	// Test Get with non-existent field
	_, ok = stmt.Get("NonExistent", date1)
	if ok {
		t.Error("Expected not to find NonExistent field")
	}

	// Test GetLatest
	latest, ok := stmt.GetLatest("TotalRevenue")
	if !ok {
		t.Error("Expected to get latest TotalRevenue")
	}
	if latest != 391035000000 {
		t.Errorf("Expected latest 391035000000, got %f", latest)
	}

	// Test GetLatest with non-existent field
	_, ok = stmt.GetLatest("NonExistent")
	if ok {
		t.Error("Expected not to find NonExistent field")
	}

	// Test Fields
	fields := stmt.Fields()
	if len(fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(fields))
	}
}

func TestFinancialsCacheInitialization(t *testing.T) {
	tkr, err := New("AAPL")
	if err != nil {
		t.Fatalf("Failed to create ticker: %v", err)
	}
	defer tkr.Close()

	if tkr.financialsCache != nil {
		t.Error("Expected financialsCache to be nil initially")
	}
}

func TestClearCacheIncludesFinancials(t *testing.T) {
	tkr, err := New("AAPL")
	if err != nil {
		t.Fatalf("Failed to create ticker: %v", err)
	}
	defer tkr.Close()

	// Manually set financialsCache
	tkr.financialsCache = &financialsCache{
		incomeAnnual: models.NewFinancialStatement(),
	}

	// Clear cache
	tkr.ClearCache()

	if tkr.financialsCache != nil {
		t.Error("Expected financialsCache to be nil after ClearCache")
	}
}

func TestFinancialItemDates(t *testing.T) {
	// Test date parsing consistency
	dateStr := "2024-09-30"
	expectedDate := time.Date(2024, 9, 30, 0, 0, 0, 0, time.UTC)

	parsed, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		t.Fatalf("Failed to parse date: %v", err)
	}

	if !parsed.Equal(expectedDate) {
		t.Errorf("Expected %v, got %v", expectedDate, parsed)
	}
}

func TestFrequencyValues(t *testing.T) {
	// Test frequency constants
	if models.FrequencyAnnual != "annual" {
		t.Error("FrequencyAnnual should be 'annual'")
	}
	if models.FrequencyQuarterly != "quarterly" {
		t.Error("FrequencyQuarterly should be 'quarterly'")
	}
	if models.FrequencyTrailing != "trailing" {
		t.Error("FrequencyTrailing should be 'trailing'")
	}
}

func TestNormalizeFrequency(t *testing.T) {
	// Test normalizeFrequency for Python yfinance compatibility
	tests := []struct {
		input    string
		expected string
	}{
		{"yearly", "annual"},
		{"annual", "annual"},
		{"quarterly", "quarterly"},
		{"trailing", "trailing"},
		{"", "annual"},
	}

	for _, tc := range tests {
		result := normalizeFrequency(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeFrequency(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestFinancialsChunkedFallbackOnSingleFailure(t *testing.T) {
	tkr, err := New("MSFT")
	if err != nil {
		t.Fatalf("Failed to create ticker: %v", err)
	}

	callCount := 0
	getter := func(_ string, params url.Values) (string, error) {
		callCount++
		if callCount == 1 {
			return "", fmt.Errorf("timeout")
		}
		return financialsPayloadForTypes(params.Get("type")), nil
	}

	stmt, err := tkr.fetchFinancialsWithParams("https://example.test/timeseries/MSFT", financialsTestParams(), "annual", []string{"TotalAssets", "TotalDebt"}, getter)
	if err != nil {
		t.Fatalf("Expected chunk fallback to recover, got %v", err)
	}
	if callCount != 2 {
		t.Fatalf("Expected single request then chunk fallback, got %d calls", callCount)
	}
	if !tkr.financialsChunkedEnabled() {
		t.Fatal("Expected chunked mode to remain sticky after fallback success")
	}
	if value, ok := stmt.GetLatest("TotalAssets"); !ok || value != 1000 {
		t.Fatalf("Expected TotalAssets from fallback payload, got %v (ok=%v)", value, ok)
	}
}

func TestFinancialsChunkedFallbackOnInvalidSingleResponse(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "missing result", body: `{"timeseries":{}}`},
		{name: "empty result", body: `{"timeseries":{"result":[]}}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tkr, err := New("MSFT")
			if err != nil {
				t.Fatalf("Failed to create ticker: %v", err)
			}

			callCount := 0
			getter := func(_ string, params url.Values) (string, error) {
				callCount++
				if callCount == 1 {
					return tc.body, nil
				}
				return financialsPayloadForTypes(params.Get("type")), nil
			}

			stmt, err := tkr.fetchFinancialsWithParams("https://example.test/timeseries/MSFT", financialsTestParams(), "annual", []string{"TotalAssets", "TotalDebt"}, getter)
			if err != nil {
				t.Fatalf("Expected chunk fallback to recover from invalid response, got %v", err)
			}
			if callCount != 2 {
				t.Fatalf("Expected single request then chunk fallback, got %d calls", callCount)
			}
			if !tkr.financialsChunkedEnabled() {
				t.Fatal("Expected chunked mode to remain sticky after fallback success")
			}
			if value, ok := stmt.GetLatest("TotalDebt"); !ok || value != 1000 {
				t.Fatalf("Expected TotalDebt from fallback payload, got %v (ok=%v)", value, ok)
			}
		})
	}
}

func TestFinancialsChunkedStickySkipsSingleRequest(t *testing.T) {
	tkr, err := New("MSFT")
	if err != nil {
		t.Fatalf("Failed to create ticker: %v", err)
	}
	tkr.setFinancialsChunked(true)

	callCount := 0
	getter := func(_ string, params url.Values) (string, error) {
		callCount++
		return financialsPayloadForTypes(params.Get("type")), nil
	}

	_, err = tkr.fetchFinancialsWithParams("https://example.test/timeseries/MSFT", financialsTestParams(), "annual", []string{"TotalAssets", "TotalDebt"}, getter)
	if err != nil {
		t.Fatalf("Expected sticky chunked request to succeed, got %v", err)
	}
	if callCount != 1 {
		t.Fatalf("Expected sticky chunked mode to skip fast path, got %d calls", callCount)
	}
	if !tkr.financialsChunkedEnabled() {
		t.Fatal("Expected chunked mode to stay enabled")
	}
}

func TestFinancialsChunkedFallbackRollbackOnChunkFailure(t *testing.T) {
	tkr, err := New("MSFT")
	if err != nil {
		t.Fatalf("Failed to create ticker: %v", err)
	}

	getter := func(_ string, _ url.Values) (string, error) {
		return "", fmt.Errorf("timeout")
	}

	_, err = tkr.fetchFinancialsWithParams("https://example.test/timeseries/MSFT", financialsTestParams(), "annual", []string{"TotalAssets", "TotalDebt"}, getter)
	if err == nil {
		t.Fatal("Expected error when chunk fallback also fails")
	}
	if tkr.financialsChunkedEnabled() {
		t.Fatal("Expected chunked mode to roll back after chunk failure")
	}
}

func TestFinancialsChunkSize(t *testing.T) {
	tkr, err := New("MSFT")
	if err != nil {
		t.Fatalf("Failed to create ticker: %v", err)
	}

	keys := make([]string, 125)
	for i := range keys {
		keys[i] = fmt.Sprintf("Key%d", i)
	}

	var chunkSizes []int
	getter := func(_ string, params url.Values) (string, error) {
		chunkSizes = append(chunkSizes, len(strings.Split(params.Get("type"), ",")))
		return financialsPayloadForTypes(params.Get("type")), nil
	}

	_, err = tkr.fetchFinancialsChunked("https://example.test/timeseries/MSFT", financialsTestParams(), "annual", keys, getter)
	if err != nil {
		t.Fatalf("Expected chunked fetch to succeed, got %v", err)
	}
	expected := []int{60, 60, 5}
	if fmt.Sprint(chunkSizes) != fmt.Sprint(expected) {
		t.Fatalf("Expected chunk sizes %v, got %v", expected, chunkSizes)
	}
}

func financialsTestParams() url.Values {
	params := url.Values{}
	params.Set("symbol", "MSFT")
	params.Set("period1", "1719792000")
	params.Set("period2", "1751328000")
	return params
}

func financialsPayloadForTypes(typeParam string) string {
	types := strings.Split(typeParam, ",")
	parts := make([]string, 0, len(types))
	for _, typeName := range types {
		if typeName == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf(`{
			"meta":{"symbol":["MSFT"],"type":["%[1]s"]},
			"timestamp":[1751241600],
			"%[1]s":[{"asOfDate":"2025-06-30","periodType":"12M","currencyCode":"USD","reportedValue":{"raw":1000,"fmt":"1,000"}}]
		}`, typeName))
	}
	return fmt.Sprintf(`{"timeseries":{"result":[%s]}}`, strings.Join(parts, ","))
}

// Integration test - commented out for CI, run manually
// func TestFinancialsLive(t *testing.T) {
// 	tkr, err := New("AAPL")
// 	if err != nil {
// 		t.Fatalf("Failed to create ticker: %v", err)
// 	}
// 	defer tkr.Close()
//
// 	// Test Income Statement Annual
// 	income, err := tkr.IncomeStatement("annual")
// 	if err != nil {
// 		t.Fatalf("Failed to get income statement: %v", err)
// 	}
//
// 	if income == nil {
// 		t.Fatal("Expected non-nil income statement")
// 	}
//
// 	t.Logf("Income Statement - Fields: %d, Dates: %d", len(income.Fields()), len(income.Dates))
//
// 	// Check for common fields
// 	if revenue, ok := income.GetLatest("TotalRevenue"); ok {
// 		t.Logf("Latest Total Revenue: %.2fB", revenue/1e9)
// 	}
//
// 	if netIncome, ok := income.GetLatest("NetIncome"); ok {
// 		t.Logf("Latest Net Income: %.2fB", netIncome/1e9)
// 	}
//
// 	// Test Balance Sheet
// 	balance, err := tkr.BalanceSheet("annual")
// 	if err != nil {
// 		t.Fatalf("Failed to get balance sheet: %v", err)
// 	}
//
// 	t.Logf("Balance Sheet - Fields: %d, Dates: %d", len(balance.Fields()), len(balance.Dates))
//
// 	if assets, ok := balance.GetLatest("TotalAssets"); ok {
// 		t.Logf("Latest Total Assets: %.2fB", assets/1e9)
// 	}
//
// 	// Test Cash Flow
// 	cashFlow, err := tkr.CashFlow("annual")
// 	if err != nil {
// 		t.Fatalf("Failed to get cash flow: %v", err)
// 	}
//
// 	t.Logf("Cash Flow - Fields: %d, Dates: %d", len(cashFlow.Fields()), len(cashFlow.Dates))
//
// 	if opCashFlow, ok := cashFlow.GetLatest("OperatingCashFlow"); ok {
// 		t.Logf("Latest Operating Cash Flow: %.2fB", opCashFlow/1e9)
// 	}
//
// 	// Test Quarterly
// 	incomeQ, err := tkr.IncomeStatement("quarterly")
// 	if err != nil {
// 		t.Fatalf("Failed to get quarterly income: %v", err)
// 	}
//
// 	t.Logf("Quarterly Income - Dates: %d", len(incomeQ.Dates))
// }
