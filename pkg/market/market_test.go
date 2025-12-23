package market

import (
	"testing"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func TestNew(t *testing.T) {
	m, err := New("us_market")
	if err != nil {
		t.Fatalf("Failed to create Market: %v", err)
	}
	defer m.Close()

	if m == nil {
		t.Fatal("Market should not be nil")
	}

	if m.market != "us_market" {
		t.Errorf("Expected market 'us_market', got '%s'", m.market)
	}

	if m.ownsClient != true {
		t.Error("ownsClient should be true when no custom client is provided")
	}
}

func TestNewWithEmptyMarket(t *testing.T) {
	_, err := New("")
	if err == nil {
		t.Error("Expected error for empty market identifier")
	}
}

func TestNewWithPredefined(t *testing.T) {
	m, err := NewWithPredefined(models.MarketUS)
	if err != nil {
		t.Fatalf("Failed to create Market: %v", err)
	}
	defer m.Close()

	if m.market != "us_market" {
		t.Errorf("Expected market 'us_market', got '%s'", m.market)
	}
}

func TestNewWithClient(t *testing.T) {
	m1, err := New("us_market")
	if err != nil {
		t.Fatalf("Failed to create first Market: %v", err)
	}

	m2, err := New("gb_market", WithClient(m1.client))
	if err != nil {
		t.Fatalf("Failed to create second Market: %v", err)
	}

	if m2.ownsClient != false {
		t.Error("ownsClient should be false when custom client is provided")
	}

	if m2.client != m1.client {
		t.Error("Client should be the same instance")
	}

	m1.Close()
}

func TestMarket(t *testing.T) {
	m, err := New("jp_market")
	if err != nil {
		t.Fatalf("Failed to create Market: %v", err)
	}
	defer m.Close()

	if m.Market() != "jp_market" {
		t.Errorf("Expected market 'jp_market', got '%s'", m.Market())
	}
}

func TestPredefinedMarkets(t *testing.T) {
	tests := []struct {
		market   models.PredefinedMarket
		expected string
	}{
		{models.MarketUS, "us_market"},
		{models.MarketGB, "gb_market"},
		{models.MarketDE, "de_market"},
		{models.MarketFR, "fr_market"},
		{models.MarketJP, "jp_market"},
		{models.MarketHK, "hk_market"},
		{models.MarketCN, "cn_market"},
		{models.MarketCA, "ca_market"},
		{models.MarketAU, "au_market"},
		{models.MarketIN, "in_market"},
		{models.MarketKR, "kr_market"},
		{models.MarketBR, "br_market"},
	}

	for _, tt := range tests {
		if string(tt.market) != tt.expected {
			t.Errorf("PredefinedMarket %v: expected '%s', got '%s'", tt.market, tt.expected, string(tt.market))
		}
	}
}

func TestHelperFunctions(t *testing.T) {
	m := map[string]interface{}{
		"str":   "hello",
		"int":   42,
		"float": 3.14,
		"int64": int64(1234567890),
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

	if got := getFloat(m, "missing"); got != 0 {
		t.Errorf("getFloat for missing key expected 0, got %f", got)
	}

	// Test getInt64
	if got := getInt64(m, "int64"); got != 1234567890 {
		t.Errorf("getInt64 expected 1234567890, got %d", got)
	}

	if got := getInt64(m, "float"); got != 3 {
		t.Errorf("getInt64 for float expected 3, got %d", got)
	}
}

func TestClearCache(t *testing.T) {
	m, err := New("us_market")
	if err != nil {
		t.Fatalf("Failed to create Market: %v", err)
	}
	defer m.Close()

	// Add something to cache manually
	m.mu.Lock()
	m.statusCache = &models.MarketStatus{ID: "test"}
	m.summaryCache = models.MarketSummary{"TEST": models.MarketSummaryItem{}}
	m.mu.Unlock()

	// Verify cache has data
	m.mu.RLock()
	hasData := m.statusCache != nil && m.summaryCache != nil
	m.mu.RUnlock()
	if !hasData {
		t.Error("Cache should have data")
	}

	// Clear cache
	m.ClearCache()

	// Verify cache is empty
	m.mu.RLock()
	isEmpty := m.statusCache == nil && m.summaryCache == nil
	m.mu.RUnlock()
	if !isEmpty {
		t.Error("Cache should be empty after ClearCache")
	}
}

func TestParseSummary(t *testing.T) {
	m, err := New("us_market")
	if err != nil {
		t.Fatalf("Failed to create Market: %v", err)
	}
	defer m.Close()

	results := []map[string]interface{}{
		{
			"exchange":                   "SNP",
			"symbol":                     "^GSPC",
			"shortName":                  "S&P 500",
			"regularMarketPrice":         4500.50,
			"regularMarketChange":        25.75,
			"regularMarketChangePercent": 0.57,
			"marketState":                "REGULAR",
		},
		{
			"exchange":                   "DJI",
			"symbol":                     "^DJI",
			"shortName":                  "Dow 30",
			"regularMarketPrice":         35000.25,
			"regularMarketChange":        150.30,
			"regularMarketChangePercent": 0.43,
		},
	}

	summary := m.parseSummary(results)

	if len(summary) != 2 {
		t.Errorf("Expected 2 items, got %d", len(summary))
	}

	snp, ok := summary["SNP"]
	if !ok {
		t.Fatal("Expected SNP in summary")
	}

	if snp.Symbol != "^GSPC" {
		t.Errorf("Expected symbol '^GSPC', got '%s'", snp.Symbol)
	}

	if snp.ShortName != "S&P 500" {
		t.Errorf("Expected shortName 'S&P 500', got '%s'", snp.ShortName)
	}

	if snp.RegularMarketPrice != 4500.50 {
		t.Errorf("Expected price 4500.50, got %f", snp.RegularMarketPrice)
	}

	if snp.MarketState != "REGULAR" {
		t.Errorf("Expected marketState 'REGULAR', got '%s'", snp.MarketState)
	}
}

func TestParseSummaryEmpty(t *testing.T) {
	m, err := New("us_market")
	if err != nil {
		t.Fatalf("Failed to create Market: %v", err)
	}
	defer m.Close()

	// Empty results
	summary := m.parseSummary(nil)
	if len(summary) != 0 {
		t.Errorf("Expected empty summary, got %d items", len(summary))
	}

	// Results without exchange
	results := []map[string]interface{}{
		{"symbol": "^GSPC", "shortName": "S&P 500"},
	}
	summary = m.parseSummary(results)
	if len(summary) != 0 {
		t.Errorf("Expected empty summary for missing exchange, got %d items", len(summary))
	}
}

// Integration tests (require network access)
// Run with: go test -v -run Integration

func TestStatusIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	m, err := New("us_market")
	if err != nil {
		t.Fatalf("Failed to create Market: %v", err)
	}
	defer m.Close()

	status, err := m.Status()
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}

	if status == nil {
		t.Fatal("Status should not be nil")
	}

	t.Logf("Market ID: %s", status.ID)
	if status.Open != nil {
		t.Logf("Open: %s", status.Open.Format("2006-01-02 15:04:05 MST"))
	}
	if status.Close != nil {
		t.Logf("Close: %s", status.Close.Format("2006-01-02 15:04:05 MST"))
	}
	if status.Timezone != nil {
		t.Logf("Timezone: %s (GMT%+d)", status.Timezone.Short, status.Timezone.GMTOffset/3600000)
	}
}

func TestSummaryIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	m, err := New("us_market")
	if err != nil {
		t.Fatalf("Failed to create Market: %v", err)
	}
	defer m.Close()

	summary, err := m.Summary()
	if err != nil {
		t.Fatalf("Summary() failed: %v", err)
	}

	if len(summary) == 0 {
		t.Error("Expected at least one item in summary")
	}

	// Print first few indices
	count := 0
	for exchange, item := range summary {
		if count >= 5 {
			break
		}
		t.Logf("%s (%s): %.2f (%.2f%%)",
			item.ShortName, exchange,
			item.RegularMarketPrice,
			item.RegularMarketChangePercent)
		count++
	}
}

func TestIsOpenIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	m, err := New("us_market")
	if err != nil {
		t.Fatalf("Failed to create Market: %v", err)
	}
	defer m.Close()

	isOpen, err := m.IsOpen()
	if err != nil {
		t.Fatalf("IsOpen() failed: %v", err)
	}

	t.Logf("US Market is open: %v", isOpen)
}

func TestMultipleMarketsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	markets := []models.PredefinedMarket{
		models.MarketUS,
		models.MarketGB,
		models.MarketJP,
	}

	for _, market := range markets {
		m, err := NewWithPredefined(market)
		if err != nil {
			t.Errorf("Failed to create Market for %s: %v", market, err)
			continue
		}

		summary, err := m.Summary()
		if err != nil {
			t.Errorf("Summary() failed for %s: %v", market, err)
			m.Close()
			continue
		}

		t.Logf("%s: %d indices", market, len(summary))
		m.Close()
	}
}

func TestCacheIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	m, err := New("us_market")
	if err != nil {
		t.Fatalf("Failed to create Market: %v", err)
	}
	defer m.Close()

	// First call - fetch from API
	summary1, err := m.Summary()
	if err != nil {
		t.Fatalf("First Summary() failed: %v", err)
	}

	// Second call - should use cache
	summary2, err := m.Summary()
	if err != nil {
		t.Fatalf("Second Summary() failed: %v", err)
	}

	if len(summary1) != len(summary2) {
		t.Errorf("Cache should return same results: %d vs %d", len(summary1), len(summary2))
	}

	// Clear cache and fetch again
	m.ClearCache()

	summary3, err := m.Summary()
	if err != nil {
		t.Fatalf("Third Summary() failed: %v", err)
	}

	t.Logf("Results: first=%d, cached=%d, after_clear=%d", len(summary1), len(summary2), len(summary3))
}
