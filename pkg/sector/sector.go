package sector

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"

	"github.com/wnjoon/go-yfinance/internal/endpoints"
	"github.com/wnjoon/go-yfinance/pkg/client"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

// Sector provides access to financial sector data from Yahoo Finance.
//
// Sector allows retrieving sector-related information such as
// overview, top companies, industries, top ETFs, and top mutual funds.
type Sector struct {
	key string

	client     *client.Client
	auth       *client.AuthManager
	ownsClient bool

	// Cached data
	mu        sync.RWMutex
	dataCache *models.SectorData
}

// Option is a function that configures a Sector instance.
type Option func(*Sector)

// WithClient sets a custom HTTP client for the Sector instance.
func WithClient(c *client.Client) Option {
	return func(s *Sector) {
		s.client = c
		s.ownsClient = false
	}
}

// New creates a new Sector instance for the given sector key.
//
// Sector keys are lowercase with hyphens, e.g., "technology", "basic-materials".
// Use predefined constants from models package for convenience.
//
// Example:
//
//	s, err := sector.New("technology")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer s.Close()
//
//	overview, err := s.Overview()
//	fmt.Printf("Sector has %d companies\n", overview.CompaniesCount)
func New(key string, opts ...Option) (*Sector, error) {
	if key == "" {
		return nil, fmt.Errorf("sector key cannot be empty")
	}

	s := &Sector{
		key:        key,
		ownsClient: true,
	}

	for _, opt := range opts {
		opt(s)
	}

	if s.client == nil {
		c, err := client.New()
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}
		s.client = c
	}

	s.auth = client.NewAuthManager(s.client)

	return s, nil
}

// NewWithPredefined creates a new Sector instance using a predefined sector constant.
//
// Example:
//
//	s, err := sector.NewWithPredefined(models.SectorTechnology)
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewWithPredefined(sector models.PredefinedSector, opts ...Option) (*Sector, error) {
	return New(string(sector), opts...)
}

// Close releases resources used by the Sector instance.
func (s *Sector) Close() {
	if s.ownsClient && s.client != nil {
		s.client.Close()
	}
}

// Key returns the sector key.
func (s *Sector) Key() string {
	return s.key
}

// fetchData fetches sector data from Yahoo Finance API.
func (s *Sector) fetchData() error {
	s.mu.RLock()
	if s.dataCache != nil {
		s.mu.RUnlock()
		return nil
	}
	s.mu.RUnlock()

	queryURL := fmt.Sprintf("%s/%s", endpoints.SectorURL, s.key)

	params := url.Values{}
	params.Set("formatted", "true")
	params.Set("withReturns", "true")
	params.Set("lang", "en-US")
	params.Set("region", "US")

	// Add crumb authentication
	params, err := s.auth.AddCrumbToParams(params)
	if err != nil {
		return fmt.Errorf("failed to add crumb: %w", err)
	}

	resp, err := s.client.Get(queryURL, params)
	if err != nil {
		return fmt.Errorf("failed to fetch sector data: %w", err)
	}

	if resp.StatusCode >= 400 {
		return client.HTTPStatusToError(resp.StatusCode, resp.Body)
	}

	var raw models.SectorResponse
	if err := json.Unmarshal([]byte(resp.Body), &raw); err != nil {
		return fmt.Errorf("failed to parse sector data: %w", err)
	}

	if raw.Error != nil {
		return fmt.Errorf("sector API error: %s - %s", raw.Error.Code, raw.Error.Description)
	}

	data := s.parseData(&raw)

	s.mu.Lock()
	s.dataCache = data
	s.mu.Unlock()

	return nil
}

// parseData converts raw API response to SectorData.
func (s *Sector) parseData(raw *models.SectorResponse) *models.SectorData {
	data := &models.SectorData{
		Key:    s.key,
		Name:   raw.Data.Name,
		Symbol: raw.Data.Symbol,
	}

	// Parse overview
	if raw.Data.Overview != nil {
		data.Overview = models.SectorOverview{
			CompaniesCount:  getInt(raw.Data.Overview, "companiesCount"),
			MarketCap:       getRawFloat(raw.Data.Overview, "marketCap"),
			MessageBoardID:  getString(raw.Data.Overview, "messageBoardId"),
			Description:     getString(raw.Data.Overview, "description"),
			IndustriesCount: getInt(raw.Data.Overview, "industriesCount"),
			MarketWeight:    getRawFloat(raw.Data.Overview, "marketWeight"),
			EmployeeCount:   getInt64(raw.Data.Overview, "employeeCount"),
		}
	}

	// Parse top companies
	for _, c := range raw.Data.TopCompanies {
		company := models.SectorTopCompany{
			Symbol:       getString(c, "symbol"),
			Name:         getString(c, "name"),
			Rating:       getString(c, "rating"),
			MarketWeight: getRawFloat(c, "marketWeight"),
		}
		if company.Symbol != "" {
			data.TopCompanies = append(data.TopCompanies, company)
		}
	}

	// Parse industries
	for _, i := range raw.Data.Industries {
		name := getString(i, "name")
		if name == "All Industries" {
			continue
		}
		industry := models.SectorIndustry{
			Key:          getString(i, "key"),
			Name:         name,
			Symbol:       getString(i, "symbol"),
			MarketWeight: getRawFloat(i, "marketWeight"),
		}
		if industry.Key != "" {
			data.Industries = append(data.Industries, industry)
		}
	}

	// Parse top ETFs
	data.TopETFs = make(map[string]string)
	for _, e := range raw.Data.TopETFs {
		symbol := getString(e, "symbol")
		name := getString(e, "name")
		if symbol != "" {
			data.TopETFs[symbol] = name
		}
	}

	// Parse top mutual funds
	data.TopMutualFunds = make(map[string]string)
	for _, f := range raw.Data.TopMutualFunds {
		symbol := getString(f, "symbol")
		name := getString(f, "name")
		if symbol != "" {
			data.TopMutualFunds[symbol] = name
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

// Data returns all sector data.
//
// This fetches and returns the complete sector information including
// overview, top companies, industries, top ETFs, and mutual funds.
//
// Example:
//
//	data, err := s.Data()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Sector: %s\n", data.Name)
//	fmt.Printf("Companies: %d\n", data.Overview.CompaniesCount)
func (s *Sector) Data() (*models.SectorData, error) {
	if err := s.fetchData(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dataCache, nil
}

// Name returns the sector name.
//
// Example:
//
//	name, err := s.Name()
//	fmt.Println(name) // "Technology"
func (s *Sector) Name() (string, error) {
	if err := s.fetchData(); err != nil {
		return "", err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dataCache.Name, nil
}

// Symbol returns the sector symbol.
func (s *Sector) Symbol() (string, error) {
	if err := s.fetchData(); err != nil {
		return "", err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dataCache.Symbol, nil
}

// Overview returns the sector overview information.
//
// Example:
//
//	overview, err := s.Overview()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Companies: %d\n", overview.CompaniesCount)
//	fmt.Printf("Market Cap: $%.2f\n", overview.MarketCap)
func (s *Sector) Overview() (models.SectorOverview, error) {
	if err := s.fetchData(); err != nil {
		return models.SectorOverview{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dataCache.Overview, nil
}

// TopCompanies returns the top companies in the sector.
//
// Example:
//
//	companies, err := s.TopCompanies()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, c := range companies {
//	    fmt.Printf("%s: %s (Weight: %.2f%%)\n",
//	        c.Symbol, c.Name, c.MarketWeight*100)
//	}
func (s *Sector) TopCompanies() ([]models.SectorTopCompany, error) {
	if err := s.fetchData(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dataCache.TopCompanies, nil
}

// Industries returns the industries within the sector.
//
// Example:
//
//	industries, err := s.Industries()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, i := range industries {
//	    fmt.Printf("%s: %s\n", i.Key, i.Name)
//	}
func (s *Sector) Industries() ([]models.SectorIndustry, error) {
	if err := s.fetchData(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dataCache.Industries, nil
}

// TopETFs returns the top ETFs for the sector.
//
// Returns a map of ETF symbols to names.
//
// Example:
//
//	etfs, err := s.TopETFs()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for symbol, name := range etfs {
//	    fmt.Printf("%s: %s\n", symbol, name)
//	}
func (s *Sector) TopETFs() (map[string]string, error) {
	if err := s.fetchData(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dataCache.TopETFs, nil
}

// TopMutualFunds returns the top mutual funds for the sector.
//
// Returns a map of mutual fund symbols to names.
//
// Example:
//
//	funds, err := s.TopMutualFunds()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for symbol, name := range funds {
//	    fmt.Printf("%s: %s\n", symbol, name)
//	}
func (s *Sector) TopMutualFunds() (map[string]string, error) {
	if err := s.fetchData(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dataCache.TopMutualFunds, nil
}

// ResearchReports returns research reports for the sector.
//
// Example:
//
//	reports, err := s.ResearchReports()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, r := range reports {
//	    fmt.Printf("%s: %s\n", r.Provider, r.Title)
//	}
func (s *Sector) ResearchReports() ([]models.ResearchReport, error) {
	if err := s.fetchData(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dataCache.ResearchReports, nil
}

// ClearCache clears the cached sector data.
// The next call to any data method will fetch fresh data.
func (s *Sector) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dataCache = nil
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
