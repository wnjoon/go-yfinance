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

// Tests below demonstrate sequential batch processing for multiple symbols
// This test is not required to run when running total tests, will be commented out
// func TestBatchCallSequential(t *testing.T) {
// 	start := time.Now()
// 	for _, symbol := range symbols {
// 		t.Logf("Processing symbol: %s", symbol)
// 		tkr, err := New(symbol)
// 		assert.NoError(t, err)
// 		assert.NotNil(t, tkr)

// 		quote, err := tkr.Quote()
// 		assert.NoError(t, err)
// 		assert.NotNil(t, quote)
// 		assert.Equal(t, symbol, quote.Symbol)

// 		bars, err := tkr.History(models.HistoryParams{
// 			Period:     "1mo",
// 			Interval:   "1d",
// 			AutoAdjust: true,
// 		})
// 		assert.NoError(t, err)
// 		assert.NotEmpty(t, bars)
// 		assert.Greater(t, len(bars), 0)

// 		info, err := tkr.Info()
// 		assert.NoError(t, err)
// 		assert.NotNil(t, info)
// 		assert.Equal(t, symbol, info.Symbol)
// 	}
// 	duration := time.Since(start)
// 	t.Logf("[Sequential] Batch call completed in %v for %d symbols (avg: %v per symbol)", duration, len(symbols), duration/time.Duration(len(symbols)))
// }

// func TestBatchCallParallel(t *testing.T) {
// 	start := time.Now()
// 	var wg sync.WaitGroup
// 	for _, symbol := range symbols {
// 		wg.Add(1)
// 		go func(sym string) {
// 			defer wg.Done()
// 			t.Logf("Processing symbol: %s", sym)
// 			tkr, err := New(sym)
// 			assert.NoError(t, err)
// 			assert.NotNil(t, tkr)

// 			quote, err := tkr.Quote()
// 			assert.NoError(t, err)
// 			assert.NotNil(t, quote)
// 			assert.Equal(t, sym, quote.Symbol)

// 			bars, err := tkr.History(models.HistoryParams{
// 				Period:     "1mo",
// 				Interval:   "1d",
// 				AutoAdjust: true,
// 			})
// 			assert.NoError(t, err)
// 			assert.NotEmpty(t, bars)
// 			assert.Greater(t, len(bars), 0)

// 			info, err := tkr.Info()
// 			assert.NoError(t, err)
// 			assert.NotNil(t, info)
// 			assert.Equal(t, sym, info.Symbol)
// 		}(symbol)
// 	}
// 	wg.Wait()
// 	duration := time.Since(start)
// 	t.Logf("[Parallel] Batch call completed in %v for %d symbols (avg: %v per symbol)", duration, len(symbols), duration/time.Duration(len(symbols)))
// }

// var symbols = []string{
// 	// US Tech & Mega Cap
// 	"AAPL", "MSFT", "GOOGL", "AMZN", "NVDA", "TSLA", "META", "AMD", "INTC", "NFLX",
// 	"ORCL", "ADBE", "CRM", "CSCO", "AVGO", "QCOM", "TXN", "IBM", "UBER", "ABNB",

// 	// Finance & Banking
// 	"JPM", "BAC", "WFC", "C", "GS", "MS", "V", "MA", "AXP", "BLK",

// 	// Consumer & Retail
// 	"WMT", "TGT", "COST", "KO", "PEP", "MCD", "SBUX", "NKE", "DIS", "HD",
// 	"PG", "CL", "EL", "LULU", "CMG",

// 	// Healthcare
// 	"JNJ", "PFE", "MRK", "ABBV", "LLY", "UNH", "AMGN", "GILD", "BMY", "CVS",

// 	// Industrial & Energy
// 	"XOM", "CVX", "GE", "BA", "CAT", "F", "GM", "TM", "BP", "SHEL",
// 	"LMT", "RTX", "HON", "UNP", "UPS",

// 	// ETFs (Index, Bond, Commodity)
// 	"SPY", "QQQ", "DIA", "IWM", "VOO", "IVV", "VTI", // Indices
// 	"TLT", "AGG", "BND", // Bonds
// 	"GLD", "SLV", "USO", // Commodities
// 	"JEPI", "SCHD", "ARKK", // Dividends & Active

// 	// Cryptocurrencies (Yahoo format: TICKER-USD)
// 	"BTC-USD", "ETH-USD", "SOL-USD", "XRP-USD", "DOGE-USD", "BNB-USD", "ADA-USD",

// 	// Global / International (Testing Suffixes)
// 	"005930.KS", "000660.KS", "035420.KS", // Korea (Samsung, SK Hynix, Naver)
// 	"7203.T", "6758.T", // Japan (Toyota, Sony)
// 	"2330.TW", // Taiwan (TSMC)
// 	"0700.HK", // Hong Kong (Tencent)
// 	"SAP.DE",  // Germany (SAP)
// 	"MC.PA",   // France (LVMH)
// 	"ULVR.L",  // UK (Unilever)

// 	// Edge Cases (Hyphens, Classes, Mutual Funds)
// 	"BRK-B", "BF-B", // Berkshire Hathaway, Brown-Forman (Hyphen handling)
// 	"GOOG",                   // Class C (vs GOOGL Class A)
// 	"VFIAX",                  // Mutual Fund (5 letters)
// 	"^GSPC", "^DJI", "^IXIC", // Indices (Carat symbol handling)
// }
