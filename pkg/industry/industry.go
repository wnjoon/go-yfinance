package industry

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"

	"github.com/wnjoon/go-yfinance/internal/endpoints"
	"github.com/wnjoon/go-yfinance/pkg/client"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

// Industry provides access to financial industry data from Yahoo Finance.
//
// Industry allows retrieving industry-related information such as
// overview, top companies, sector information, and performing/growth companies.
type Industry struct {
	key string

	client     *client.Client
	auth       *client.AuthManager
	ownsClient bool

	// Cached data
	mu        sync.RWMutex
	dataCache *models.IndustryData
}

// Option is a function that configures an Industry instance.
type Option func(*Industry)

// WithClient sets a custom HTTP client for the Industry instance.
func WithClient(c *client.Client) Option {
	return func(i *Industry) {
		i.client = c
		i.ownsClient = false
	}
}

// New creates a new Industry instance for the given industry key.
//
// Industry keys are lowercase with hyphens, e.g., "semiconductors", "software-infrastructure".
// Use predefined constants from models package for convenience.
//
// Example:
//
//	i, err := industry.New("semiconductors")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer i.Close()
//
//	overview, err := i.Overview()
//	fmt.Printf("Industry has %d companies\n", overview.CompaniesCount)
func New(key string, opts ...Option) (*Industry, error) {
	if key == "" {
		return nil, fmt.Errorf("industry key cannot be empty")
	}

	i := &Industry{
		key:        key,
		ownsClient: true,
	}

	for _, opt := range opts {
		opt(i)
	}

	if i.client == nil {
		c, err := client.New()
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}
		i.client = c
	}

	i.auth = client.NewAuthManager(i.client)

	return i, nil
}

// NewWithPredefined creates a new Industry instance using a predefined industry constant.
//
// Example:
//
//	i, err := industry.NewWithPredefined(models.IndustrySemiconductors)
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewWithPredefined(ind models.PredefinedIndustry, opts ...Option) (*Industry, error) {
	return New(string(ind), opts...)
}

// Close releases resources used by the Industry instance.
func (i *Industry) Close() {
	if i.ownsClient && i.client != nil {
		i.client.Close()
	}
}

// Key returns the industry key.
func (i *Industry) Key() string {
	return i.key
}

// fetchData fetches industry data from Yahoo Finance API.
func (i *Industry) fetchData() error {
	i.mu.RLock()
	if i.dataCache != nil {
		i.mu.RUnlock()
		return nil
	}
	i.mu.RUnlock()

	queryURL := fmt.Sprintf("%s/%s", endpoints.IndustryURL, i.key)

	params := url.Values{}
	params.Set("formatted", "true")
	params.Set("withReturns", "true")
	params.Set("lang", "en-US")
	params.Set("region", "US")

	// Add crumb authentication
	params, err := i.auth.AddCrumbToParams(params)
	if err != nil {
		return fmt.Errorf("failed to add crumb: %w", err)
	}

	resp, err := i.client.Get(queryURL, params)
	if err != nil {
		return fmt.Errorf("failed to fetch industry data: %w", err)
	}

	if resp.StatusCode >= 400 {
		return client.HTTPStatusToError(resp.StatusCode, resp.Body)
	}

	var raw models.IndustryResponse
	if err := json.Unmarshal([]byte(resp.Body), &raw); err != nil {
		return fmt.Errorf("failed to parse industry data: %w", err)
	}

	if raw.Error != nil {
		return fmt.Errorf("industry API error: %s - %s", raw.Error.Code, raw.Error.Description)
	}

	data := i.parseData(&raw)

	i.mu.Lock()
	i.dataCache = data
	i.mu.Unlock()

	return nil
}

// parseData converts raw API response to IndustryData.
func (i *Industry) parseData(raw *models.IndustryResponse) *models.IndustryData {
	data := &models.IndustryData{
		Key:        i.key,
		Name:       raw.Data.Name,
		Symbol:     raw.Data.Symbol,
		SectorKey:  raw.Data.SectorKey,
		SectorName: raw.Data.SectorName,
	}

	// Parse overview
	if raw.Data.Overview != nil {
		data.Overview = models.IndustryOverview{
			CompaniesCount: getInt(raw.Data.Overview, "companiesCount"),
			MarketCap:      getRawFloat(raw.Data.Overview, "marketCap"),
			MessageBoardID: getString(raw.Data.Overview, "messageBoardId"),
			Description:    getString(raw.Data.Overview, "description"),
			MarketWeight:   getRawFloat(raw.Data.Overview, "marketWeight"),
			EmployeeCount:  getInt64(raw.Data.Overview, "employeeCount"),
		}
	}

	// Parse top companies
	for _, c := range raw.Data.TopCompanies {
		company := models.IndustryTopCompany{
			Symbol:       getString(c, "symbol"),
			Name:         getString(c, "name"),
			Rating:       getString(c, "rating"),
			MarketWeight: getRawFloat(c, "marketWeight"),
		}
		if company.Symbol != "" {
			data.TopCompanies = append(data.TopCompanies, company)
		}
	}

	// Parse top performing companies
	for _, c := range raw.Data.TopPerformingCompanies {
		company := models.PerformingCompany{
			Symbol:      getString(c, "symbol"),
			Name:        getString(c, "name"),
			YTDReturn:   getRawFloat(c, "ytdReturn"),
			LastPrice:   getRawFloat(c, "lastPrice"),
			TargetPrice: getRawFloat(c, "targetPrice"),
		}
		if company.Symbol != "" {
			data.TopPerformingCompanies = append(data.TopPerformingCompanies, company)
		}
	}

	// Parse top growth companies
	for _, c := range raw.Data.TopGrowthCompanies {
		company := models.GrowthCompany{
			Symbol:         getString(c, "symbol"),
			Name:           getString(c, "name"),
			YTDReturn:      getRawFloat(c, "ytdReturn"),
			GrowthEstimate: getRawFloat(c, "growthEstimate"),
		}
		if company.Symbol != "" {
			data.TopGrowthCompanies = append(data.TopGrowthCompanies, company)
		}
	}

	// Parse research reports
	for _, r := range raw.Data.ResearchReports {
		report := models.ResearchReport{
			ID:          getString(r, "id"),
			Title:       getString(r, "title"),
			Provider:    getString(r, "provider"),
			PublishDate: getString(r, "publishDate"),
			Summary:     getString(r, "summary"),
		}
		if report.ID != "" || report.Title != "" {
			data.ResearchReports = append(data.ResearchReports, report)
		}
	}

	return data
}

// Data returns all industry data.
//
// This fetches and returns the complete industry information including
// overview, top companies, sector info, and performing/growth companies.
//
// Example:
//
//	data, err := i.Data()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Industry: %s in Sector: %s\n", data.Name, data.SectorName)
func (i *Industry) Data() (*models.IndustryData, error) {
	if err := i.fetchData(); err != nil {
		return nil, err
	}

	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.dataCache, nil
}

// Name returns the industry name.
//
// Example:
//
//	name, err := i.Name()
//	fmt.Println(name) // "Semiconductors"
func (i *Industry) Name() (string, error) {
	if err := i.fetchData(); err != nil {
		return "", err
	}

	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.dataCache.Name, nil
}

// Symbol returns the industry symbol.
func (i *Industry) Symbol() (string, error) {
	if err := i.fetchData(); err != nil {
		return "", err
	}

	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.dataCache.Symbol, nil
}

// SectorKey returns the key of the parent sector.
//
// Example:
//
//	sectorKey, err := i.SectorKey()
//	fmt.Println(sectorKey) // "technology"
func (i *Industry) SectorKey() (string, error) {
	if err := i.fetchData(); err != nil {
		return "", err
	}

	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.dataCache.SectorKey, nil
}

// SectorName returns the name of the parent sector.
//
// Example:
//
//	sectorName, err := i.SectorName()
//	fmt.Println(sectorName) // "Technology"
func (i *Industry) SectorName() (string, error) {
	if err := i.fetchData(); err != nil {
		return "", err
	}

	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.dataCache.SectorName, nil
}

// Overview returns the industry overview information.
//
// Example:
//
//	overview, err := i.Overview()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Companies: %d\n", overview.CompaniesCount)
//	fmt.Printf("Market Cap: $%.2f\n", overview.MarketCap)
func (i *Industry) Overview() (models.IndustryOverview, error) {
	if err := i.fetchData(); err != nil {
		return models.IndustryOverview{}, err
	}

	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.dataCache.Overview, nil
}

// TopCompanies returns the top companies in the industry.
//
// Example:
//
//	companies, err := i.TopCompanies()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, c := range companies {
//	    fmt.Printf("%s: %s (Weight: %.2f%%)\n",
//	        c.Symbol, c.Name, c.MarketWeight*100)
//	}
func (i *Industry) TopCompanies() ([]models.IndustryTopCompany, error) {
	if err := i.fetchData(); err != nil {
		return nil, err
	}

	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.dataCache.TopCompanies, nil
}

// TopPerformingCompanies returns the top performing companies in the industry.
//
// Returns companies with highest year-to-date returns.
//
// Example:
//
//	companies, err := i.TopPerformingCompanies()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, c := range companies {
//	    fmt.Printf("%s: YTD %.2f%%, Last: $%.2f, Target: $%.2f\n",
//	        c.Symbol, c.YTDReturn*100, c.LastPrice, c.TargetPrice)
//	}
func (i *Industry) TopPerformingCompanies() ([]models.PerformingCompany, error) {
	if err := i.fetchData(); err != nil {
		return nil, err
	}

	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.dataCache.TopPerformingCompanies, nil
}

// TopGrowthCompanies returns the top growth companies in the industry.
//
// Returns companies with highest growth estimates.
//
// Example:
//
//	companies, err := i.TopGrowthCompanies()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, c := range companies {
//	    fmt.Printf("%s: YTD %.2f%%, Growth Estimate: %.2f%%\n",
//	        c.Symbol, c.YTDReturn*100, c.GrowthEstimate*100)
//	}
func (i *Industry) TopGrowthCompanies() ([]models.GrowthCompany, error) {
	if err := i.fetchData(); err != nil {
		return nil, err
	}

	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.dataCache.TopGrowthCompanies, nil
}

// ResearchReports returns research reports for the industry.
//
// Example:
//
//	reports, err := i.ResearchReports()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, r := range reports {
//	    fmt.Printf("%s: %s\n", r.Provider, r.Title)
//	}
func (i *Industry) ResearchReports() ([]models.ResearchReport, error) {
	if err := i.fetchData(); err != nil {
		return nil, err
	}

	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.dataCache.ResearchReports, nil
}

// ClearCache clears the cached industry data.
// The next call to any data method will fetch fresh data.
func (i *Industry) ClearCache() {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.dataCache = nil
}

// Helper functions for parsing map values

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	}
	return 0
}

func getInt64(m map[string]interface{}, key string) int64 {
	switch v := m[key].(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	}
	return 0
}

func getFloat(m map[string]interface{}, key string) float64 {
	switch v := m[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return 0
}

// getRawFloat extracts the "raw" value from a nested object like {"raw": 123.45, "fmt": "123.45"}
func getRawFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(map[string]interface{}); ok {
		return getFloat(v, "raw")
	}
	return getFloat(m, key)
}
