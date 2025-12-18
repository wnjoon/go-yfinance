package ticker

import (
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
