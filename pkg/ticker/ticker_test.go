package ticker

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func TestGetHistoryMetadataReturnsDeepClone(t *testing.T) {
	tkr, _ := New("AAPL")
	pre := int64(1)
	meta := models.ChartMeta{Symbol: "AAPL", ValidRanges: []string{"1d"}}
	meta = meta.WithTradingPeriods([]models.TradingPeriod{{Start: 2, End: 3, PreStart: &pre}})
	tkr.setHistoryMetadata(&meta)
	first := tkr.GetHistoryMetadata()
	first.ValidRanges[0] = "changed"
	*first.TradingPeriods[0].PreStart = 99
	second := tkr.GetHistoryMetadata()
	if second.ValidRanges[0] != "1d" || *second.TradingPeriods[0].PreStart != 1 {
		t.Fatalf("cached metadata was mutated: %+v", second)
	}
}

func TestGetTradingPeriodsCachesClonesAndPreservesBase(t *testing.T) {
	tkr, _ := New("AAPL")
	tkr.setHistoryMetadata(&models.ChartMeta{Symbol: "BASE", Currency: "USD"})
	var calls atomic.Int32
	tkr.tradingPeriodsFetcher = func() (*models.ChartMeta, error) {
		calls.Add(1)
		meta := models.ChartMeta{Symbol: "INTRADAY"}.WithTradingPeriods([]models.TradingPeriod{{Start: 10, End: 20}})
		return &meta, nil
	}
	periods, err := tkr.GetTradingPeriods()
	if err != nil {
		t.Fatal(err)
	}
	periods[0].Start = 999
	again, err := tkr.GetTradingPeriods()
	if err != nil || again[0].Start != 10 || calls.Load() != 1 {
		t.Fatalf("periods=%+v calls=%d err=%v", again, calls.Load(), err)
	}
	if meta := tkr.GetHistoryMetadata(); meta.Symbol != "BASE" || meta.Currency != "USD" {
		t.Fatalf("base replaced: %+v", meta)
	}
}

func TestSetHistoryMetadataPreservesCachedTradingPeriods(t *testing.T) {
	tkr, _ := New("AAPL")
	intraday := models.ChartMeta{Symbol: "OLD"}.WithTradingPeriods([]models.TradingPeriod{{Start: 10, End: 20}})
	tkr.setHistoryMetadata(&intraday)
	tkr.setHistoryMetadata(&models.ChartMeta{Symbol: "NEW", Currency: "USD"})
	meta := tkr.GetHistoryMetadata()
	if meta.Symbol != "NEW" || len(meta.TradingPeriods) != 1 || meta.TradingPeriods[0].Start != 10 {
		t.Fatalf("metadata merge failed: %+v", meta)
	}
}

func TestGetTradingPeriodsCoalescesConcurrentLoad(t *testing.T) {
	tkr, _ := New("AAPL")
	var calls atomic.Int32
	release := make(chan struct{})
	tkr.tradingPeriodsFetcher = func() (*models.ChartMeta, error) {
		calls.Add(1)
		<-release
		meta := models.ChartMeta{}.WithTradingPeriods([]models.TradingPeriod{{Start: 10, End: 20}})
		return &meta, nil
	}
	const workers = 12
	var wg sync.WaitGroup
	wg.Add(workers)
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			p, err := tkr.GetTradingPeriods()
			if err == nil && len(p) != 1 {
				err = errors.New("missing period")
			}
			errs <- err
		}()
	}
	time.Sleep(10 * time.Millisecond)
	close(release)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("fetches=%d, want 1", calls.Load())
	}
}

func TestGetTradingPeriodsCoalescesConcurrentFailure(t *testing.T) {
	tkr, _ := New("AAPL")
	var calls atomic.Int32
	release := make(chan struct{})
	tkr.tradingPeriodsFetcher = func() (*models.ChartMeta, error) {
		calls.Add(1)
		<-release
		return nil, errors.New("temporary")
	}
	const workers = 12
	var wg sync.WaitGroup
	wg.Add(workers)
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		go func() { defer wg.Done(); _, err := tkr.GetTradingPeriods(); errs <- err }()
	}
	time.Sleep(10 * time.Millisecond)
	close(release)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err == nil || err.Error() != "temporary" {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("fetches=%d, want 1", calls.Load())
	}
}

func TestGetTradingPeriodsFailureRetriesAndClearCacheResets(t *testing.T) {
	tkr, _ := New("AAPL")
	var calls atomic.Int32
	tkr.tradingPeriodsFetcher = func() (*models.ChartMeta, error) {
		if calls.Add(1) == 1 {
			return nil, errors.New("temporary")
		}
		meta := models.ChartMeta{}.WithTradingPeriods([]models.TradingPeriod{{Start: 10, End: 20}})
		return &meta, nil
	}
	if _, err := tkr.GetTradingPeriods(); err == nil {
		t.Fatal("expected failure")
	}
	if _, err := tkr.GetTradingPeriods(); err != nil {
		t.Fatalf("retry: %v", err)
	}
	tkr.ClearCache()
	if _, err := tkr.GetTradingPeriods(); err != nil {
		t.Fatalf("after clear: %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("fetches=%d, want 3", calls.Load())
	}
}

func TestGetTradingPeriodsClearDuringLoadInvalidatesResult(t *testing.T) {
	tkr, _ := New("AAPL")
	started := make(chan struct{})
	release := make(chan struct{})
	tkr.tradingPeriodsFetcher = func() (*models.ChartMeta, error) {
		close(started)
		<-release
		meta := models.ChartMeta{}.WithTradingPeriods([]models.TradingPeriod{{Start: 10, End: 20}})
		return &meta, nil
	}
	errCh := make(chan error, 1)
	go func() { _, err := tkr.GetTradingPeriods(); errCh <- err }()
	<-started
	tkr.ClearCache()
	close(release)
	if err := <-errCh; err == nil {
		t.Fatal("in-flight result survived ClearCache")
	}
	if meta := tkr.GetHistoryMetadata(); meta != nil {
		t.Fatalf("cleared metadata repopulated: %+v", meta)
	}
}

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
