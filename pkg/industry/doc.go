// Package industry provides Yahoo Finance industry data functionality.
//
// # Overview
//
// The industry package allows retrieving financial industry information including
// overview data, top companies, sector information, top performing companies,
// and top growth companies. This is useful for industry-based analysis and screening.
//
// # Basic Usage
//
//	i, err := industry.New("semiconductors")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer i.Close()
//
//	// Get industry overview
//	overview, err := i.Overview()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Companies: %d\n", overview.CompaniesCount)
//	fmt.Printf("Market Cap: $%.2f\n", overview.MarketCap)
//
// # Sector Information
//
// Get the parent sector of an industry:
//
//	sectorKey, err := i.SectorKey()
//	sectorName, err := i.SectorName()
//	fmt.Printf("Part of: %s (%s)\n", sectorName, sectorKey)
//
// # Top Performing Companies
//
// Get companies with highest year-to-date returns:
//
//	companies, err := i.TopPerformingCompanies()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, c := range companies {
//	    fmt.Printf("%s: YTD %.2f%%, Target: $%.2f\n",
//	        c.Symbol, c.YTDReturn*100, c.TargetPrice)
//	}
//
// # Top Growth Companies
//
// Get companies with highest growth estimates:
//
//	companies, err := i.TopGrowthCompanies()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, c := range companies {
//	    fmt.Printf("%s: Growth Estimate: %.2f%%\n",
//	        c.Symbol, c.GrowthEstimate*100)
//	}
//
// # Predefined Industries
//
// Common industry identifiers are available as constants:
//
//	i, _ := industry.NewWithPredefined(models.IndustrySemiconductors)
//	i, _ := industry.NewWithPredefined(models.IndustryBiotechnology)
//	i, _ := industry.NewWithPredefined(models.IndustryBanksDiversified)
//
// Available predefined industries (grouped by sector):
//
// Technology:
//   - IndustrySemiconductors
//   - IndustrySoftwareInfrastructure
//   - IndustrySoftwareApplication
//   - IndustryConsumerElectronics
//   - IndustryComputerHardware
//
// Healthcare:
//   - IndustryBiotechnology
//   - IndustryDrugManufacturersGeneral
//   - IndustryMedicalDevices
//   - IndustryHealthcarePlans
//
// Financial Services:
//   - IndustryBanksDiversified
//   - IndustryBanksRegional
//   - IndustryAssetManagement
//   - IndustryCapitalMarkets
//
// Energy:
//   - IndustryOilGasIntegrated
//   - IndustryOilGasEP
//   - IndustryOilGasMidstream
//
// # Caching
//
// Industry data is cached after the first fetch. To refresh:
//
//	i.ClearCache()
//
// # Custom Client
//
// Provide a custom HTTP client:
//
//	c, _ := client.New()
//	i, _ := industry.New("semiconductors", industry.WithClient(c))
//
// # Thread Safety
//
// All Industry methods are safe for concurrent use from multiple goroutines.
//
// # Python Compatibility
//
// This package implements the same functionality as Python yfinance's Industry class:
//
//	Python                              | Go
//	------------------------------------|------------------------------------
//	yf.Industry("semiconductors")       | industry.New("semiconductors")
//	industry.key                        | i.Key()
//	industry.name                       | i.Name()
//	industry.symbol                     | i.Symbol()
//	industry.sector_key                 | i.SectorKey()
//	industry.sector_name                | i.SectorName()
//	industry.overview                   | i.Overview()
//	industry.top_companies              | i.TopCompanies()
//	industry.top_performing_companies   | i.TopPerformingCompanies()
//	industry.top_growth_companies       | i.TopGrowthCompanies()
//	industry.research_reports           | i.ResearchReports()
package industry
