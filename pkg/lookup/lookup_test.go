package lookup

import (
	"testing"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func TestNew(t *testing.T) {
	l, err := New("AAPL")
	if err != nil {
		t.Fatalf("Failed to create Lookup: %v", err)
	}
	defer l.Close()

	if l == nil {
		t.Fatal("Lookup should not be nil")
	}

	if l.query != "AAPL" {
		t.Errorf("Expected query 'AAPL', got '%s'", l.query)
	}

	if l.ownsClient != true {
		t.Error("ownsClient should be true when no custom client is provided")
	}
}

func TestNewWithEmptyQuery(t *testing.T) {
	_, err := New("")
	if err == nil {
		t.Error("Expected error for empty query")
	}
}

func TestNewWithClient(t *testing.T) {
	// Create a Lookup instance to get a client
	l1, err := New("AAPL")
	if err != nil {
		t.Fatalf("Failed to create first Lookup: %v", err)
	}

	// Create another Lookup with the same client
	l2, err := New("MSFT", WithClient(l1.client))
	if err != nil {
		t.Fatalf("Failed to create second Lookup: %v", err)
	}

	if l2.ownsClient != false {
		t.Error("ownsClient should be false when custom client is provided")
	}

	if l2.client != l1.client {
		t.Error("Client should be the same instance")
	}

	// Only close l1 since it owns the client
	l1.Close()
}

func TestQuery(t *testing.T) {
	l, err := New("Apple Inc")
	if err != nil {
		t.Fatalf("Failed to create Lookup: %v", err)
	}
	defer l.Close()

	if l.Query() != "Apple Inc" {
		t.Errorf("Expected query 'Apple Inc', got '%s'", l.Query())
	}
}

func TestDefaultLookupParams(t *testing.T) {
	params := models.DefaultLookupParams()

	if params.Type != models.LookupTypeAll {
		t.Errorf("Expected Type to be 'all', got '%s'", params.Type)
	}

	if params.Count != 25 {
		t.Errorf("Expected Count to be 25, got %d", params.Count)
	}

	if params.Start != 0 {
		t.Errorf("Expected Start to be 0, got %d", params.Start)
	}

	if params.FetchPricingData != true {
		t.Error("Expected FetchPricingData to be true")
	}
}

func TestLookupTypes(t *testing.T) {
	tests := []struct {
		lookupType models.LookupType
		expected   string
	}{
		{models.LookupTypeAll, "all"},
		{models.LookupTypeEquity, "equity"},
		{models.LookupTypeMutualFund, "mutualfund"},
		{models.LookupTypeETF, "etf"},
		{models.LookupTypeIndex, "index"},
		{models.LookupTypeFuture, "future"},
		{models.LookupTypeCurrency, "currency"},
		{models.LookupTypeCryptocurrency, "cryptocurrency"},
	}

	for _, tt := range tests {
		if string(tt.lookupType) != tt.expected {
			t.Errorf("LookupType %v: expected '%s', got '%s'", tt.lookupType, tt.expected, string(tt.lookupType))
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

	if got := getString(m, "int"); got != "" {
		t.Errorf("getString for int expected '', got '%s'", got)
	}

	// Test getFloat
	if got := getFloat(m, "float"); got != 3.14 {
		t.Errorf("getFloat expected 3.14, got %f", got)
	}

	if got := getFloat(m, "int"); got != 42.0 {
		t.Errorf("getFloat for int expected 42.0, got %f", got)
	}

	if got := getFloat(m, "int64"); got != 1234567890.0 {
		t.Errorf("getFloat for int64 expected 1234567890.0, got %f", got)
	}

	if got := getFloat(m, "missing"); got != 0 {
		t.Errorf("getFloat for missing key expected 0, got %f", got)
	}

	// Test getInt64
	if got := getInt64(m, "float"); got != 3 {
		t.Errorf("getInt64 for float expected 3, got %d", got)
	}

	if got := getInt64(m, "int"); got != 42 {
		t.Errorf("getInt64 for int expected 42, got %d", got)
	}

	if got := getInt64(m, "int64"); got != 1234567890 {
		t.Errorf("getInt64 for int64 expected 1234567890, got %d", got)
	}

	if got := getInt64(m, "missing"); got != 0 {
		t.Errorf("getInt64 for missing key expected 0, got %d", got)
	}
}

func TestClearCache(t *testing.T) {
	l, err := New("AAPL")
	if err != nil {
		t.Fatalf("Failed to create Lookup: %v", err)
	}
	defer l.Close()

	// Add something to cache manually
	l.mu.Lock()
	l.cache["test:10"] = &models.LookupResult{Count: 1}
	l.mu.Unlock()

	// Verify cache has data
	l.mu.RLock()
	if len(l.cache) == 0 {
		t.Error("Cache should have data")
	}
	l.mu.RUnlock()

	// Clear cache
	l.ClearCache()

	// Verify cache is empty
	l.mu.RLock()
	if len(l.cache) != 0 {
		t.Error("Cache should be empty after ClearCache")
	}
	l.mu.RUnlock()
}

func TestParseResponse(t *testing.T) {
	l, err := New("AAPL")
	if err != nil {
		t.Fatalf("Failed to create Lookup: %v", err)
	}
	defer l.Close()

	// Test with empty response
	emptyResp := &models.LookupResponse{}
	result := l.parseResponse(emptyResp)

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if len(result.Documents) != 0 {
		t.Error("Documents should be empty for empty response")
	}

	// Test with valid response
	validResp := &models.LookupResponse{}
	validResp.Finance.Result = []struct {
		Documents []map[string]interface{} `json:"documents"`
		Count     int                      `json:"count,omitempty"`
		Start     int                      `json:"start,omitempty"`
		Total     int                      `json:"total,omitempty"`
	}{
		{
			Documents: []map[string]interface{}{
				{
					"symbol":             "AAPL",
					"name":               "Apple Inc.",
					"exchange":           "NMS",
					"quoteType":          "EQUITY",
					"regularMarketPrice": 150.25,
					"marketCap":          float64(2500000000000),
				},
			},
			Count: 1,
		},
	}

	result = l.parseResponse(validResp)

	if result.Count != 1 {
		t.Errorf("Expected count 1, got %d", result.Count)
	}

	if len(result.Documents) != 1 {
		t.Fatalf("Expected 1 document, got %d", len(result.Documents))
	}

	doc := result.Documents[0]
	if doc.Symbol != "AAPL" {
		t.Errorf("Expected symbol 'AAPL', got '%s'", doc.Symbol)
	}

	if doc.Name != "Apple Inc." {
		t.Errorf("Expected name 'Apple Inc.', got '%s'", doc.Name)
	}

	if doc.Exchange != "NMS" {
		t.Errorf("Expected exchange 'NMS', got '%s'", doc.Exchange)
	}

	if doc.QuoteType != "EQUITY" {
		t.Errorf("Expected quoteType 'EQUITY', got '%s'", doc.QuoteType)
	}

	if doc.RegularMarketPrice != 150.25 {
		t.Errorf("Expected price 150.25, got %f", doc.RegularMarketPrice)
	}

	if doc.MarketCap != 2500000000000 {
		t.Errorf("Expected marketCap 2500000000000, got %d", doc.MarketCap)
	}
}

func TestParseResponseWithShortName(t *testing.T) {
	l, err := New("AAPL")
	if err != nil {
		t.Fatalf("Failed to create Lookup: %v", err)
	}
	defer l.Close()

	// Test that shortName is used when name is empty
	validResp := &models.LookupResponse{}
	validResp.Finance.Result = []struct {
		Documents []map[string]interface{} `json:"documents"`
		Count     int                      `json:"count,omitempty"`
		Start     int                      `json:"start,omitempty"`
		Total     int                      `json:"total,omitempty"`
	}{
		{
			Documents: []map[string]interface{}{
				{
					"symbol":    "AAPL",
					"shortName": "Apple Inc",
					// name is missing
				},
			},
			Count: 1,
		},
	}

	result := l.parseResponse(validResp)

	if len(result.Documents) != 1 {
		t.Fatalf("Expected 1 document, got %d", len(result.Documents))
	}

	doc := result.Documents[0]
	if doc.Name != "Apple Inc" {
		t.Errorf("Expected name to be 'Apple Inc' from shortName, got '%s'", doc.Name)
	}
}

// Integration tests (require network access)
// Run with: go test -v -run Integration

func TestLookupAllIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	l, err := New("Apple")
	if err != nil {
		t.Fatalf("Failed to create Lookup: %v", err)
	}
	defer l.Close()

	results, err := l.All(10)
	if err != nil {
		t.Fatalf("All() failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected at least one result")
	}

	// Check if AAPL is in the results
	found := false
	for _, doc := range results {
		if doc.Symbol == "AAPL" {
			found = true
			t.Logf("Found AAPL: %s, Price: %.2f", doc.Name, doc.RegularMarketPrice)
			break
		}
	}
	if !found {
		t.Log("AAPL not found in results, but other results were returned")
	}

	// Print first few results for debugging
	for i, doc := range results {
		if i >= 3 {
			break
		}
		t.Logf("Result %d: %s - %s (%s)", i+1, doc.Symbol, doc.Name, doc.QuoteType)
	}
}

func TestLookupStockIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	l, err := New("Microsoft")
	if err != nil {
		t.Fatalf("Failed to create Lookup: %v", err)
	}
	defer l.Close()

	results, err := l.Stock(5)
	if err != nil {
		t.Fatalf("Stock() failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected at least one stock result")
	}

	// Verify all results are equity type
	for _, doc := range results {
		if doc.QuoteType != "" && doc.QuoteType != "EQUITY" {
			t.Logf("Warning: Got non-EQUITY result: %s (%s)", doc.Symbol, doc.QuoteType)
		}
		t.Logf("Stock: %s - %s, Price: %.2f", doc.Symbol, doc.Name, doc.RegularMarketPrice)
	}
}

func TestLookupETFIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	l, err := New("SPY")
	if err != nil {
		t.Fatalf("Failed to create Lookup: %v", err)
	}
	defer l.Close()

	results, err := l.ETF(5)
	if err != nil {
		t.Fatalf("ETF() failed: %v", err)
	}

	t.Logf("Found %d ETF results for 'SPY'", len(results))
	for _, doc := range results {
		t.Logf("ETF: %s - %s", doc.Symbol, doc.Name)
	}
}

func TestLookupCryptocurrencyIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	l, err := New("Bitcoin")
	if err != nil {
		t.Fatalf("Failed to create Lookup: %v", err)
	}
	defer l.Close()

	results, err := l.Cryptocurrency(5)
	if err != nil {
		t.Fatalf("Cryptocurrency() failed: %v", err)
	}

	t.Logf("Found %d cryptocurrency results for 'Bitcoin'", len(results))
	for _, doc := range results {
		t.Logf("Crypto: %s - %s, Price: %.2f", doc.Symbol, doc.Name, doc.RegularMarketPrice)
	}
}

func TestLookupCacheIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	l, err := New("AAPL")
	if err != nil {
		t.Fatalf("Failed to create Lookup: %v", err)
	}
	defer l.Close()

	// First call - should fetch from API
	results1, err := l.All(5)
	if err != nil {
		t.Fatalf("First All() failed: %v", err)
	}

	// Second call - should use cache
	results2, err := l.All(5)
	if err != nil {
		t.Fatalf("Second All() failed: %v", err)
	}

	// Results should be the same
	if len(results1) != len(results2) {
		t.Errorf("Cache should return same results: got %d vs %d", len(results1), len(results2))
	}

	// Clear cache and fetch again
	l.ClearCache()

	results3, err := l.All(5)
	if err != nil {
		t.Fatalf("Third All() failed: %v", err)
	}

	t.Logf("Results: first=%d, cached=%d, after_clear=%d", len(results1), len(results2), len(results3))
}

func TestLookupDefaultCountIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	l, err := New("tech")
	if err != nil {
		t.Fatalf("Failed to create Lookup: %v", err)
	}
	defer l.Close()

	// Test with count <= 0 (should use default 25)
	results, err := l.All(0)
	if err != nil {
		t.Fatalf("All(0) failed: %v", err)
	}

	t.Logf("All(0) returned %d results (default should be 25)", len(results))
}
