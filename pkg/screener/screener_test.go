package screener

import (
	"testing"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func TestNew(t *testing.T) {
	s, err := New()
	if err != nil {
		t.Fatalf("Failed to create Screener: %v", err)
	}

	if s == nil {
		t.Fatal("Screener should not be nil")
	}

	if s.ownsClient != true {
		t.Error("ownsClient should be true when no custom client is provided")
	}
}

func TestNewWithClient(t *testing.T) {
	// Create a Screener instance to get a client
	s1, err := New()
	if err != nil {
		t.Fatalf("Failed to create first Screener: %v", err)
	}

	// Create another Screener with the same client
	s2, err := New(WithClient(s1.client))
	if err != nil {
		t.Fatalf("Failed to create second Screener: %v", err)
	}

	if s2.ownsClient != false {
		t.Error("ownsClient should be false when custom client is provided")
	}

	if s2.client != s1.client {
		t.Error("Client should be the same instance")
	}
}

func TestDefaultScreenerParams(t *testing.T) {
	params := models.DefaultScreenerParams()

	if params.Offset != 0 {
		t.Errorf("Expected Offset to be 0, got %d", params.Offset)
	}

	if params.Count != 25 {
		t.Errorf("Expected Count to be 25, got %d", params.Count)
	}

	if params.SortField != "ticker" {
		t.Errorf("Expected SortField to be 'ticker', got '%s'", params.SortField)
	}

	if params.SortAsc != false {
		t.Error("Expected SortAsc to be false")
	}
}

func TestPredefinedScreeners(t *testing.T) {
	screeners := models.AllPredefinedScreeners()

	if len(screeners) == 0 {
		t.Error("Expected at least one predefined screener")
	}

	// Check that well-known screeners exist
	expectedScreeners := []models.PredefinedScreener{
		models.ScreenerDayGainers,
		models.ScreenerDayLosers,
		models.ScreenerMostActives,
	}

	for _, expected := range expectedScreeners {
		found := false
		for _, s := range screeners {
			if s == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected screener %s not found", expected)
		}
	}
}

func TestNewEquityQuery(t *testing.T) {
	query := models.NewEquityQuery(models.OpAND, []interface{}{
		models.NewEquityQuery(models.OpEQ, []interface{}{"region", "us"}),
		models.NewEquityQuery(models.OpGT, []interface{}{"intradayprice", 10}),
	})

	if query == nil {
		t.Fatal("Query should not be nil")
	}

	if query.Operator != models.OpAND {
		t.Errorf("Expected operator AND, got %s", query.Operator)
	}

	if len(query.Operands) != 2 {
		t.Errorf("Expected 2 operands, got %d", len(query.Operands))
	}
}

func TestQueryOperators(t *testing.T) {
	testCases := []struct {
		name     string
		operator models.QueryOperator
		expected string
	}{
		{"Equals", models.OpEQ, "eq"},
		{"Greater than", models.OpGT, "gt"},
		{"Less than", models.OpLT, "lt"},
		{"Greater than or equal", models.OpGTE, "gte"},
		{"Less than or equal", models.OpLTE, "lte"},
		{"Between", models.OpBTWN, "btwn"},
		{"AND", models.OpAND, "and"},
		{"OR", models.OpOR, "or"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if string(tc.operator) != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, tc.operator)
			}
		})
	}
}

func TestScreenWithQueryNilQuery(t *testing.T) {
	s, err := New()
	if err != nil {
		t.Fatalf("Failed to create Screener: %v", err)
	}

	_, err = s.ScreenWithQuery(nil, nil)
	if err == nil {
		t.Error("Expected error for nil query")
	}
}

func TestHelperFunctions(t *testing.T) {
	m := map[string]interface{}{
		"str":     "hello",
		"int":     42,
		"int64":   int64(100),
		"float":   3.14,
		"float64": float64(2.71),
	}

	// Test getString
	if got := getString(m, "str"); got != "hello" {
		t.Errorf("getString expected 'hello', got '%s'", got)
	}

	if got := getString(m, "missing"); got != "" {
		t.Errorf("getString for missing key expected '', got '%s'", got)
	}

	// Test getFloat
	if got := getFloat(m, "float"); got != 3.14 {
		t.Errorf("getFloat expected 3.14, got %f", got)
	}

	if got := getFloat(m, "int"); got != 42.0 {
		t.Errorf("getFloat for int expected 42.0, got %f", got)
	}

	if got := getFloat(m, "int64"); got != 100.0 {
		t.Errorf("getFloat for int64 expected 100.0, got %f", got)
	}

	// Test getInt64
	if got := getInt64(m, "int64"); got != 100 {
		t.Errorf("getInt64 expected 100, got %d", got)
	}

	if got := getInt64(m, "int"); got != 42 {
		t.Errorf("getInt64 for int expected 42, got %d", got)
	}

	if got := getInt64(m, "float64"); got != 2 {
		t.Errorf("getInt64 for float64 expected 2, got %d", got)
	}
}

// Integration tests (require network access)
// Run with: go test -v -run Integration

// func TestScreenDayGainersIntegration(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("Skipping integration test")
// 	}
//
// 	s, err := New()
// 	if err != nil {
// 		t.Fatalf("Failed to create Screener: %v", err)
// 	}
// 	defer s.Close()
//
// 	result, err := s.DayGainers(10)
// 	if err != nil {
// 		t.Fatalf("DayGainers failed: %v", err)
// 	}
//
// 	if len(result.Quotes) == 0 {
// 		t.Error("Expected at least one quote")
// 	}
//
// 	// Check that all quotes have positive change
// 	for _, q := range result.Quotes {
// 		if q.RegularMarketChangePercent <= 0 {
// 			t.Logf("Warning: Quote %s has non-positive change: %.2f%%",
// 				q.Symbol, q.RegularMarketChangePercent)
// 		}
// 	}
// }
//
// func TestScreenMostActivesIntegration(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("Skipping integration test")
// 	}
//
// 	s, err := New()
// 	if err != nil {
// 		t.Fatalf("Failed to create Screener: %v", err)
// 	}
// 	defer s.Close()
//
// 	result, err := s.MostActives(5)
// 	if err != nil {
// 		t.Fatalf("MostActives failed: %v", err)
// 	}
//
// 	if len(result.Quotes) == 0 {
// 		t.Error("Expected at least one quote")
// 	}
//
// 	// Check that all quotes have volume
// 	for _, q := range result.Quotes {
// 		if q.RegularMarketVolume <= 0 {
// 			t.Errorf("Quote %s has no volume", q.Symbol)
// 		}
// 	}
// }
