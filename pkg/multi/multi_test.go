package multi

import (
	"testing"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func TestNewTickers(t *testing.T) {
	// Test with valid symbols
	tickers, err := NewTickers([]string{"AAPL", "MSFT", "GOOGL"})
	if err != nil {
		t.Fatalf("Failed to create tickers: %v", err)
	}

	if tickers.Count() != 3 {
		t.Errorf("Expected 3 tickers, got %d", tickers.Count())
	}

	// Note: Don't call Close() since CycleTLS is lazily initialized
}

func TestNewTickersEmpty(t *testing.T) {
	_, err := NewTickers([]string{})
	if err == nil {
		t.Error("Expected error for empty symbols")
	}
}

func TestNewTickersWhitespaceOnly(t *testing.T) {
	_, err := NewTickers([]string{"  ", ""})
	if err == nil {
		t.Error("Expected error for whitespace-only symbols")
	}
}

func TestNewTickersFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"space separated", "AAPL MSFT GOOGL", 3},
		{"comma separated", "AAPL,MSFT,GOOGL", 3},
		{"mixed separators", "AAPL, MSFT GOOGL", 3},
		{"extra whitespace", "  AAPL   MSFT  ", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tickers, err := NewTickersFromString(tt.input)
			if err != nil {
				t.Fatalf("Failed to create tickers: %v", err)
			}
			if tickers.Count() != tt.expected {
				t.Errorf("Expected %d tickers, got %d", tt.expected, tickers.Count())
			}
		})
	}
}

func TestNewTickersFromStringEmpty(t *testing.T) {
	_, err := NewTickersFromString("")
	if err == nil {
		t.Error("Expected error for empty string")
	}
}

func TestTickersSymbols(t *testing.T) {
	tickers, err := NewTickers([]string{"aapl", "MSFT"})
	if err != nil {
		t.Fatalf("Failed to create tickers: %v", err)
	}

	symbols := tickers.Symbols()
	if len(symbols) != 2 {
		t.Fatalf("Expected 2 symbols, got %d", len(symbols))
	}

	// Symbols should be normalized to uppercase
	if symbols[0] != "AAPL" {
		t.Errorf("Expected 'AAPL', got '%s'", symbols[0])
	}
	if symbols[1] != "MSFT" {
		t.Errorf("Expected 'MSFT', got '%s'", symbols[1])
	}
}

func TestTickersGet(t *testing.T) {
	tickers, err := NewTickers([]string{"AAPL", "MSFT"})
	if err != nil {
		t.Fatalf("Failed to create tickers: %v", err)
	}

	// Test getting existing ticker
	aapl := tickers.Get("AAPL")
	if aapl == nil {
		t.Error("Expected to get AAPL ticker")
	}

	// Test case-insensitivity
	aapl2 := tickers.Get("aapl")
	if aapl2 == nil {
		t.Error("Expected to get AAPL ticker with lowercase")
	}

	// Test non-existent ticker
	unknown := tickers.Get("UNKNOWN")
	if unknown != nil {
		t.Error("Expected nil for unknown ticker")
	}
}

func TestDefaultDownloadParams(t *testing.T) {
	params := models.DefaultDownloadParams()

	if params.Period != "1mo" {
		t.Errorf("Expected period '1mo', got '%s'", params.Period)
	}
	if params.Interval != "1d" {
		t.Errorf("Expected interval '1d', got '%s'", params.Interval)
	}
	if !params.AutoAdjust {
		t.Error("Expected AutoAdjust to be true")
	}
	if params.Threads != 0 {
		t.Errorf("Expected Threads 0, got %d", params.Threads)
	}
	if params.Timeout != 10 {
		t.Errorf("Expected Timeout 10, got %d", params.Timeout)
	}
}

func TestMultiTickerResult(t *testing.T) {
	result := &models.MultiTickerResult{
		Data: map[string][]models.Bar{
			"AAPL": {{Close: 150.0}},
			"MSFT": {{Close: 300.0}},
		},
		Errors: map[string]error{
			"INVALID": nil, // Simulating an error entry
		},
		Symbols: []string{"AAPL", "MSFT"},
	}

	// Test Get
	aaplBars := result.Get("AAPL")
	if len(aaplBars) != 1 {
		t.Errorf("Expected 1 bar for AAPL, got %d", len(aaplBars))
	}

	unknownBars := result.Get("UNKNOWN")
	if unknownBars != nil {
		t.Error("Expected nil for unknown symbol")
	}

	// Test HasErrors
	if !result.HasErrors() {
		t.Error("Expected HasErrors to be true")
	}

	// Test SuccessCount
	if result.SuccessCount() != 2 {
		t.Errorf("Expected SuccessCount 2, got %d", result.SuccessCount())
	}

	// Test ErrorCount
	if result.ErrorCount() != 1 {
		t.Errorf("Expected ErrorCount 1, got %d", result.ErrorCount())
	}
}

func TestMultiTickerResultNilData(t *testing.T) {
	result := &models.MultiTickerResult{}

	// Should not panic
	bars := result.Get("AAPL")
	if bars != nil {
		t.Error("Expected nil for nil data map")
	}

	if result.HasErrors() {
		t.Error("Expected HasErrors false for nil errors map")
	}

	if result.SuccessCount() != 0 {
		t.Error("Expected SuccessCount 0")
	}

	if result.ErrorCount() != 0 {
		t.Error("Expected ErrorCount 0")
	}
}

// Integration tests (require network access)
// These are skipped by default; run with: go test -v -short=false

// func TestDownloadIntegration(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("Skipping integration test")
// 	}
//
// 	result, err := Download([]string{"AAPL", "MSFT"}, nil)
// 	if err != nil {
// 		t.Fatalf("Failed to download: %v", err)
// 	}
// 	defer func() {
// 		// Cleanup is handled internally
// 	}()
//
// 	if result.SuccessCount() == 0 {
// 		t.Error("Expected at least one successful download")
// 	}
//
// 	for symbol, bars := range result.Data {
// 		t.Logf("%s: %d bars", symbol, len(bars))
// 	}
// }
