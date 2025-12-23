package models

// SectorOverview contains overview information for a sector.
type SectorOverview struct {
	// CompaniesCount is the number of companies in the sector.
	CompaniesCount int `json:"companies_count,omitempty"`

	// MarketCap is the total market capitalization of the sector.
	MarketCap float64 `json:"market_cap,omitempty"`

	// MessageBoardID is the Yahoo Finance message board identifier.
	MessageBoardID string `json:"message_board_id,omitempty"`

	// Description is a text description of the sector.
	Description string `json:"description,omitempty"`

	// IndustriesCount is the number of industries in the sector.
	IndustriesCount int `json:"industries_count,omitempty"`

	// MarketWeight is the sector's weight in the market.
	MarketWeight float64 `json:"market_weight,omitempty"`

	// EmployeeCount is the total number of employees in the sector.
	EmployeeCount int64 `json:"employee_count,omitempty"`
}

// SectorTopCompany represents a top company within a sector.
type SectorTopCompany struct {
	// Symbol is the stock ticker symbol.
	Symbol string `json:"symbol"`

	// Name is the company name.
	Name string `json:"name"`

	// Rating is the analyst rating.
	Rating string `json:"rating,omitempty"`

	// MarketWeight is the company's weight within the sector.
	MarketWeight float64 `json:"market_weight,omitempty"`
}

// SectorIndustry represents an industry within a sector.
type SectorIndustry struct {
	// Key is the unique identifier for the industry.
	Key string `json:"key"`

	// Name is the display name of the industry.
	Name string `json:"name"`

	// Symbol is the associated symbol.
	Symbol string `json:"symbol,omitempty"`

	// MarketWeight is the industry's weight within the sector.
	MarketWeight float64 `json:"market_weight,omitempty"`
}

// ResearchReport represents a research report.
type ResearchReport struct {
	// ID is the unique report identifier.
	ID string `json:"id,omitempty"`

	// Title is the report title.
	Title string `json:"title,omitempty"`

	// Provider is the report provider name.
	Provider string `json:"provider,omitempty"`

	// PublishDate is the publication date.
	PublishDate string `json:"publish_date,omitempty"`

	// Summary is a brief summary of the report.
	Summary string `json:"summary,omitempty"`
}

// SectorData contains all data for a sector.
//
// This includes overview information, top companies, industries,
// top ETFs, and top mutual funds.
//
// Example:
//
//	s, err := sector.New("technology")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	data, err := s.Data()
//	fmt.Printf("Sector: %s has %d companies\n", data.Name, data.Overview.CompaniesCount)
type SectorData struct {
	// Key is the sector identifier.
	Key string `json:"key"`

	// Name is the display name of the sector.
	Name string `json:"name"`

	// Symbol is the associated symbol (e.g., index symbol).
	Symbol string `json:"symbol,omitempty"`

	// Overview contains sector overview information.
	Overview SectorOverview `json:"overview"`

	// TopCompanies lists the top companies in the sector.
	TopCompanies []SectorTopCompany `json:"top_companies,omitempty"`

	// Industries lists the industries within the sector.
	Industries []SectorIndustry `json:"industries,omitempty"`

	// TopETFs maps ETF symbols to names.
	TopETFs map[string]string `json:"top_etfs,omitempty"`

	// TopMutualFunds maps mutual fund symbols to names.
	TopMutualFunds map[string]string `json:"top_mutual_funds,omitempty"`

	// ResearchReports contains research reports for the sector.
	ResearchReports []ResearchReport `json:"research_reports,omitempty"`
}

// SectorResponse represents the raw API response for sector data.
type SectorResponse struct {
	Data struct {
		Name            string                   `json:"name"`
		Symbol          string                   `json:"symbol"`
		Overview        map[string]interface{}   `json:"overview"`
		TopCompanies    []map[string]interface{} `json:"topCompanies"`
		Industries      []map[string]interface{} `json:"industries"`
		TopETFs         []map[string]interface{} `json:"topETFs"`
		TopMutualFunds  []map[string]interface{} `json:"topMutualFunds"`
		ResearchReports []map[string]interface{} `json:"researchReports"`
	} `json:"data"`
	Error *struct {
		Code        string `json:"code"`
		Description string `json:"description"`
	} `json:"error,omitempty"`
}

// PredefinedSector represents commonly used sector identifiers.
type PredefinedSector string

const (
	// SectorBasicMaterials represents the Basic Materials sector.
	SectorBasicMaterials PredefinedSector = "basic-materials"

	// SectorCommunicationServices represents the Communication Services sector.
	SectorCommunicationServices PredefinedSector = "communication-services"

	// SectorConsumerCyclical represents the Consumer Cyclical sector.
	SectorConsumerCyclical PredefinedSector = "consumer-cyclical"

	// SectorConsumerDefensive represents the Consumer Defensive sector.
	SectorConsumerDefensive PredefinedSector = "consumer-defensive"

	// SectorEnergy represents the Energy sector.
	SectorEnergy PredefinedSector = "energy"

	// SectorFinancialServices represents the Financial Services sector.
	SectorFinancialServices PredefinedSector = "financial-services"

	// SectorHealthcare represents the Healthcare sector.
	SectorHealthcare PredefinedSector = "healthcare"

	// SectorIndustrials represents the Industrials sector.
	SectorIndustrials PredefinedSector = "industrials"

	// SectorRealEstate represents the Real Estate sector.
	SectorRealEstate PredefinedSector = "real-estate"

	// SectorTechnology represents the Technology sector.
	SectorTechnology PredefinedSector = "technology"

	// SectorUtilities represents the Utilities sector.
	SectorUtilities PredefinedSector = "utilities"
)

// AllSectors returns a list of all predefined sector identifiers.
func AllSectors() []PredefinedSector {
	return []PredefinedSector{
		SectorBasicMaterials,
		SectorCommunicationServices,
		SectorConsumerCyclical,
		SectorConsumerDefensive,
		SectorEnergy,
		SectorFinancialServices,
		SectorHealthcare,
		SectorIndustrials,
		SectorRealEstate,
		SectorTechnology,
		SectorUtilities,
	}
}
