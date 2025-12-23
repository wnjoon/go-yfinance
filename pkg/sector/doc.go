// Package sector provides Yahoo Finance sector data functionality.
//
// # Overview
//
// The sector package allows retrieving financial sector information including
// overview data, top companies, industries within the sector, top ETFs,
// and top mutual funds. This is useful for sector-based analysis and screening.
//
// # Basic Usage
//
//	s, err := sector.New("technology")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer s.Close()
//
//	// Get sector overview
//	overview, err := s.Overview()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Companies: %d\n", overview.CompaniesCount)
//	fmt.Printf("Market Cap: $%.2f\n", overview.MarketCap)
//
// # Top Companies
//
// Get the top companies within a sector:
//
//	companies, err := s.TopCompanies()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, c := range companies {
//	    fmt.Printf("%s: %s (Weight: %.4f)\n",
//	        c.Symbol, c.Name, c.MarketWeight)
//	}
//
// # Industries
//
// Get industries within the sector:
//
//	industries, err := s.Industries()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, i := range industries {
//	    fmt.Printf("%s: %s\n", i.Key, i.Name)
//	}
//
// # Top ETFs and Mutual Funds
//
// Get top ETFs and mutual funds for the sector:
//
//	etfs, err := s.TopETFs()
//	funds, err := s.TopMutualFunds()
//
// # Predefined Sectors
//
// Common sector identifiers are available as constants:
//
//	s, _ := sector.NewWithPredefined(models.SectorTechnology)
//	s, _ := sector.NewWithPredefined(models.SectorHealthcare)
//	s, _ := sector.NewWithPredefined(models.SectorFinancialServices)
//
// Available predefined sectors:
//   - SectorBasicMaterials: Basic Materials
//   - SectorCommunicationServices: Communication Services
//   - SectorConsumerCyclical: Consumer Cyclical
//   - SectorConsumerDefensive: Consumer Defensive
//   - SectorEnergy: Energy
//   - SectorFinancialServices: Financial Services
//   - SectorHealthcare: Healthcare
//   - SectorIndustrials: Industrials
//   - SectorRealEstate: Real Estate
//   - SectorTechnology: Technology
//   - SectorUtilities: Utilities
//
// # Caching
//
// Sector data is cached after the first fetch. To refresh:
//
//	s.ClearCache()
//
// # Custom Client
//
// Provide a custom HTTP client:
//
//	c, _ := client.New()
//	s, _ := sector.New("technology", sector.WithClient(c))
//
// # Thread Safety
//
// All Sector methods are safe for concurrent use from multiple goroutines.
//
// # Python Compatibility
//
// This package implements the same functionality as Python yfinance's Sector class:
//
//	Python                          | Go
//	--------------------------------|--------------------------------
//	yf.Sector("technology")         | sector.New("technology")
//	sector.key                      | s.Key()
//	sector.name                     | s.Name()
//	sector.symbol                   | s.Symbol()
//	sector.overview                 | s.Overview()
//	sector.top_companies            | s.TopCompanies()
//	sector.industries               | s.Industries()
//	sector.top_etfs                 | s.TopETFs()
//	sector.top_mutual_funds         | s.TopMutualFunds()
//	sector.research_reports         | s.ResearchReports()
package sector
