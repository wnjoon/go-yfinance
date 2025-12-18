package ticker

import (
	"testing"
	"time"
)

func TestOptionsModels(t *testing.T) {
	// Test optionsCache struct
	cache := &optionsCache{
		expirations: make(map[string]int64),
		strikes:     []float64{100.0, 105.0, 110.0},
	}

	cache.expirations["2024-01-19"] = 1705622400
	cache.expirations["2024-02-16"] = 1708041600

	if len(cache.expirations) != 2 {
		t.Errorf("Expected 2 expirations, got %d", len(cache.expirations))
	}

	if len(cache.strikes) != 3 {
		t.Errorf("Expected 3 strikes, got %d", len(cache.strikes))
	}
}

func TestTickerWithOptionsCache(t *testing.T) {
	// Test that Ticker properly initializes with nil optionsCache
	tkr, err := New("AAPL")
	if err != nil {
		t.Fatalf("Failed to create ticker: %v", err)
	}
	defer tkr.Close()

	if tkr.optionsCache != nil {
		t.Error("Expected optionsCache to be nil initially")
	}
}

func TestClearCacheIncludesOptions(t *testing.T) {
	tkr, err := New("AAPL")
	if err != nil {
		t.Fatalf("Failed to create ticker: %v", err)
	}
	defer tkr.Close()

	// Manually set optionsCache
	tkr.optionsCache = &optionsCache{
		expirations: map[string]int64{"2024-01-19": 1705622400},
		strikes:     []float64{100.0},
	}

	// Clear cache
	tkr.ClearCache()

	if tkr.optionsCache != nil {
		t.Error("Expected optionsCache to be nil after ClearCache")
	}
}

func TestOptionModelHelpers(t *testing.T) {
	// Import models package for this test
	// Test time conversion helpers

	// Unix timestamp for 2024-01-19 00:00:00 UTC
	timestamp := int64(1705622400)
	expected := time.Unix(timestamp, 0)

	if expected.Year() != 2024 {
		t.Errorf("Expected year 2024, got %d", expected.Year())
	}

	if expected.Month() != time.January {
		t.Errorf("Expected January, got %v", expected.Month())
	}
}

// Integration test - commented out for CI, run manually
// func TestOptionsLive(t *testing.T) {
// 	tkr, err := New("AAPL")
// 	if err != nil {
// 		t.Fatalf("Failed to create ticker: %v", err)
// 	}
// 	defer tkr.Close()
//
// 	// Test Options() - get expiration dates
// 	expirations, err := tkr.Options()
// 	if err != nil {
// 		t.Fatalf("Failed to get options: %v", err)
// 	}
//
// 	if len(expirations) == 0 {
// 		t.Error("Expected at least one expiration date")
// 	}
//
// 	t.Logf("Found %d expiration dates", len(expirations))
// 	for i, exp := range expirations[:min(5, len(expirations))] {
// 		t.Logf("  %d: %s", i+1, exp.Format("2006-01-02"))
// 	}
//
// 	// Test OptionChain() - get nearest expiration
// 	chain, err := tkr.OptionChain("")
// 	if err != nil {
// 		t.Fatalf("Failed to get option chain: %v", err)
// 	}
//
// 	if chain == nil {
// 		t.Fatal("Expected non-nil option chain")
// 	}
//
// 	t.Logf("Calls: %d, Puts: %d", len(chain.Calls), len(chain.Puts))
//
// 	if len(chain.Calls) > 0 {
// 		call := chain.Calls[0]
// 		t.Logf("First call: Strike=%.2f, Bid=%.2f, Ask=%.2f, IV=%.4f",
// 			call.Strike, call.Bid, call.Ask, call.ImpliedVolatility)
// 	}
//
// 	// Test Strikes()
// 	strikes, err := tkr.Strikes()
// 	if err != nil {
// 		t.Fatalf("Failed to get strikes: %v", err)
// 	}
//
// 	t.Logf("Found %d strike prices", len(strikes))
// }
