// Package screener provides Yahoo Finance stock screener functionality.
//
// # Overview
//
// The screener package allows you to filter stocks based on various criteria
// using Yahoo Finance's predefined screeners or custom queries.
//
// # Predefined Screeners
//
// Yahoo Finance provides several predefined screeners:
//
//   - day_gainers: Stocks with highest percentage gain today
//   - day_losers: Stocks with highest percentage loss today
//   - most_actives: Stocks with highest trading volume
//   - aggressive_small_caps: Aggressive small cap stocks
//   - growth_technology_stocks: Growth technology stocks
//   - undervalued_growth_stocks: Undervalued growth stocks
//   - undervalued_large_caps: Undervalued large cap stocks
//   - most_shorted_stocks: Most shorted stocks
//   - small_cap_gainers: Small cap gainers
//
// # Basic Usage
//
//	s, err := screener.New()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer s.Close()
//
//	result, err := s.DayGainers(10)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, quote := range result.Quotes {
//	    fmt.Printf("%s: %.2f%%\n", quote.Symbol, quote.RegularMarketChangePercent)
//	}
//
// # Screener Methods
//
// The Screener struct provides the following methods:
//
// Predefined Screeners:
//   - [Screener.Screen]: Use any predefined screener
//   - [Screener.DayGainers]: Top gaining stocks
//   - [Screener.DayLosers]: Top losing stocks
//   - [Screener.MostActives]: Most actively traded stocks
//
// Custom Queries:
//   - [Screener.ScreenWithQuery]: Use custom query criteria
//
// # Custom Queries
//
// Build custom queries using [models.ScreenerQuery]:
//
//	// Find US tech stocks with price > $50
//	query := models.NewEquityQuery(models.OpAND, []interface{}{
//	    models.NewEquityQuery(models.OpEQ, []interface{}{"region", "us"}),
//	    models.NewEquityQuery(models.OpEQ, []interface{}{"sector", "Technology"}),
//	    models.NewEquityQuery(models.OpGT, []interface{}{"intradayprice", 50}),
//	})
//	result, err := s.ScreenWithQuery(query, nil)
//
// # Query Operators
//
// Available operators for custom queries:
//
//   - OpEQ: Equals
//   - OpGT: Greater than
//   - OpLT: Less than
//   - OpGTE: Greater than or equal
//   - OpLTE: Less than or equal
//   - OpBTWN: Between (requires min and max values)
//   - OpAND: All conditions must match
//   - OpOR: Any condition can match
//
// # Thread Safety
//
// All Screener methods are safe for concurrent use from multiple goroutines.
package screener
