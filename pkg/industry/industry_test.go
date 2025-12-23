package industry

import (
	"testing"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func TestNew(t *testing.T) {
	i, err := New("semiconductors")
	if err != nil {
		t.Fatalf("Failed to create Industry: %v", err)
	}
	defer i.Close()

	if i == nil {
		t.Fatal("Industry should not be nil")
	}

	if i.key != "semiconductors" {
		t.Errorf("Expected key 'semiconductors', got '%s'", i.key)
	}

	if i.ownsClient != true {
		t.Error("ownsClient should be true when no custom client is provided")
	}
}

func TestNewWithEmptyIndustry(t *testing.T) {
	_, err := New("")
	if err == nil {
		t.Error("Expected error for empty industry key")
	}
}

func TestNewWithPredefined(t *testing.T) {
	i, err := NewWithPredefined(models.IndustrySemiconductors)
	if err != nil {
		t.Fatalf("Failed to create Industry: %v", err)
	}
	defer i.Close()

	if i.key != "semiconductors" {
		t.Errorf("Expected key 'semiconductors', got '%s'", i.key)
	}
}

func TestNewWithClient(t *testing.T) {
	i1, err := New("semiconductors")
	if err != nil {
		t.Fatalf("Failed to create first Industry: %v", err)
	}

	i2, err := New("biotechnology", WithClient(i1.client))
	if err != nil {
		t.Fatalf("Failed to create second Industry: %v", err)
	}

	if i2.ownsClient != false {
		t.Error("ownsClient should be false when custom client is provided")
	}

	if i2.client != i1.client {
		t.Error("Client should be the same instance")
	}

	i1.Close()
}

func TestKey(t *testing.T) {
	i, err := New("software-infrastructure")
	if err != nil {
		t.Fatalf("Failed to create Industry: %v", err)
	}
	defer i.Close()

	if i.Key() != "software-infrastructure" {
		t.Errorf("Expected key 'software-infrastructure', got '%s'", i.Key())
	}
}

func TestPredefinedIndustries(t *testing.T) {
	tests := []struct {
		industry models.PredefinedIndustry
		expected string
	}{
		{models.IndustrySemiconductors, "semiconductors"},
		{models.IndustrySoftwareInfrastructure, "software-infrastructure"},
		{models.IndustrySoftwareApplication, "software-application"},
		{models.IndustryConsumerElectronics, "consumer-electronics"},
		{models.IndustryBiotechnology, "biotechnology"},
		{models.IndustryBanksDiversified, "banks-diversified"},
		{models.IndustryOilGasIntegrated, "oil-gas-integrated"},
		{models.IndustryAutoManufacturers, "auto-manufacturers"},
		{models.IndustryAerospaceDefense, "aerospace-defense"},
	}

	for _, tt := range tests {
		if string(tt.industry) != tt.expected {
			t.Errorf("PredefinedIndustry %v: expected '%s', got '%s'", tt.industry, tt.expected, string(tt.industry))
		}
	}
}

func TestAllIndustries(t *testing.T) {
	industries := models.AllIndustries()
	if len(industries) < 10 {
		t.Errorf("Expected at least 10 predefined industries, got %d", len(industries))
	}
}

func TestClearCache(t *testing.T) {
	i, err := New("semiconductors")
	if err != nil {
		t.Fatalf("Failed to create Industry: %v", err)
	}
	defer i.Close()

	// Add something to cache manually
	i.mu.Lock()
	i.dataCache = &models.IndustryData{Key: "test", Name: "Test"}
	i.mu.Unlock()

	// Verify cache has data
	i.mu.RLock()
	hasData := i.dataCache != nil
	i.mu.RUnlock()
	if !hasData {
		t.Error("Cache should have data")
	}

	// Clear cache
	i.ClearCache()

	// Verify cache is empty
	i.mu.RLock()
	isEmpty := i.dataCache == nil
	i.mu.RUnlock()
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

	i, err := NewWithPredefined(models.IndustrySemiconductors)
	if err != nil {
		t.Fatalf("Failed to create Industry: %v", err)
	}
	defer i.Close()

	data, err := i.Data()
	if err != nil {
		t.Fatalf("Data() failed: %v", err)
	}

	if data == nil {
		t.Fatal("Data should not be nil")
	}

	t.Logf("Industry Key: %s", data.Key)
	t.Logf("Industry Name: %s", data.Name)
	t.Logf("Symbol: %s", data.Symbol)
	t.Logf("Sector Key: %s", data.SectorKey)
	t.Logf("Sector Name: %s", data.SectorName)
	t.Logf("Companies Count: %d", data.Overview.CompaniesCount)
	t.Logf("Top Companies: %d", len(data.TopCompanies))
	t.Logf("Top Performing Companies: %d", len(data.TopPerformingCompanies))
	t.Logf("Top Growth Companies: %d", len(data.TopGrowthCompanies))
}

func TestOverviewIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	i, err := NewWithPredefined(models.IndustrySemiconductors)
	if err != nil {
		t.Fatalf("Failed to create Industry: %v", err)
	}
	defer i.Close()

	overview, err := i.Overview()
	if err != nil {
		t.Fatalf("Overview() failed: %v", err)
	}

	if overview.CompaniesCount == 0 {
		t.Error("Expected companies count > 0")
	}

	t.Logf("Companies: %d", overview.CompaniesCount)
	t.Logf("Market Weight: %.4f", overview.MarketWeight)
	if overview.Description != "" {
		t.Logf("Description: %.100s...", overview.Description)
	}
}

func TestSectorInfoIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	i, err := NewWithPredefined(models.IndustrySemiconductors)
	if err != nil {
		t.Fatalf("Failed to create Industry: %v", err)
	}
	defer i.Close()

	sectorKey, err := i.SectorKey()
	if err != nil {
		t.Fatalf("SectorKey() failed: %v", err)
	}

	sectorName, err := i.SectorName()
	if err != nil {
		t.Fatalf("SectorName() failed: %v", err)
	}

	if sectorKey == "" {
		t.Error("Expected non-empty sector key")
	}

	if sectorName == "" {
		t.Error("Expected non-empty sector name")
	}

	t.Logf("Sector Key: %s", sectorKey)
	t.Logf("Sector Name: %s", sectorName)
}

func TestTopCompaniesIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	i, err := NewWithPredefined(models.IndustrySemiconductors)
	if err != nil {
		t.Fatalf("Failed to create Industry: %v", err)
	}
	defer i.Close()

	companies, err := i.TopCompanies()
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

func TestTopPerformingCompaniesIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	i, err := NewWithPredefined(models.IndustrySemiconductors)
	if err != nil {
		t.Fatalf("Failed to create Industry: %v", err)
	}
	defer i.Close()

	companies, err := i.TopPerformingCompanies()
	if err != nil {
		t.Fatalf("TopPerformingCompanies() failed: %v", err)
	}

	t.Logf("Top Performing Companies count: %d", len(companies))
	for _, c := range companies {
		t.Logf("%s: %s (YTD: %.2f%%, Last: $%.2f, Target: $%.2f)",
			c.Symbol, c.Name, c.YTDReturn*100, c.LastPrice, c.TargetPrice)
	}
}

func TestTopGrowthCompaniesIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	i, err := NewWithPredefined(models.IndustrySemiconductors)
	if err != nil {
		t.Fatalf("Failed to create Industry: %v", err)
	}
	defer i.Close()

	companies, err := i.TopGrowthCompanies()
	if err != nil {
		t.Fatalf("TopGrowthCompanies() failed: %v", err)
	}

	t.Logf("Top Growth Companies count: %d", len(companies))
	for _, c := range companies {
		t.Logf("%s: %s (YTD: %.2f%%, Growth Est: %.2f%%)",
			c.Symbol, c.Name, c.YTDReturn*100, c.GrowthEstimate*100)
	}
}

func TestMultipleIndustriesIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	industries := []models.PredefinedIndustry{
		models.IndustrySemiconductors,
		models.IndustryBiotechnology,
		models.IndustryBanksDiversified,
	}

	for _, ind := range industries {
		i, err := NewWithPredefined(ind)
		if err != nil {
			t.Errorf("Failed to create Industry for %s: %v", ind, err)
			continue
		}

		name, err := i.Name()
		if err != nil {
			t.Errorf("Name() failed for %s: %v", ind, err)
			i.Close()
			continue
		}

		sectorName, err := i.SectorName()
		if err != nil {
			t.Errorf("SectorName() failed for %s: %v", ind, err)
			i.Close()
			continue
		}

		overview, err := i.Overview()
		if err != nil {
			t.Errorf("Overview() failed for %s: %v", ind, err)
			i.Close()
			continue
		}

		t.Logf("%s: %s in %s - %d companies",
			ind, name, sectorName, overview.CompaniesCount)
		i.Close()
	}
}

func TestCacheIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	i, err := NewWithPredefined(models.IndustrySemiconductors)
	if err != nil {
		t.Fatalf("Failed to create Industry: %v", err)
	}
	defer i.Close()

	// First call - fetch from API
	data1, err := i.Data()
	if err != nil {
		t.Fatalf("First Data() failed: %v", err)
	}

	// Second call - should use cache
	data2, err := i.Data()
	if err != nil {
		t.Fatalf("Second Data() failed: %v", err)
	}

	if data1.Name != data2.Name {
		t.Errorf("Cache should return same results: %s vs %s", data1.Name, data2.Name)
	}

	// Clear cache and fetch again
	i.ClearCache()

	data3, err := i.Data()
	if err != nil {
		t.Fatalf("Third Data() failed: %v", err)
	}

	t.Logf("Results: first=%s, cached=%s, after_clear=%s", data1.Name, data2.Name, data3.Name)
}
