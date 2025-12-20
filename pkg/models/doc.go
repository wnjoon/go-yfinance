// Package models provides data structures for Yahoo Finance API responses.
//
// # Overview
//
// The models package defines all the data types used throughout go-yfinance.
// These types represent the structured data returned by Yahoo Finance APIs.
//
// # Core Types
//
// Quote and Price Data:
//   - [Quote]: Real-time quote data including price, volume, and market state
//   - [FastInfo]: Quick-access subset of quote/info data
//   - [Bar]: Single OHLCV candlestick bar
//   - [History]: Historical price data as a collection of bars
//
// Company Information:
//   - [Info]: Comprehensive company information and statistics
//   - [Officer]: Company officer/executive information
//
// Options:
//   - [Option]: Single option contract (call or put)
//   - [OptionChain]: Complete option chain with calls and puts
//   - [OptionsData]: All expiration dates and strikes
//
// Financial Statements:
//   - [FinancialStatement]: Income statement, balance sheet, or cash flow data
//   - [FinancialItem]: Single financial data point with date and value
//
// Analysis:
//   - [RecommendationTrend]: Analyst buy/hold/sell recommendations
//   - [PriceTarget]: Analyst price targets
//   - [EarningsEstimate]: Earnings estimates by period
//   - [RevenueEstimate]: Revenue estimates by period
//   - [EPSTrend]: EPS trend over time
//   - [EPSRevision]: EPS revision counts
//   - [EarningsHistory]: Historical earnings vs estimates
//   - [GrowthEstimate]: Growth estimates from various sources
//
// Holders:
//   - [MajorHolders]: Major shareholders breakdown (insiders, institutions)
//   - [Holder]: Institutional or mutual fund holder information
//   - [InsiderTransaction]: Insider purchase/sale transaction
//   - [InsiderHolder]: Company insider with holdings
//   - [InsiderPurchases]: Net share purchase activity summary
//   - [HoldersData]: Aggregate holder data container
//
// Calendar:
//   - [Calendar]: Upcoming events including earnings and dividend dates
//
// Search:
//   - [SearchResult]: Complete search response with quotes, news, lists
//   - [SearchQuote]: Quote match from search results
//   - [SearchNews]: News article from search results
//   - [SearchParams]: Search query parameters
//
// Screener:
//   - [ScreenerResult]: Stock screener results with pagination
//   - [ScreenerQuote]: Stock from screener results with financial data
//   - [ScreenerQuery]: Custom screener query structure
//   - [ScreenerParams]: Screener parameters (offset, count, sort)
//   - [PredefinedScreener]: Predefined screener identifiers (day_gainers, etc.)
//
// # History Parameters
//
// The [HistoryParams] type controls historical data fetching:
//
//	params := models.HistoryParams{
//	    Period:   "1mo",      // 1d, 5d, 1mo, 3mo, 6mo, 1y, 2y, 5y, 10y, ytd, max
//	    Interval: "1d",       // 1m, 2m, 5m, 15m, 30m, 60m, 90m, 1h, 1d, 5d, 1wk, 1mo, 3mo
//	    PrePost:  false,      // Include pre/post market data
//	}
//
// Use [ValidPeriods] and [ValidIntervals] to get lists of valid values.
package models
