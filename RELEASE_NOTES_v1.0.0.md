# go-yfinance v1.0.0

🎉 **Full Python yfinance v1.0.0 Compatibility Release**

This release brings go-yfinance to feature parity with Python yfinance v1.0.0, providing a complete Go implementation of Yahoo Finance data access.

## ✨ New Features

### Lookup Module
- `lookup.New(query)` - Search for ticker symbols
- Support for all asset types: Stock, ETF, MutualFund, Index, Future, Currency, Cryptocurrency

### Market Module
- `market.New(market)` / `market.NewWithPredefined(predefined)`
- `Status()` - Get market open/close times and state
- `Summary()` - Get market indices summary
- `IsOpen()` - Check if market is currently open
- 12 predefined markets (US, GB, DE, FR, JP, HK, CN, CA, AU, IN, KR, BR)

### Sector Module
- `sector.New(key)` / `sector.NewWithPredefined(predefined)`
- `Overview()`, `TopCompanies()`, `Industries()`, `TopETFs()`, `TopMutualFunds()`
- 11 predefined sectors

### Industry Module
- `industry.New(key)` / `industry.NewWithPredefined(predefined)`
- `Overview()`, `TopCompanies()`, `TopPerformingCompanies()`, `TopGrowthCompanies()`
- 45 predefined industries

### Calendars Module
- `calendars.New()` / `calendars.New(WithDateRange(start, end))`
- `Earnings(opts)` - S&P earnings announcements
- `IPOs(opts)` - IPO information
- `EconomicEvents(opts)` - Economic releases
- `Splits(opts)` - Stock splits

## 🔧 Improvements

- Thread-safe caching for all new modules
- Comprehensive unit and integration tests
- Full documentation with examples
- Go Report Card A+ rating maintained

## 📚 Documentation

- [API Documentation](https://wnjoon.github.io/go-yfinance/)
- [Python-Go API Mapping](https://github.com/wnjoon/go-yfinance/blob/main/.claude/API_MAPPING.md)

## 📦 Installation

```bash
go get github.com/wnjoon/go-yfinance@v1.0.0
```

## 🙏 Acknowledgments

Full Python yfinance compatibility achieved through careful analysis of the original Python implementation.
