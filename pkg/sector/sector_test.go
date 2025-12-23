package sector

import (
	"testing"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func TestNew(t *testing.T) {
	s, err := New("technology")
	if err != nil {
		t.Fatalf("Failed to create Sector: %v", err)
	}
	defer s.Close()

	if s == nil {
		t.Fatal("Sector should not be nil")
	}

	if s.key != "technology" {
		t.Errorf("Expected key 'technology', got '%s'", s.key)
	}

	if s.ownsClient != true {
		t.Error("ownsClient should be true when no custom client is provided")
	}
}

func TestNewWithEmptySector(t *testing.T) {
	_, err := New("")
	if err == nil {
		t.Error("Expected error for empty sector key")
	}
}

func TestNewWithPredefined(t *testing.T) {
	s, err := NewWithPredefined(models.SectorTechnology)
	if err != nil {
		t.Fatalf("Failed to create Sector: %v", err)
	}
	defer s.Close()

	if s.key != "technology" {
		t.Errorf("Expected key 'technology', got '%s'", s.key)
	}
}

func TestNewWithClient(t *testing.T) {
	s1, err := New("technology")
	if err != nil {
		t.Fatalf("Failed to create first Sector: %v", err)
	}

	s2, err := New("healthcare", WithClient(s1.client))
	if err != nil {
		t.Fatalf("Failed to create second Sector: %v", err)
	}

	if s2.ownsClient != false {
		t.Error("ownsClient should be false when custom client is provided")
	}

	if s2.client != s1.client {
		t.Error("Client should be the same instance")
	}

	s1.Close()
}

func TestKey(t *testing.T) {
	s, err := New("financial-services")
	if err != nil {
		t.Fatalf("Failed to create Sector: %v", err)
	}
	defer s.Close()

	if s.Key() != "financial-services" {
		t.Errorf("Expected key 'financial-services', got '%s'", s.Key())
	}
}

func TestPredefinedSectors(t *testing.T) {
	tests := []struct {
		sector   models.PredefinedSector
		expected string
	}{
		{models.SectorBasicMaterials, "basic-materials"},
		{models.SectorCommunicationServices, "communication-services"},
		{models.SectorConsumerCyclical, "consumer-cyclical"},
		{models.SectorConsumerDefensive, "consumer-defensive"},
		{models.SectorEnergy, "energy"},
		{models.SectorFinancialServices, "financial-services"},
		{models.SectorHealthcare, "healthcare"},
		{models.SectorIndustrials, "industrials"},
		{models.SectorRealEstate, "real-estate"},
		{models.SectorTechnology, "technology"},
		{models.SectorUtilities, "utilities"},
	}

	for _, tt := range tests {
		if string(tt.sector) != tt.expected {
			t.Errorf("PredefinedSector %v: expected '%s', got '%s'", tt.sector, tt.expected, string(tt.sector))
		}
	}
}

func TestAllSectors(t *testing.T) {
	sectors := models.AllSectors()
	if len(sectors) != 11 {
		t.Errorf("Expected 11 predefined sectors, got %d", len(sectors))
	}
}

func TestClearCache(t *testing.T) {
	s, err := New("technology")
	if err != nil {
		t.Fatalf("Failed to create Sector: %v", err)
	}
	defer s.Close()

	// Add something to cache manually
	s.mu.Lock()
	s.dataCache = &models.SectorData{Key: "test", Name: "Test"}
	s.mu.Unlock()

	// Verify cache has data
	s.mu.RLock()
	hasData := s.dataCache != nil
	s.mu.RUnlock()
	if !hasData {
		t.Error("Cache should have data")
	}

	// Clear cache
	s.ClearCache()

	// Verify cache is empty
	s.mu.RLock()
	isEmpty := s.dataCache == nil
	s.mu.RUnlock()
	if !isEmpty {
		t.Error("Cache should be empty after ClearCache")
	}
}

func TestHelperFunctions(t *testing.T) {
	m := map[string]interface{}{
		"str":   "hello",
		"int":   42,
		"float": 3.14,
		"int64": int64(1234567890),
		"nested": map[string]interface{}{
			"raw": 99.99,
			"fmt": "99.99",
		},
	}

	// Test getString
	if got := getString(m, "str"); got != "hello" {
		t.Errorf("getString expected 'hello', got '%s'", got)
	}

	if got := getString(m, "missing"); got != "" {
		t.Errorf("getString for missing key expected '', got '%s'", got)
	}

	// Test getInt
	if got := getInt(m, "int"); got != 42 {
		t.Errorf("getInt expected 42, got %d", got)
	}

	if got := getInt(m, "missing"); got != 0 {
		t.Errorf("getInt for missing key expected 0, got %d", got)
	}

	// Test getFloat
	if got := getFloat(m, "float"); got != 3.14 {
		t.Errorf("getFloat expected 3.14, got %f", got)
	}

	if got := getFloat(m, "int"); got != 42.0 {
		t.Errorf("getFloat for int expected 42.0, got %f", got)
	}

	// Test getInt64
	if got := getInt64(m, "int64"); got != 1234567890 {
		t.Errorf("getInt64 expected 1234567890, got %d", got)
	}

	if got := getInt64(m, "float"); got != 3 {
		t.Errorf("getInt64 for float expected 3, got %d", got)
	}

	// Test getRawFloat
	if got := getRawFloat(m, "nested"); got != 99.99 {
		t.Errorf("getRawFloat expected 99.99, got %f", got)
	}

	if got := getRawFloat(m, "float"); got != 3.14 {
		t.Errorf("getRawFloat for simple float expected 3.14, got %f", got)
	}
}

// Integration tests (require network access)
// Run with: go test -v -run Integration

func TestDataIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	s, err := NewWithPredefined(models.SectorTechnology)
	if err != nil {
		t.Fatalf("Failed to create Sector: %v", err)
	}
	defer s.Close()

	data, err := s.Data()
	if err != nil {
		t.Fatalf("Data() failed: %v", err)
	}

	if data == nil {
		t.Fatal("Data should not be nil")
	}

	t.Logf("Sector Key: %s", data.Key)
	t.Logf("Sector Name: %s", data.Name)
	t.Logf("Symbol: %s", data.Symbol)
	t.Logf("Companies Count: %d", data.Overview.CompaniesCount)
	t.Logf("Industries Count: %d", data.Overview.IndustriesCount)
	t.Logf("Top Companies: %d", len(data.TopCompanies))
	t.Logf("Industries: %d", len(data.Industries))
	t.Logf("Top ETFs: %d", len(data.TopETFs))
	t.Logf("Top Mutual Funds: %d", len(data.TopMutualFunds))
}

func TestOverviewIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	s, err := NewWithPredefined(models.SectorTechnology)
	if err != nil {
		t.Fatalf("Failed to create Sector: %v", err)
	}
	defer s.Close()

	overview, err := s.Overview()
	if err != nil {
		t.Fatalf("Overview() failed: %v", err)
	}

	if overview.CompaniesCount == 0 {
		t.Error("Expected companies count > 0")
	}

	t.Logf("Companies: %d", overview.CompaniesCount)
	t.Logf("Industries: %d", overview.IndustriesCount)
	t.Logf("Market Weight: %.4f", overview.MarketWeight)
	t.Logf("Description: %.100s...", overview.Description)
}

func TestTopCompaniesIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	s, err := NewWithPredefined(models.SectorTechnology)
	if err != nil {
		t.Fatalf("Failed to create Sector: %v", err)
	}
	defer s.Close()

	companies, err := s.TopCompanies()
	if err != nil {
		t.Fatalf("TopCompanies() failed: %v", err)
	}

	if len(companies) == 0 {
		t.Error("Expected at least one top company")
	}

	// Print first 5 companies
	count := 0
	for _, c := range companies {
		if count >= 5 {
			break
		}
		t.Logf("%s: %s (Weight: %.4f)", c.Symbol, c.Name, c.MarketWeight)
		count++
	}
}

func TestIndustriesIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	s, err := NewWithPredefined(models.SectorTechnology)
	if err != nil {
		t.Fatalf("Failed to create Sector: %v", err)
	}
	defer s.Close()

	industries, err := s.Industries()
	if err != nil {
		t.Fatalf("Industries() failed: %v", err)
	}

	if len(industries) == 0 {
		t.Error("Expected at least one industry")
	}

	for _, i := range industries {
		t.Logf("%s: %s (Weight: %.4f)", i.Key, i.Name, i.MarketWeight)
	}
}

func TestTopETFsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	s, err := NewWithPredefined(models.SectorTechnology)
	if err != nil {
		t.Fatalf("Failed to create Sector: %v", err)
	}
	defer s.Close()

	etfs, err := s.TopETFs()
	if err != nil {
		t.Fatalf("TopETFs() failed: %v", err)
	}

	t.Logf("Top ETFs count: %d", len(etfs))
	for symbol, name := range etfs {
		t.Logf("%s: %s", symbol, name)
	}
}

func TestTopMutualFundsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	s, err := NewWithPredefined(models.SectorTechnology)
	if err != nil {
		t.Fatalf("Failed to create Sector: %v", err)
	}
	defer s.Close()

	funds, err := s.TopMutualFunds()
	if err != nil {
		t.Fatalf("TopMutualFunds() failed: %v", err)
	}

	t.Logf("Top Mutual Funds count: %d", len(funds))
	for symbol, name := range funds {
		t.Logf("%s: %s", symbol, name)
	}
}

func TestMultipleSectorsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	sectors := []models.PredefinedSector{
		models.SectorTechnology,
		models.SectorHealthcare,
		models.SectorFinancialServices,
	}

	for _, sector := range sectors {
		s, err := NewWithPredefined(sector)
		if err != nil {
			t.Errorf("Failed to create Sector for %s: %v", sector, err)
			continue
		}

		name, err := s.Name()
		if err != nil {
			t.Errorf("Name() failed for %s: %v", sector, err)
			s.Close()
			continue
		}

		overview, err := s.Overview()
		if err != nil {
			t.Errorf("Overview() failed for %s: %v", sector, err)
			s.Close()
			continue
		}

		t.Logf("%s: %s - %d companies, %d industries",
			sector, name, overview.CompaniesCount, overview.IndustriesCount)
		s.Close()
	}
}

func TestCacheIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	s, err := NewWithPredefined(models.SectorTechnology)
	if err != nil {
		t.Fatalf("Failed to create Sector: %v", err)
	}
	defer s.Close()

	// First call - fetch from API
	data1, err := s.Data()
	if err != nil {
		t.Fatalf("First Data() failed: %v", err)
	}

	// Second call - should use cache
	data2, err := s.Data()
	if err != nil {
		t.Fatalf("Second Data() failed: %v", err)
	}

	if data1.Name != data2.Name {
		t.Errorf("Cache should return same results: %s vs %s", data1.Name, data2.Name)
	}

	// Clear cache and fetch again
	s.ClearCache()

	data3, err := s.Data()
	if err != nil {
		t.Fatalf("Third Data() failed: %v", err)
	}

	t.Logf("Results: first=%s, cached=%s, after_clear=%s", data1.Name, data2.Name, data3.Name)
}
