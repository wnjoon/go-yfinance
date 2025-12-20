package search

import (
	"testing"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func TestNew(t *testing.T) {
	s, err := New()
	if err != nil {
		t.Fatalf("Failed to create Search: %v", err)
	}

	if s == nil {
		t.Error("Search should not be nil")
	}

	if s.ownsClient != true {
		t.Error("ownsClient should be true when no custom client is provided")
	}
}

func TestNewWithClient(t *testing.T) {
	// Create a Search instance to get a client
	s1, err := New()
	if err != nil {
		t.Fatalf("Failed to create first Search: %v", err)
	}

	// Create another Search with the same client
	s2, err := New(WithClient(s1.client))
	if err != nil {
		t.Fatalf("Failed to create second Search: %v", err)
	}

	if s2.ownsClient != false {
		t.Error("ownsClient should be false when custom client is provided")
	}

	if s2.client != s1.client {
		t.Error("Client should be the same instance")
	}
}

func TestSearchWithEmptyQuery(t *testing.T) {
	s, err := New()
	if err != nil {
		t.Fatalf("Failed to create Search: %v", err)
	}

	_, err = s.Search("")
	if err == nil {
		t.Error("Expected error for empty query")
	}
}

func TestSearchParamsValidation(t *testing.T) {
	s, err := New()
	if err != nil {
		t.Fatalf("Failed to create Search: %v", err)
	}

	params := models.SearchParams{
		Query: "", // Empty query should fail
	}

	_, err = s.SearchWithParams(params)
	if err == nil {
		t.Error("Expected error for empty query in params")
	}
}

func TestDefaultSearchParams(t *testing.T) {
	params := models.DefaultSearchParams()

	if params.MaxResults != 8 {
		t.Errorf("Expected MaxResults to be 8, got %d", params.MaxResults)
	}

	if params.NewsCount != 8 {
		t.Errorf("Expected NewsCount to be 8, got %d", params.NewsCount)
	}

	if params.ListsCount != 8 {
		t.Errorf("Expected ListsCount to be 8, got %d", params.ListsCount)
	}
}

func TestHelperFunctions(t *testing.T) {
	// Test getString
	m := map[string]interface{}{
		"str":   "hello",
		"int":   42,
		"float": 3.14,
		"bool":  true,
	}

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

	// Test getBool
	if got := getBool(m, "bool"); got != true {
		t.Errorf("getBool expected true, got %v", got)
	}

	if got := getBool(m, "missing"); got != false {
		t.Errorf("getBool for missing key expected false, got %v", got)
	}
}

func TestGetStringSlice(t *testing.T) {
	m := map[string]interface{}{
		"tags": []interface{}{"a", "b", "c"},
	}

	result := getStringSlice(m, "tags")
	if len(result) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(result))
	}

	if result[0] != "a" || result[1] != "b" || result[2] != "c" {
		t.Errorf("Unexpected values: %v", result)
	}

	// Test missing key
	result = getStringSlice(m, "missing")
	if result != nil {
		t.Errorf("Expected nil for missing key, got %v", result)
	}
}

// Integration tests (require network access)
// Run with: go test -v -run Integration

// func TestSearchIntegration(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("Skipping integration test")
// 	}
//
// 	s, err := New()
// 	if err != nil {
// 		t.Fatalf("Failed to create Search: %v", err)
// 	}
// 	defer s.Close()
//
// 	result, err := s.Search("Apple")
// 	if err != nil {
// 		t.Fatalf("Search failed: %v", err)
// 	}
//
// 	if len(result.Quotes) == 0 {
// 		t.Error("Expected at least one quote")
// 	}
//
// 	// Check if AAPL is in the results
// 	found := false
// 	for _, q := range result.Quotes {
// 		if q.Symbol == "AAPL" {
// 			found = true
// 			break
// 		}
// 	}
// 	if !found {
// 		t.Error("Expected AAPL in search results for 'Apple'")
// 	}
// }
//
// func TestQuotesIntegration(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("Skipping integration test")
// 	}
//
// 	s, err := New()
// 	if err != nil {
// 		t.Fatalf("Failed to create Search: %v", err)
// 	}
// 	defer s.Close()
//
// 	quotes, err := s.Quotes("Tesla", 5)
// 	if err != nil {
// 		t.Fatalf("Quotes failed: %v", err)
// 	}
//
// 	if len(quotes) == 0 {
// 		t.Error("Expected at least one quote")
// 	}
//
// 	if len(quotes) > 5 {
// 		t.Errorf("Expected at most 5 quotes, got %d", len(quotes))
// 	}
// }
