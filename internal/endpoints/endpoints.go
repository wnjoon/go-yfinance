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

	// Market Time
	MarketTimeURL = Query1URL + "/v6/finance/markettime"

	// Sector/Industry (Domain)
	SectorURL   = Query1URL + "/v1/finance/sectors"
	IndustryURL = Query1URL + "/v1/finance/industries"

	// News
	NewsURL = RootURL + "/xhr/ncp"

	// WebSocket (real-time)
	WebSocketURL = "wss://streamer.finance.yahoo.com/?version=2"
)

// IncomeStatementKeys defines fields for income statement (financials in Yahoo's terminology).
var IncomeStatementKeys = []string{
	"TotalRevenue", "OperatingRevenue", "CostOfRevenue", "GrossProfit",
	"OperatingExpense", "SellingGeneralAndAdministration", "ResearchAndDevelopment",
	"OperatingIncome", "NetNonOperatingInterestIncomeExpense", "InterestIncomeNonOperating",
	"InterestExpenseNonOperating", "OtherIncomeExpense", "PretaxIncome",
	"TaxProvision", "NetIncome", "NetIncomeCommonStockholders",
	"DilutedEPS", "BasicEPS", "DilutedAverageShares", "BasicAverageShares",
	"EBITDA", "EBIT", "NormalizedEBITDA", "NormalizedIncome",
	"ReconciledDepreciation", "ReconciledCostOfRevenue",
}

// BalanceSheetKeys defines fields for balance sheet.
var BalanceSheetKeys = []string{
	"TotalAssets", "CurrentAssets", "CashAndCashEquivalents", "CashCashEquivalentsAndShortTermInvestments",
	"Receivables", "AccountsReceivable", "Inventory", "PrepaidAssets", "OtherCurrentAssets",
	"TotalNonCurrentAssets", "NetPPE", "GrossPPE", "AccumulatedDepreciation",
	"GoodwillAndOtherIntangibleAssets", "Goodwill", "OtherIntangibleAssets",
	"InvestmentsAndAdvances", "LongTermEquityInvestment", "OtherNonCurrentAssets",
	"TotalLiabilitiesNetMinorityInterest", "CurrentLiabilities", "AccountsPayable",
	"PayablesAndAccruedExpenses", "CurrentDebtAndCapitalLeaseObligation", "CurrentDebt",
	"OtherCurrentLiabilities", "TotalNonCurrentLiabilitiesNetMinorityInterest",
	"LongTermDebtAndCapitalLeaseObligation", "LongTermDebt", "OtherNonCurrentLiabilities",
	"TotalEquityGrossMinorityInterest", "StockholdersEquity", "CommonStockEquity",
	"RetainedEarnings", "AdditionalPaidInCapital", "TreasuryStock",
	"TotalDebt", "NetDebt", "WorkingCapital", "TangibleBookValue", "InvestedCapital",
}

// CashFlowKeys defines fields for cash flow statement.
var CashFlowKeys = []string{
	"OperatingCashFlow", "NetIncomeFromContinuingOperations", "DepreciationAmortizationDepletion",
	"Depreciation", "AmortizationOfIntangibles", "DeferredIncomeTax", "DeferredTax",
	"StockBasedCompensation", "ChangeInWorkingCapital", "ChangeInReceivables",
	"ChangeInInventory", "ChangeInPayablesAndAccruedExpense", "ChangeInAccountPayable",
	"ChangeInOtherWorkingCapital", "OtherNonCashItems",
	"InvestingCashFlow", "CapitalExpenditure", "PurchaseOfPPE", "PurchaseOfInvestment",
	"SaleOfInvestment", "NetInvestmentPurchaseAndSale", "PurchaseOfBusiness",
	"NetBusinessPurchaseAndSale", "NetIntangiblesPurchaseAndSale", "OtherInvestingChanges",
	"FinancingCashFlow", "NetIssuancePaymentsOfDebt", "NetLongTermDebtIssuance",
	"LongTermDebtIssuance", "LongTermDebtPayments", "NetShortTermDebtIssuance",
	"NetCommonStockIssuance", "CommonStockIssuance", "CommonStockPayments",
	"RepurchaseOfCapitalStock", "CashDividendsPaid", "CommonStockDividendPaid",
	"NetOtherFinancingCharges", "FreeCashFlow",
	"BeginningCashPosition", "EndCashPosition", "ChangesInCash", "EffectOfExchangeRateChanges",
}

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
