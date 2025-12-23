package models

// IndustryOverview contains overview information for an industry.
type IndustryOverview struct {
	// CompaniesCount is the number of companies in the industry.
	CompaniesCount int `json:"companies_count,omitempty"`

	// MarketCap is the total market capitalization of the industry.
	MarketCap float64 `json:"market_cap,omitempty"`

	// MessageBoardID is the Yahoo Finance message board identifier.
	MessageBoardID string `json:"message_board_id,omitempty"`

	// Description is a text description of the industry.
	Description string `json:"description,omitempty"`

	// MarketWeight is the industry's weight in the market.
	MarketWeight float64 `json:"market_weight,omitempty"`

	// EmployeeCount is the total number of employees in the industry.
	EmployeeCount int64 `json:"employee_count,omitempty"`
}

// IndustryTopCompany represents a top company within an industry.
type IndustryTopCompany struct {
	// Symbol is the stock ticker symbol.
	Symbol string `json:"symbol"`

	// Name is the company name.
	Name string `json:"name"`

	// Rating is the analyst rating.
	Rating string `json:"rating,omitempty"`

	// MarketWeight is the company's weight within the industry.
	MarketWeight float64 `json:"market_weight,omitempty"`
}

// PerformingCompany represents a top performing company in the industry.
type PerformingCompany struct {
	// Symbol is the stock ticker symbol.
	Symbol string `json:"symbol"`

	// Name is the company name.
	Name string `json:"name"`

	// YTDReturn is the year-to-date return.
	YTDReturn float64 `json:"ytd_return,omitempty"`

	// LastPrice is the last traded price.
	LastPrice float64 `json:"last_price,omitempty"`

	// TargetPrice is the analyst target price.
	TargetPrice float64 `json:"target_price,omitempty"`
}

// GrowthCompany represents a top growth company in the industry.
type GrowthCompany struct {
	// Symbol is the stock ticker symbol.
	Symbol string `json:"symbol"`

	// Name is the company name.
	Name string `json:"name"`

	// YTDReturn is the year-to-date return.
	YTDReturn float64 `json:"ytd_return,omitempty"`

	// GrowthEstimate is the estimated growth rate.
	GrowthEstimate float64 `json:"growth_estimate,omitempty"`
}

// IndustryData contains all data for an industry.
//
// This includes overview information, top companies, sector info,
// performing companies, and growth companies.
//
// Example:
//
//	i, err := industry.New("semiconductors")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	data, err := i.Data()
//	fmt.Printf("Industry: %s in Sector: %s\n", data.Name, data.SectorName)
type IndustryData struct {
	// Key is the industry identifier.
	Key string `json:"key"`

	// Name is the display name of the industry.
	Name string `json:"name"`

	// Symbol is the associated symbol (e.g., index symbol).
	Symbol string `json:"symbol,omitempty"`

	// SectorKey is the key of the parent sector.
	SectorKey string `json:"sector_key,omitempty"`

	// SectorName is the name of the parent sector.
	SectorName string `json:"sector_name,omitempty"`

	// Overview contains industry overview information.
	Overview IndustryOverview `json:"overview"`

	// TopCompanies lists the top companies in the industry.
	TopCompanies []IndustryTopCompany `json:"top_companies,omitempty"`

	// TopPerformingCompanies lists the top performing companies.
	TopPerformingCompanies []PerformingCompany `json:"top_performing_companies,omitempty"`

	// TopGrowthCompanies lists the top growth companies.
	TopGrowthCompanies []GrowthCompany `json:"top_growth_companies,omitempty"`

	// ResearchReports contains research reports for the industry.
	ResearchReports []ResearchReport `json:"research_reports,omitempty"`
}

// IndustryResponse represents the raw API response for industry data.
type IndustryResponse struct {
	Data struct {
		Name                   string                   `json:"name"`
		Symbol                 string                   `json:"symbol"`
		SectorKey              string                   `json:"sectorKey"`
		SectorName             string                   `json:"sectorName"`
		Overview               map[string]interface{}   `json:"overview"`
		TopCompanies           []map[string]interface{} `json:"topCompanies"`
		TopPerformingCompanies []map[string]interface{} `json:"topPerformingCompanies"`
		TopGrowthCompanies     []map[string]interface{} `json:"topGrowthCompanies"`
		ResearchReports        []map[string]interface{} `json:"researchReports"`
	} `json:"data"`
	Error *struct {
		Code        string `json:"code"`
		Description string `json:"description"`
	} `json:"error,omitempty"`
}

// PredefinedIndustry represents commonly used industry identifiers.
type PredefinedIndustry string

// Technology Sector Industries
const (
	// IndustrySemiconductors represents the Semiconductors industry.
	IndustrySemiconductors PredefinedIndustry = "semiconductors"

	// IndustrySoftwareInfrastructure represents the Software - Infrastructure industry.
	IndustrySoftwareInfrastructure PredefinedIndustry = "software-infrastructure"

	// IndustrySoftwareApplication represents the Software - Application industry.
	IndustrySoftwareApplication PredefinedIndustry = "software-application"

	// IndustryConsumerElectronics represents the Consumer Electronics industry.
	IndustryConsumerElectronics PredefinedIndustry = "consumer-electronics"

	// IndustryComputerHardware represents the Computer Hardware industry.
	IndustryComputerHardware PredefinedIndustry = "computer-hardware"

	// IndustrySemiconductorEquipment represents the Semiconductor Equipment & Materials industry.
	IndustrySemiconductorEquipment PredefinedIndustry = "semiconductor-equipment-materials"

	// IndustryITServices represents the Information Technology Services industry.
	IndustryITServices PredefinedIndustry = "information-technology-services"

	// IndustryCommunicationEquipment represents the Communication Equipment industry.
	IndustryCommunicationEquipment PredefinedIndustry = "communication-equipment"

	// IndustryElectronicComponents represents the Electronic Components industry.
	IndustryElectronicComponents PredefinedIndustry = "electronic-components"

	// IndustrySolar represents the Solar industry.
	IndustrySolar PredefinedIndustry = "solar"
)

// Healthcare Sector Industries
const (
	// IndustryBiotechnology represents the Biotechnology industry.
	IndustryBiotechnology PredefinedIndustry = "biotechnology"

	// IndustryDrugManufacturersGeneral represents the Drug Manufacturers - General industry.
	IndustryDrugManufacturersGeneral PredefinedIndustry = "drug-manufacturers-general"

	// IndustryMedicalDevices represents the Medical Devices industry.
	IndustryMedicalDevices PredefinedIndustry = "medical-devices"

	// IndustryHealthcarePlans represents the Healthcare Plans industry.
	IndustryHealthcarePlans PredefinedIndustry = "healthcare-plans"

	// IndustryDiagnosticsResearch represents the Diagnostics & Research industry.
	IndustryDiagnosticsResearch PredefinedIndustry = "diagnostics-research"
)

// Financial Services Sector Industries
const (
	// IndustryBanksDiversified represents the Banks - Diversified industry.
	IndustryBanksDiversified PredefinedIndustry = "banks-diversified"

	// IndustryBanksRegional represents the Banks - Regional industry.
	IndustryBanksRegional PredefinedIndustry = "banks-regional"

	// IndustryAssetManagement represents the Asset Management industry.
	IndustryAssetManagement PredefinedIndustry = "asset-management"

	// IndustryCapitalMarkets represents the Capital Markets industry.
	IndustryCapitalMarkets PredefinedIndustry = "capital-markets"

	// IndustryInsuranceDiversified represents the Insurance - Diversified industry.
	IndustryInsuranceDiversified PredefinedIndustry = "insurance-diversified"

	// IndustryCreditServices represents the Credit Services industry.
	IndustryCreditServices PredefinedIndustry = "credit-services"
)

// Energy Sector Industries
const (
	// IndustryOilGasIntegrated represents the Oil & Gas Integrated industry.
	IndustryOilGasIntegrated PredefinedIndustry = "oil-gas-integrated"

	// IndustryOilGasEP represents the Oil & Gas E&P industry.
	IndustryOilGasEP PredefinedIndustry = "oil-gas-e-p"

	// IndustryOilGasMidstream represents the Oil & Gas Midstream industry.
	IndustryOilGasMidstream PredefinedIndustry = "oil-gas-midstream"

	// IndustryOilGasEquipmentServices represents the Oil & Gas Equipment & Services industry.
	IndustryOilGasEquipmentServices PredefinedIndustry = "oil-gas-equipment-services"
)

// Consumer Cyclical Industries
const (
	// IndustryAutoManufacturers represents the Auto Manufacturers industry.
	IndustryAutoManufacturers PredefinedIndustry = "auto-manufacturers"

	// IndustryInternetRetail represents the Internet Retail industry.
	IndustryInternetRetail PredefinedIndustry = "internet-retail"

	// IndustryRestaurants represents the Restaurants industry.
	IndustryRestaurants PredefinedIndustry = "restaurants"

	// IndustryHomeImprovementRetail represents the Home Improvement Retail industry.
	IndustryHomeImprovementRetail PredefinedIndustry = "home-improvement-retail"
)

// Industrials Sector Industries
const (
	// IndustryAerospaceDefense represents the Aerospace & Defense industry.
	IndustryAerospaceDefense PredefinedIndustry = "aerospace-defense"

	// IndustryRailroads represents the Railroads industry.
	IndustryRailroads PredefinedIndustry = "railroads"

	// IndustryAirlines represents the Airlines industry.
	IndustryAirlines PredefinedIndustry = "airlines"

	// IndustryBuildingProductsEquipment represents the Building Products & Equipment industry.
	IndustryBuildingProductsEquipment PredefinedIndustry = "building-products-equipment"
)

// AllIndustries returns a list of common predefined industry identifiers.
// Note: This is not exhaustive - there are many more industries available.
func AllIndustries() []PredefinedIndustry {
	return []PredefinedIndustry{
		// Technology
		IndustrySemiconductors,
		IndustrySoftwareInfrastructure,
		IndustrySoftwareApplication,
		IndustryConsumerElectronics,
		IndustryComputerHardware,
		// Healthcare
		IndustryBiotechnology,
		IndustryDrugManufacturersGeneral,
		IndustryMedicalDevices,
		// Financial
		IndustryBanksDiversified,
		IndustryAssetManagement,
		IndustryCapitalMarkets,
		// Energy
		IndustryOilGasIntegrated,
		IndustryOilGasEP,
		// Consumer
		IndustryAutoManufacturers,
		IndustryInternetRetail,
		// Industrials
		IndustryAerospaceDefense,
		IndustryRailroads,
	}
}
