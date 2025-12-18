// Package models provides data structures for Yahoo Finance API responses.
package models

import "time"

// Frequency represents the time frequency for financial data.
type Frequency string

const (
	FrequencyAnnual    Frequency = "annual"
	FrequencyQuarterly Frequency = "quarterly"
	FrequencyTrailing  Frequency = "trailing"
)

// FinancialItem represents a single financial data point.
type FinancialItem struct {
	AsOfDate     time.Time `json:"asOfDate"`
	CurrencyCode string    `json:"currencyCode"`
	PeriodType   string    `json:"periodType"` // "12M" for annual, "3M" for quarterly
	Value        float64   `json:"value"`
	Formatted    string    `json:"formatted"`
}

// FinancialStatement represents a financial statement (income, balance sheet, or cash flow).
// It maps field names to their historical values.
type FinancialStatement struct {
	// Data maps field names (e.g., "TotalRevenue") to time-ordered values.
	// Each slice is ordered by date ascending.
	Data map[string][]FinancialItem `json:"data"`

	// Dates contains all unique dates in the statement, ordered ascending.
	Dates []time.Time `json:"dates"`

	// Currency is the primary currency of the statement.
	Currency string `json:"currency"`
}

// NewFinancialStatement creates an empty FinancialStatement.
func NewFinancialStatement() *FinancialStatement {
	return &FinancialStatement{
		Data:  make(map[string][]FinancialItem),
		Dates: []time.Time{},
	}
}

// Get returns the value for a specific field and date.
// Returns 0 and false if not found.
func (fs *FinancialStatement) Get(field string, date time.Time) (float64, bool) {
	items, ok := fs.Data[field]
	if !ok {
		return 0, false
	}

	for _, item := range items {
		if item.AsOfDate.Equal(date) {
			return item.Value, true
		}
	}
	return 0, false
}

// GetLatest returns the most recent value for a field.
// Returns 0 and false if not found.
func (fs *FinancialStatement) GetLatest(field string) (float64, bool) {
	items, ok := fs.Data[field]
	if !ok || len(items) == 0 {
		return 0, false
	}
	return items[len(items)-1].Value, true
}

// Fields returns all available field names in the statement.
func (fs *FinancialStatement) Fields() []string {
	fields := make([]string, 0, len(fs.Data))
	for k := range fs.Data {
		fields = append(fields, k)
	}
	return fields
}

// Financials holds all financial statements for a ticker.
type Financials struct {
	// Income statements
	IncomeStatementAnnual    *FinancialStatement `json:"incomeStatementAnnual,omitempty"`
	IncomeStatementQuarterly *FinancialStatement `json:"incomeStatementQuarterly,omitempty"`

	// Balance sheets
	BalanceSheetAnnual    *FinancialStatement `json:"balanceSheetAnnual,omitempty"`
	BalanceSheetQuarterly *FinancialStatement `json:"balanceSheetQuarterly,omitempty"`

	// Cash flow statements
	CashFlowAnnual    *FinancialStatement `json:"cashFlowAnnual,omitempty"`
	CashFlowQuarterly *FinancialStatement `json:"cashFlowQuarterly,omitempty"`
}

// TimeseriesResponse represents the API response from fundamentals-timeseries endpoint.
type TimeseriesResponse struct {
	Timeseries struct {
		Result []TimeseriesResult `json:"result"`
		Error  *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"timeseries"`
}

// TimeseriesResult represents a single result item in the timeseries response.
// The actual data is in a dynamically named field matching the type (e.g., "annualTotalRevenue").
type TimeseriesResult struct {
	Meta struct {
		Symbol []string `json:"symbol"`
		Type   []string `json:"type"`
	} `json:"meta"`
	Timestamp []int64 `json:"timestamp"`

	// Raw data - we'll unmarshal this separately
	RawData []TimeseriesDataPoint `json:"-"`
}

// TimeseriesDataPoint represents a single data point in the timeseries.
type TimeseriesDataPoint struct {
	AsOfDate      string `json:"asOfDate"`
	CurrencyCode  string `json:"currencyCode"`
	DataID        int    `json:"dataId"`
	PeriodType    string `json:"periodType"`
	ReportedValue struct {
		Fmt string  `json:"fmt"`
		Raw float64 `json:"raw"`
	} `json:"reportedValue"`
}
