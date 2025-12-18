// Package endpoints defines Yahoo Finance API endpoint constants.
package endpoints

const (
	// Base URLs
	BaseURL   = "https://query2.finance.yahoo.com"
	Query1URL = "https://query1.finance.yahoo.com"
	RootURL   = "https://finance.yahoo.com"

	// Authentication
	CookieURL = "https://fc.yahoo.com"
	CrumbURL  = BaseURL + "/v1/test/getcrumb"

	// CSRF Consent (fallback authentication)
	ConsentURL        = "https://guce.yahoo.com/consent"
	CollectConsentURL = "https://consent.yahoo.com/v2/collectConsent"
	CopyConsentURL    = "https://guce.yahoo.com/copyConsent"
	CrumbCSRFURL      = BaseURL + "/v1/test/getcrumb"

	// Chart / History
	ChartURL = BaseURL + "/v8/finance/chart"

	// Quote Summary (info, financials, etc.)
	QuoteSummaryURL = BaseURL + "/v10/finance/quoteSummary"

	// Quote (real-time quote data)
	QuoteURL = Query1URL + "/v7/finance/quote"

	// Options
	OptionsURL = BaseURL + "/v7/finance/options"

	// Fundamentals / Timeseries
	FundamentalsURL = BaseURL + "/ws/fundamentals-timeseries/v1/finance/timeseries"

	// Search
	SearchURL = BaseURL + "/v1/finance/search"

	// Lookup
	LookupURL = Query1URL + "/v1/finance/lookup"

	// Screener
	ScreenerURL = Query1URL + "/v1/finance/screener"

	// Market Summary
	MarketSummaryURL = Query1URL + "/v6/finance/quote/marketSummary"

	// WebSocket (real-time)
	WebSocketURL = "wss://streamer.finance.yahoo.com/?version=2"
)

// QuoteSummaryModules defines available modules for quoteSummary endpoint.
var QuoteSummaryModules = []string{
	"summaryProfile",
	"summaryDetail",
	"assetProfile",
	"fundProfile",
	"price",
	"quoteType",
	"esgScores",
	"incomeStatementHistory",
	"incomeStatementHistoryQuarterly",
	"balanceSheetHistory",
	"balanceSheetHistoryQuarterly",
	"cashFlowStatementHistory",
	"cashFlowStatementHistoryQuarterly",
	"defaultKeyStatistics",
	"financialData",
	"calendarEvents",
	"secFilings",
	"upgradeDowngradeHistory",
	"institutionOwnership",
	"fundOwnership",
	"majorDirectHolders",
	"majorHoldersBreakdown",
	"insiderTransactions",
	"insiderHolders",
	"netSharePurchaseActivity",
	"earnings",
	"earningsHistory",
	"earningsTrend",
	"industryTrend",
	"indexTrend",
	"sectorTrend",
	"recommendationTrend",
	"futuresChain",
}
