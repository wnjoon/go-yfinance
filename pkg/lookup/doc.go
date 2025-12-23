// Package lookup provides Yahoo Finance ticker lookup functionality.
//
// # Overview
//
// The lookup package allows searching for financial instruments (stocks, ETFs,
// mutual funds, indices, futures, currencies, and cryptocurrencies) by query string.
// This is useful for finding ticker symbols when you know the company name or
// a partial symbol.
//
// Unlike the search package which provides general search across news, lists, and
// research, lookup is specifically designed for finding ticker symbols with
// optional filtering by instrument type.
//
// # Basic Usage
//
//	l, err := lookup.New("Apple")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer l.Close()
//
//	// Get all matching instruments
//	results, err := l.All(10)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, doc := range results {
//	    fmt.Printf("%s: %s (%s)\n", doc.Symbol, doc.Name, doc.QuoteType)
//	}
//
// # Filtering by Type
//
// You can filter results by specific instrument types:
//
//	// Get only stocks
//	stocks, err := l.Stock(10)
//
//	// Get only ETFs
//	etfs, err := l.ETF(10)
//
//	// Get only mutual funds
//	funds, err := l.MutualFund(10)
//
//	// Get only indices
//	indices, err := l.Index(10)
//
//	// Get only futures
//	futures, err := l.Future(10)
//
//	// Get only currencies
//	currencies, err := l.Currency(10)
//
//	// Get only cryptocurrencies
//	cryptos, err := l.Cryptocurrency(10)
//
// # Pricing Data
//
// By default, lookup results include real-time pricing data such as:
//   - RegularMarketPrice: Current market price
//   - RegularMarketChange: Price change
//   - RegularMarketChangePercent: Percentage change
//   - RegularMarketVolume: Trading volume
//   - MarketCap: Market capitalization
//   - FiftyTwoWeekHigh/Low: 52-week price range
//
// # Caching
//
// Lookup results are cached per query/type combination. To clear the cache:
//
//	l.ClearCache()
//
// # Custom Client
//
// You can provide a custom HTTP client for the Lookup instance:
//
//	c, _ := client.New()
//	l, _ := lookup.New("AAPL", lookup.WithClient(c))
//
// # Thread Safety
//
// All Lookup methods are safe for concurrent use from multiple goroutines.
//
// # Python Compatibility
//
// This package implements the same functionality as Python yfinance's Lookup class:
//
//	Python                      | Go
//	----------------------------|---------------------------
//	yf.Lookup("AAPL")           | lookup.New("AAPL")
//	lookup.get_all(count=10)    | l.All(10)
//	lookup.get_stock(count=10)  | l.Stock(10)
//	lookup.get_etf(count=10)    | l.ETF(10)
//	lookup.get_mutualfund(10)   | l.MutualFund(10)
//	lookup.get_index(count=10)  | l.Index(10)
//	lookup.get_future(count=10) | l.Future(10)
//	lookup.get_currency(10)     | l.Currency(10)
//	lookup.get_cryptocurrency() | l.Cryptocurrency(10)
package lookup
