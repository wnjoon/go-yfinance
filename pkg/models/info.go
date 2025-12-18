package models

// Info represents comprehensive company information from quoteSummary API.
type Info struct {
	// Basic identifiers
	Symbol    string `json:"symbol"`
	ShortName string `json:"shortName"`
	LongName  string `json:"longName"`

	// Company profile (assetProfile module)
	Sector                 string   `json:"sector,omitempty"`
	Industry               string   `json:"industry,omitempty"`
	FullTimeEmployees      int64    `json:"fullTimeEmployees,omitempty"`
	City                   string   `json:"city,omitempty"`
	State                  string   `json:"state,omitempty"`
	Country                string   `json:"country,omitempty"`
	Phone                  string   `json:"phone,omitempty"`
	Address1               string   `json:"address1,omitempty"`
	Address2               string   `json:"address2,omitempty"`
	Zip                    string   `json:"zip,omitempty"`
	Website                string   `json:"website,omitempty"`
	LongBusinessSummary    string   `json:"longBusinessSummary,omitempty"`
	CompanyOfficers        []Officer `json:"companyOfficers,omitempty"`
	AuditRisk              int      `json:"auditRisk,omitempty"`
	BoardRisk              int      `json:"boardRisk,omitempty"`
	CompensationRisk       int      `json:"compensationRisk,omitempty"`
	ShareHolderRightsRisk  int      `json:"shareHolderRightsRisk,omitempty"`
	OverallRisk            int      `json:"overallRisk,omitempty"`
	GovernanceEpochDate    int64    `json:"governanceEpochDate,omitempty"`
	CompensationAsOfEpochDate int64 `json:"compensationAsOfEpochDate,omitempty"`

	// Quote type info (quoteType module)
	QuoteType              string `json:"quoteType,omitempty"`
	Exchange               string `json:"exchange,omitempty"`
	ExchangeTimezoneName   string `json:"exchangeTimezoneName,omitempty"`
	ExchangeTimezoneShort  string `json:"exchangeTimezoneShortName,omitempty"`
	Currency               string `json:"currency,omitempty"`
	Market                 string `json:"market,omitempty"`
	FirstTradeDateEpoch    int64  `json:"firstTradeDateEpochUtc,omitempty"`
	GMTOffset              int    `json:"gmtOffSetMilliseconds,omitempty"`
	TimeZoneFullName       string `json:"timeZoneFullName,omitempty"`
	TimeZoneShortName      string `json:"timeZoneShortName,omitempty"`

	// Key statistics (defaultKeyStatistics module)
	EnterpriseValue                  int64   `json:"enterpriseValue,omitempty"`
	ForwardPE                        float64 `json:"forwardPE,omitempty"`
	ProfitMargins                    float64 `json:"profitMargins,omitempty"`
	FloatShares                      int64   `json:"floatShares,omitempty"`
	SharesOutstanding                int64   `json:"sharesOutstanding,omitempty"`
	SharesShort                      int64   `json:"sharesShort,omitempty"`
	SharesShortPriorMonth            int64   `json:"sharesShortPriorMonth,omitempty"`
	SharesShortPreviousMonthDate     int64   `json:"sharesShortPreviousMonthDate,omitempty"`
	DateShortInterest                int64   `json:"dateShortInterest,omitempty"`
	SharesPercentSharesOut           float64 `json:"sharesPercentSharesOut,omitempty"`
	HeldPercentInsiders              float64 `json:"heldPercentInsiders,omitempty"`
	HeldPercentInstitutions          float64 `json:"heldPercentInstitutions,omitempty"`
	ShortRatio                       float64 `json:"shortRatio,omitempty"`
	ShortPercentOfFloat              float64 `json:"shortPercentOfFloat,omitempty"`
	Beta                             float64 `json:"beta,omitempty"`
	ImpliedSharesOutstanding         int64   `json:"impliedSharesOutstanding,omitempty"`
	BookValue                        float64 `json:"bookValue,omitempty"`
	PriceToBook                      float64 `json:"priceToBook,omitempty"`
	LastFiscalYearEnd                int64   `json:"lastFiscalYearEnd,omitempty"`
	NextFiscalYearEnd                int64   `json:"nextFiscalYearEnd,omitempty"`
	MostRecentQuarter                int64   `json:"mostRecentQuarter,omitempty"`
	EarningsQuarterlyGrowth          float64 `json:"earningsQuarterlyGrowth,omitempty"`
	NetIncomeToCommon                int64   `json:"netIncomeToCommon,omitempty"`
	TrailingEps                      float64 `json:"trailingEps,omitempty"`
	ForwardEps                       float64 `json:"forwardEps,omitempty"`
	PegRatio                         float64 `json:"pegRatio,omitempty"`
	LastSplitFactor                  string  `json:"lastSplitFactor,omitempty"`
	LastSplitDate                    int64   `json:"lastSplitDate,omitempty"`
	EnterpriseToRevenue              float64 `json:"enterpriseToRevenue,omitempty"`
	EnterpriseToEbitda               float64 `json:"enterpriseToEbitda,omitempty"`
	FiftyTwoWeekChange               float64 `json:"52WeekChange,omitempty"`
	SandP52WeekChange                float64 `json:"SandP52WeekChange,omitempty"`
	LastDividendValue                float64 `json:"lastDividendValue,omitempty"`
	LastDividendDate                 int64   `json:"lastDividendDate,omitempty"`
	LastCapGain                      float64 `json:"lastCapGain,omitempty"`
	AnnualHoldingsTurnover           float64 `json:"annualHoldingsTurnover,omitempty"`

	// Summary detail (summaryDetail module)
	PreviousClose            float64 `json:"previousClose,omitempty"`
	Open                     float64 `json:"open,omitempty"`
	DayLow                   float64 `json:"dayLow,omitempty"`
	DayHigh                  float64 `json:"dayHigh,omitempty"`
	RegularMarketPreviousClose float64 `json:"regularMarketPreviousClose,omitempty"`
	RegularMarketOpen        float64 `json:"regularMarketOpen,omitempty"`
	RegularMarketDayLow      float64 `json:"regularMarketDayLow,omitempty"`
	RegularMarketDayHigh     float64 `json:"regularMarketDayHigh,omitempty"`
	DividendRate             float64 `json:"dividendRate,omitempty"`
	DividendYield            float64 `json:"dividendYield,omitempty"`
	ExDividendDate           int64   `json:"exDividendDate,omitempty"`
	PayoutRatio              float64 `json:"payoutRatio,omitempty"`
	FiveYearAvgDividendYield float64 `json:"fiveYearAvgDividendYield,omitempty"`
	Beta5Y                   float64 `json:"beta,omitempty"`
	TrailingPE               float64 `json:"trailingPE,omitempty"`
	Volume                   int64   `json:"volume,omitempty"`
	RegularMarketVolume      int64   `json:"regularMarketVolume,omitempty"`
	AverageVolume            int64   `json:"averageVolume,omitempty"`
	AverageVolume10Days      int64   `json:"averageVolume10days,omitempty"`
	AverageDailyVolume10Day  int64   `json:"averageDailyVolume10Day,omitempty"`
	Bid                      float64 `json:"bid,omitempty"`
	Ask                      float64 `json:"ask,omitempty"`
	BidSize                  int64   `json:"bidSize,omitempty"`
	AskSize                  int64   `json:"askSize,omitempty"`
	MarketCap                int64   `json:"marketCap,omitempty"`
	FiftyTwoWeekLow          float64 `json:"fiftyTwoWeekLow,omitempty"`
	FiftyTwoWeekHigh         float64 `json:"fiftyTwoWeekHigh,omitempty"`
	PriceToSalesTrailing12Mo float64 `json:"priceToSalesTrailing12Months,omitempty"`
	FiftyDayAverage          float64 `json:"fiftyDayAverage,omitempty"`
	TwoHundredDayAverage     float64 `json:"twoHundredDayAverage,omitempty"`
	TrailingAnnualDividendRate float64 `json:"trailingAnnualDividendRate,omitempty"`
	TrailingAnnualDividendYield float64 `json:"trailingAnnualDividendYield,omitempty"`

	// Financial data (financialData module)
	CurrentPrice              float64 `json:"currentPrice,omitempty"`
	TargetHighPrice           float64 `json:"targetHighPrice,omitempty"`
	TargetLowPrice            float64 `json:"targetLowPrice,omitempty"`
	TargetMeanPrice           float64 `json:"targetMeanPrice,omitempty"`
	TargetMedianPrice         float64 `json:"targetMedianPrice,omitempty"`
	RecommendationMean        float64 `json:"recommendationMean,omitempty"`
	RecommendationKey         string  `json:"recommendationKey,omitempty"`
	NumberOfAnalystOpinions   int     `json:"numberOfAnalystOpinions,omitempty"`
	TotalCash                 int64   `json:"totalCash,omitempty"`
	TotalCashPerShare         float64 `json:"totalCashPerShare,omitempty"`
	Ebitda                    int64   `json:"ebitda,omitempty"`
	TotalDebt                 int64   `json:"totalDebt,omitempty"`
	QuickRatio                float64 `json:"quickRatio,omitempty"`
	CurrentRatio              float64 `json:"currentRatio,omitempty"`
	TotalRevenue              int64   `json:"totalRevenue,omitempty"`
	DebtToEquity              float64 `json:"debtToEquity,omitempty"`
	RevenuePerShare           float64 `json:"revenuePerShare,omitempty"`
	ReturnOnAssets            float64 `json:"returnOnAssets,omitempty"`
	ReturnOnEquity            float64 `json:"returnOnEquity,omitempty"`
	GrossProfits              int64   `json:"grossProfits,omitempty"`
	FreeCashflow              int64   `json:"freeCashflow,omitempty"`
	OperatingCashflow         int64   `json:"operatingCashflow,omitempty"`
	EarningsGrowth            float64 `json:"earningsGrowth,omitempty"`
	RevenueGrowth             float64 `json:"revenueGrowth,omitempty"`
	GrossMargins              float64 `json:"grossMargins,omitempty"`
	EbitdaMargins             float64 `json:"ebitdaMargins,omitempty"`
	OperatingMargins          float64 `json:"operatingMargins,omitempty"`
	FinancialCurrency         string  `json:"financialCurrency,omitempty"`

	// Trailing PEG (from timeseries API)
	TrailingPegRatio float64 `json:"trailingPegRatio,omitempty"`
}

// Officer represents a company officer.
type Officer struct {
	MaxAge           int     `json:"maxAge,omitempty"`
	Name             string  `json:"name"`
	Age              int     `json:"age,omitempty"`
	Title            string  `json:"title"`
	YearBorn         int     `json:"yearBorn,omitempty"`
	FiscalYear       int     `json:"fiscalYear,omitempty"`
	TotalPay         int64   `json:"totalPay,omitempty"`
	ExercisedValue   int64   `json:"exercisedValue,omitempty"`
	UnexercisedValue int64   `json:"unexercisedValue,omitempty"`
}
