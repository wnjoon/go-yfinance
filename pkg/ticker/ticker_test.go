package ticker

import (
	"testing"
)

func TestNew(t *testing.T) {
	// Test with valid symbol
	tkr, err := New("AAPL")
	if err != nil {
		t.Fatalf("Failed to create ticker: %v", err)
	}

	if tkr.Symbol() != "AAPL" {
		t.Errorf("Symbol should be 'AAPL', got '%s'", tkr.Symbol())
	}

	// Note: Don't call Close() here since CycleTLS is lazily initialized
	// and calling Close() on uninitialized client would panic
}

func TestNewWithEmptySymbol(t *testing.T) {
	_, err := New("")
	if err == nil {
		t.Error("Expected error for empty symbol")
	}
}

func TestSymbolNormalization(t *testing.T) {
	tkr, err := New("aapl")
	if err != nil {
		t.Fatalf("Failed to create ticker: %v", err)
	}

	if tkr.Symbol() != "AAPL" {
		t.Errorf("Symbol should be normalized to 'AAPL', got '%s'", tkr.Symbol())
	}
}

func TestClearCache(t *testing.T) {
	tkr, err := New("AAPL")
	if err != nil {
		t.Fatalf("Failed to create ticker: %v", err)
	}

	// Should not panic
	tkr.ClearCache()
}

func TestGetHistoryMetadata(t *testing.T) {
	tkr, err := New("AAPL")
	if err != nil {
		t.Fatalf("Failed to create ticker: %v", err)
	}

	// Initially should be nil
	meta := tkr.GetHistoryMetadata()
	if meta != nil {
		t.Error("Metadata should be nil initially")
	}
}

// Integration tests (require network access)
// These are skipped by default; run with: go test -v -tags=integration

// func TestQuoteIntegration(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("Skipping integration test")
// 	}
//
// 	tkr, err := New("AAPL")
// 	if err != nil {
// 		t.Fatalf("Failed to create ticker: %v", err)
// 	}
// 	defer tkr.Close()
//
// 	quote, err := tkr.Quote()
// 	if err != nil {
// 		t.Fatalf("Failed to get quote: %v", err)
// 	}
//
// 	if quote.Symbol != "AAPL" {
// 		t.Errorf("Expected symbol 'AAPL', got '%s'", quote.Symbol)
// 	}
// 	if quote.RegularMarketPrice <= 0 {
// 		t.Error("Market price should be positive")
// 	}
// }
//
// func TestHistoryIntegration(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("Skipping integration test")
// 	}
//
// 	tkr, err := New("AAPL")
// 	if err != nil {
// 		t.Fatalf("Failed to create ticker: %v", err)
// 	}
// 	defer tkr.Close()
//
// 	bars, err := tkr.HistoryPeriod("1mo")
// 	if err != nil {
// 		t.Fatalf("Failed to get history: %v", err)
// 	}
//
// 	if len(bars) == 0 {
// 		t.Error("Should have at least one bar")
// 	}
// }
//
// func TestInfoIntegration(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("Skipping integration test")
// 	}
//
// 	tkr, err := New("AAPL")
// 	if err != nil {
// 		t.Fatalf("Failed to create ticker: %v", err)
// 	}
// 	defer tkr.Close()
//
// 	info, err := tkr.Info()
// 	if err != nil {
// 		t.Fatalf("Failed to get info: %v", err)
// 	}
//
// 	if info.Symbol != "AAPL" {
// 		t.Errorf("Expected symbol 'AAPL', got '%s'", info.Symbol)
// 	}
// 	if info.Sector == "" {
// 		t.Error("Sector should not be empty")
// 	}
// }
