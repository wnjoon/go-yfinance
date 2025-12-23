// Package market provides Yahoo Finance market status and summary functionality.
//
// # Overview
//
// The market package allows retrieving market trading hours, timezone information,
// and summary data for major market indices. This is useful for determining
// market state and getting an overview of major indices like S&P 500, Dow Jones, etc.
//
// # Basic Usage
//
//	m, err := market.New("us_market")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer m.Close()
//
//	// Get market status (opening/closing times)
//	status, err := m.Status()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if status.Open != nil {
//	    fmt.Printf("Market opens at: %s\n", status.Open.Format("15:04"))
//	}
//	fmt.Printf("Timezone: %s\n", status.Timezone.Short)
//
// # Market Summary
//
// Get an overview of major market indices:
//
//	summary, err := m.Summary()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for exchange, item := range summary {
//	    fmt.Printf("%s: %.2f (%.2f%%)\n",
//	        item.ShortName,
//	        item.RegularMarketPrice,
//	        item.RegularMarketChangePercent)
//	}
//
// # Predefined Markets
//
// Common market identifiers are available as constants:
//
//	m, _ := market.NewWithPredefined(models.MarketUS)  // US market
//	m, _ := market.NewWithPredefined(models.MarketJP)  // Japan market
//	m, _ := market.NewWithPredefined(models.MarketGB)  // UK market
//
// Available predefined markets:
//   - MarketUS: United States
//   - MarketGB: United Kingdom
//   - MarketDE: Germany
//   - MarketFR: France
//   - MarketJP: Japan
//   - MarketHK: Hong Kong
//   - MarketCN: China
//   - MarketCA: Canada
//   - MarketAU: Australia
//   - MarketIN: India
//   - MarketKR: Korea
//   - MarketBR: Brazil
//
// # Market State
//
// Check if the market is currently open:
//
//	isOpen, err := m.IsOpen()
//	if isOpen {
//	    fmt.Println("Market is currently trading")
//	}
//
// # Caching
//
// Market data is cached after the first fetch. To refresh:
//
//	m.ClearCache()
//
// # Custom Client
//
// Provide a custom HTTP client:
//
//	c, _ := client.New()
//	m, _ := market.New("us_market", market.WithClient(c))
//
// # Thread Safety
//
// All Market methods are safe for concurrent use from multiple goroutines.
//
// # Python Compatibility
//
// This package implements the same functionality as Python yfinance's Market class:
//
//	Python                  | Go
//	------------------------|---------------------------
//	yf.Market("us_market")  | market.New("us_market")
//	market.status           | m.Status()
//	market.summary          | m.Summary()
package market
