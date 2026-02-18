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

	if params.UserID != "" {
		t.Errorf("Expected UserID to be empty, got '%s'", params.UserID)
	}

	if params.UserIDType != "guid" {
		t.Errorf("Expected UserIDType to be 'guid', got '%s'", params.UserIDType)
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
	eq1, err := models.NewEquityQuery("eq", []any{"region", "us"})
	if err != nil {
		t.Fatalf("Failed to create EQ query: %v", err)
	}

	gt1, err := models.NewEquityQuery("gt", []any{"intradayprice", 10})
	if err != nil {
		t.Fatalf("Failed to create GT query: %v", err)
	}

	query, err := models.NewEquityQuery("and", []any{eq1, gt1})
	if err != nil {
		t.Fatalf("Failed to create AND query: %v", err)
	}

	if query == nil {
		t.Fatal("Query should not be nil")
	}

	if query.QuoteType() != "EQUITY" {
		t.Errorf("Expected quoteType EQUITY, got %s", query.QuoteType())
	}

	dict := query.ToDict()
	if dict["operator"] != "AND" {
		t.Errorf("Expected operator AND, got %v", dict["operator"])
	}

	ops, ok := dict["operands"].([]any)
	if !ok {
		t.Fatal("operands should be []any")
	}
	if len(ops) != 2 {
		t.Errorf("Expected 2 operands, got %d", len(ops))
	}
}

func TestNewFundQuery(t *testing.T) {
	eq1, err := models.NewFundQuery("eq", []any{"exchange", "NAS"})
	if err != nil {
		t.Fatalf("Failed to create FundQuery EQ: %v", err)
	}

	if eq1.QuoteType() != "MUTUALFUND" {
		t.Errorf("Expected quoteType MUTUALFUND, got %s", eq1.QuoteType())
	}

	dict := eq1.ToDict()
	if dict["operator"] != "EQ" {
		t.Errorf("Expected operator EQ, got %v", dict["operator"])
	}
}

func TestISINExpansion(t *testing.T) {
	q, err := models.NewEquityQuery("is-in", []any{"exchange", "NMS", "NYQ"})
	if err != nil {
		t.Fatalf("Failed to create IS-IN query: %v", err)
	}

	dict := q.ToDict()

	// IS-IN should expand to OR
	if dict["operator"] != "OR" {
		t.Errorf("Expected IS-IN to expand to OR, got %v", dict["operator"])
	}

	ops, ok := dict["operands"].([]any)
	if !ok {
		t.Fatal("operands should be []any")
	}
	if len(ops) != 2 {
		t.Errorf("Expected 2 expanded EQ operands, got %d", len(ops))
	}

	// Each expanded child should be an EQ query
	for i, op := range ops {
		child, ok := op.(map[string]any)
		if !ok {
			t.Fatalf("operand %d should be map[string]any", i)
		}
		if child["operator"] != "EQ" {
			t.Errorf("operand %d: expected operator EQ, got %v", i, child["operator"])
		}
	}
}

func TestQueryValidation(t *testing.T) {
	tests := []struct {
		name      string
		op        string
		operands  []any
		expectErr bool
	}{
		{"valid EQ", "eq", []any{"region", "us"}, false},
		{"invalid field", "eq", []any{"nonexistent_field", "value"}, true},
		{"invalid EQ value", "eq", []any{"region", "invalid_region"}, true},
		{"valid GT", "gt", []any{"intradayprice", 10}, false},
		{"GT non-numeric", "gt", []any{"intradayprice", "not_a_number"}, true},
		{"valid BTWN", "btwn", []any{"peratio.lasttwelvemonths", 0, 20}, false},
		{"BTWN wrong length", "btwn", []any{"peratio.lasttwelvemonths", 0}, true},
		{"invalid operator", "INVALID", []any{"field", "value"}, true},
		{"empty operands", "eq", []any{}, true},
		{"valid sector", "eq", []any{"sector", "Technology"}, false},
		{"invalid sector", "eq", []any{"sector", "FakeSector"}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := models.NewEquityQuery(tc.op, tc.operands)
			if tc.expectErr && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tc.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestFundQueryValidation(t *testing.T) {
	tests := []struct {
		name      string
		op        string
		operands  []any
		expectErr bool
	}{
		{"valid exchange", "eq", []any{"exchange", "NAS"}, false},
		{"invalid exchange", "eq", []any{"exchange", "FAKE"}, true},
		{"valid categoryname", "eq", []any{"categoryname", "Large Blend"}, false},
		{"valid performanceratingoverall", "gt", []any{"performanceratingoverall", 3}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := models.NewFundQuery(tc.op, tc.operands)
			if tc.expectErr && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tc.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestQueryOperators(t *testing.T) {
	testCases := []struct {
		name     string
		operator string
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
		{"IS-IN", models.OpISIN, "is-in"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.operator != tc.expected {
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

func TestScreenWithQueryCountLimit(t *testing.T) {
	s, err := New()
	if err != nil {
		t.Fatalf("Failed to create Screener: %v", err)
	}

	q, _ := models.NewEquityQuery("eq", []any{"region", "us"})
	params := &models.ScreenerParams{Count: 300}
	_, err = s.ScreenWithQuery(q, params)
	if err == nil {
		t.Error("Expected error for count > 250")
	}
}

func TestPredefinedScreenerQueries(t *testing.T) {
	expectedEquity := []string{
		"aggressive_small_caps", "day_gainers", "day_losers",
		"growth_technology_stocks", "most_actives", "most_shorted_stocks",
		"small_cap_gainers", "undervalued_growth_stocks", "undervalued_large_caps",
	}

	expectedFund := []string{
		"conservative_foreign_funds", "high_yield_bond", "portfolio_anchors",
		"solid_large_growth_funds", "solid_midcap_growth_funds", "top_mutual_funds",
	}

	for _, name := range expectedEquity {
		pq, ok := PredefinedScreenerQueries[name]
		if !ok {
			t.Errorf("Expected predefined screener %q not found", name)
			continue
		}
		if pq.Query.QuoteType() != "EQUITY" {
			t.Errorf("Screener %q: expected EQUITY, got %s", name, pq.Query.QuoteType())
		}
	}

	for _, name := range expectedFund {
		pq, ok := PredefinedScreenerQueries[name]
		if !ok {
			t.Errorf("Expected predefined screener %q not found", name)
			continue
		}
		if pq.Query.QuoteType() != "MUTUALFUND" {
			t.Errorf("Screener %q: expected MUTUALFUND, got %s", name, pq.Query.QuoteType())
		}
	}

	// Check total count
	if len(PredefinedScreenerQueries) != 15 {
		t.Errorf("Expected 15 predefined screeners, got %d", len(PredefinedScreenerQueries))
	}

	// Verify day_losers has SortAsc = true
	if !PredefinedScreenerQueries["day_losers"].SortAsc {
		t.Error("day_losers should have SortAsc = true")
	}
}

func TestPredefinedScreenerToDicts(t *testing.T) {
	// Verify all predefined queries can serialize without panic
	for name, pq := range PredefinedScreenerQueries {
		dict := pq.Query.ToDict()
		if dict == nil {
			t.Errorf("Screener %q: ToDict returned nil", name)
		}
		if _, ok := dict["operator"]; !ok {
			t.Errorf("Screener %q: ToDict missing operator", name)
		}
		if _, ok := dict["operands"]; !ok {
			t.Errorf("Screener %q: ToDict missing operands", name)
		}
	}
}

func TestHelperFunctions(t *testing.T) {
	m := map[string]any{
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
