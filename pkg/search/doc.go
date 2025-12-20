// Package search provides Yahoo Finance search functionality.
//
// # Overview
//
// The search package allows you to search for stock symbols, company names,
// and other financial assets on Yahoo Finance. It also retrieves related
// news articles, lists, and research reports.
//
// # Basic Usage
//
//	s, err := search.New()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer s.Close()
//
//	result, err := s.Search("AAPL")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, quote := range result.Quotes {
//	    fmt.Printf("%s: %s (%s)\n", quote.Symbol, quote.ShortName, quote.Exchange)
//	}
//
// # Search Methods
//
// The Search struct provides the following methods:
//
//   - [Search.Search]: Simple search with default parameters
//   - [Search.SearchWithParams]: Search with custom parameters
//   - [Search.Quotes]: Get only quote results
//   - [Search.News]: Get only news results
//
// # Search Parameters
//
// Use [models.SearchParams] to customize your search:
//
//	params := models.SearchParams{
//	    Query:            "Apple",
//	    MaxResults:       10,
//	    NewsCount:        5,
//	    EnableFuzzyQuery: true,
//	}
//	result, err := s.SearchWithParams(params)
//
// # Thread Safety
//
// All Search methods are safe for concurrent use from multiple goroutines.
package search
