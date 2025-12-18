package ticker

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/wnjoon/go-yfinance/pkg/client"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

// Info fetches comprehensive company information for the ticker.
func (t *Ticker) Info() (*models.Info, error) {
	// Check cache first
	t.mu.RLock()
	if t.infoCache != nil {
		info := t.infoCache
		t.mu.RUnlock()
		return info, nil
	}
	t.mu.RUnlock()

	// Fetch from API
	modules := []string{
		"assetProfile",
		"summaryDetail",
		"defaultKeyStatistics",
		"financialData",
		"quoteType",
	}

	params := url.Values{}
	params.Set("modules", joinModules(modules))
	params.Set("corsDomain", "finance.yahoo.com")
	params.Set("formatted", "false")

	resp, err := t.getWithCrumb(t.quoteSummaryURL(), params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch info: %w", err)
	}

	var summaryResp models.QuoteSummaryResponse
	if err := json.Unmarshal([]byte(resp.Body), &summaryResp); err != nil {
		return nil, client.WrapInvalidResponseError(err)
	}

	if summaryResp.QuoteSummary.Error != nil {
		return nil, fmt.Errorf("API error: %s", summaryResp.QuoteSummary.Error.Description)
	}

	if len(summaryResp.QuoteSummary.Result) == 0 {
		return nil, client.WrapNotFoundError(t.symbol)
	}

	result := summaryResp.QuoteSummary.Result[0]
	info := t.parseInfo(&result)

	// Cache the info
	t.mu.Lock()
	t.infoCache = info
	t.mu.Unlock()

	return info, nil
}

// parseInfo converts the raw quoteSummary response to Info struct.
func (t *Ticker) parseInfo(result *models.QuoteSummaryResult) *models.Info {
	info := &models.Info{
		Symbol: t.symbol,
	}

	// Parse assetProfile
	if profile := result.AssetProfile; profile != nil {
		info.Sector = getString(profile, "sector")
		info.Industry = getString(profile, "industry")
		info.FullTimeEmployees = getInt64(profile, "fullTimeEmployees")
		info.City = getString(profile, "city")
		info.State = getString(profile, "state")
		info.Country = getString(profile, "country")
		info.Phone = getString(profile, "phone")
		info.Address1 = getString(profile, "address1")
		info.Address2 = getString(profile, "address2")
		info.Zip = getString(profile, "zip")
		info.Website = getString(profile, "website")
		info.LongBusinessSummary = getString(profile, "longBusinessSummary")
		info.AuditRisk = getInt(profile, "auditRisk")
		info.BoardRisk = getInt(profile, "boardRisk")
		info.CompensationRisk = getInt(profile, "compensationRisk")
		info.ShareHolderRightsRisk = getInt(profile, "shareHolderRightsRisk")
		info.OverallRisk = getInt(profile, "overallRisk")

		// Parse company officers
		if officers, ok := profile["companyOfficers"].([]interface{}); ok {
			for _, o := range officers {
				if officerMap, ok := o.(map[string]interface{}); ok {
					officer := models.Officer{
						Name:             getString(officerMap, "name"),
						Title:            getString(officerMap, "title"),
						Age:              getInt(officerMap, "age"),
						YearBorn:         getInt(officerMap, "yearBorn"),
						FiscalYear:       getInt(officerMap, "fiscalYear"),
						TotalPay:         getInt64(officerMap, "totalPay"),
						ExercisedValue:   getInt64(officerMap, "exercisedValue"),
						UnexercisedValue: getInt64(officerMap, "unexercisedValue"),
					}
					info.CompanyOfficers = append(info.CompanyOfficers, officer)
				}
			}
		}
	}

	// Parse quoteType
	if qt := result.QuoteType; qt != nil {
		info.ShortName = getString(qt, "shortName")
		info.LongName = getString(qt, "longName")
		info.QuoteType = getString(qt, "quoteType")
		info.Exchange = getString(qt, "exchange")
		info.ExchangeTimezoneName = getString(qt, "exchangeTimezoneName")
		info.ExchangeTimezoneShort = getString(qt, "exchangeTimezoneShortName")
		info.Currency = getString(qt, "currency")
		info.Market = getString(qt, "market")
		info.FirstTradeDateEpoch = getInt64(qt, "firstTradeDateEpochUtc")
	}

	// Parse defaultKeyStatistics
	if stats := result.DefaultKeyStatistics; stats != nil {
		info.EnterpriseValue = getInt64(stats, "enterpriseValue")
		info.ForwardPE = getFloat64(stats, "forwardPE")
		info.ProfitMargins = getFloat64(stats, "profitMargins")
		info.FloatShares = getInt64(stats, "floatShares")
		info.SharesOutstanding = getInt64(stats, "sharesOutstanding")
		info.SharesShort = getInt64(stats, "sharesShort")
		info.SharesShortPriorMonth = getInt64(stats, "sharesShortPriorMonth")
		info.DateShortInterest = getInt64(stats, "dateShortInterest")
		info.SharesPercentSharesOut = getFloat64(stats, "sharesPercentSharesOut")
		info.HeldPercentInsiders = getFloat64(stats, "heldPercentInsiders")
		info.HeldPercentInstitutions = getFloat64(stats, "heldPercentInstitutions")
		info.ShortRatio = getFloat64(stats, "shortRatio")
		info.ShortPercentOfFloat = getFloat64(stats, "shortPercentOfFloat")
		info.Beta = getFloat64(stats, "beta")
		info.BookValue = getFloat64(stats, "bookValue")
		info.PriceToBook = getFloat64(stats, "priceToBook")
		info.LastFiscalYearEnd = getInt64(stats, "lastFiscalYearEnd")
		info.NextFiscalYearEnd = getInt64(stats, "nextFiscalYearEnd")
		info.MostRecentQuarter = getInt64(stats, "mostRecentQuarter")
		info.EarningsQuarterlyGrowth = getFloat64(stats, "earningsQuarterlyGrowth")
		info.NetIncomeToCommon = getInt64(stats, "netIncomeToCommon")
		info.TrailingEps = getFloat64(stats, "trailingEps")
		info.ForwardEps = getFloat64(stats, "forwardEps")
		info.PegRatio = getFloat64(stats, "pegRatio")
		info.LastSplitFactor = getString(stats, "lastSplitFactor")
		info.LastSplitDate = getInt64(stats, "lastSplitDate")
		info.EnterpriseToRevenue = getFloat64(stats, "enterpriseToRevenue")
		info.EnterpriseToEbitda = getFloat64(stats, "enterpriseToEbitda")
		info.FiftyTwoWeekChange = getFloat64(stats, "52WeekChange")
		info.SandP52WeekChange = getFloat64(stats, "SandP52WeekChange")
		info.LastDividendValue = getFloat64(stats, "lastDividendValue")
		info.LastDividendDate = getInt64(stats, "lastDividendDate")
	}

	// Parse summaryDetail
	if detail := result.SummaryDetail; detail != nil {
		info.PreviousClose = getFloat64(detail, "previousClose")
		info.Open = getFloat64(detail, "open")
		info.DayLow = getFloat64(detail, "dayLow")
		info.DayHigh = getFloat64(detail, "dayHigh")
		info.RegularMarketPreviousClose = getFloat64(detail, "regularMarketPreviousClose")
		info.RegularMarketOpen = getFloat64(detail, "regularMarketOpen")
		info.RegularMarketDayLow = getFloat64(detail, "regularMarketDayLow")
		info.RegularMarketDayHigh = getFloat64(detail, "regularMarketDayHigh")
		info.DividendRate = getFloat64(detail, "dividendRate")
		info.DividendYield = getFloat64(detail, "dividendYield")
		info.ExDividendDate = getInt64(detail, "exDividendDate")
		info.PayoutRatio = getFloat64(detail, "payoutRatio")
		info.FiveYearAvgDividendYield = getFloat64(detail, "fiveYearAvgDividendYield")
		info.TrailingPE = getFloat64(detail, "trailingPE")
		info.Volume = getInt64(detail, "volume")
		info.RegularMarketVolume = getInt64(detail, "regularMarketVolume")
		info.AverageVolume = getInt64(detail, "averageVolume")
		info.AverageVolume10Days = getInt64(detail, "averageVolume10days")
		info.AverageDailyVolume10Day = getInt64(detail, "averageDailyVolume10Day")
		info.Bid = getFloat64(detail, "bid")
		info.Ask = getFloat64(detail, "ask")
		info.BidSize = getInt64(detail, "bidSize")
		info.AskSize = getInt64(detail, "askSize")
		info.MarketCap = getInt64(detail, "marketCap")
		info.FiftyTwoWeekLow = getFloat64(detail, "fiftyTwoWeekLow")
		info.FiftyTwoWeekHigh = getFloat64(detail, "fiftyTwoWeekHigh")
		info.PriceToSalesTrailing12Mo = getFloat64(detail, "priceToSalesTrailing12Months")
		info.FiftyDayAverage = getFloat64(detail, "fiftyDayAverage")
		info.TwoHundredDayAverage = getFloat64(detail, "twoHundredDayAverage")
		info.TrailingAnnualDividendRate = getFloat64(detail, "trailingAnnualDividendRate")
		info.TrailingAnnualDividendYield = getFloat64(detail, "trailingAnnualDividendYield")
	}

	// Parse financialData
	if fin := result.FinancialData; fin != nil {
		info.CurrentPrice = getFloat64(fin, "currentPrice")
		info.TargetHighPrice = getFloat64(fin, "targetHighPrice")
		info.TargetLowPrice = getFloat64(fin, "targetLowPrice")
		info.TargetMeanPrice = getFloat64(fin, "targetMeanPrice")
		info.TargetMedianPrice = getFloat64(fin, "targetMedianPrice")
		info.RecommendationMean = getFloat64(fin, "recommendationMean")
		info.RecommendationKey = getString(fin, "recommendationKey")
		info.NumberOfAnalystOpinions = getInt(fin, "numberOfAnalystOpinions")
		info.TotalCash = getInt64(fin, "totalCash")
		info.TotalCashPerShare = getFloat64(fin, "totalCashPerShare")
		info.Ebitda = getInt64(fin, "ebitda")
		info.TotalDebt = getInt64(fin, "totalDebt")
		info.QuickRatio = getFloat64(fin, "quickRatio")
		info.CurrentRatio = getFloat64(fin, "currentRatio")
		info.TotalRevenue = getInt64(fin, "totalRevenue")
		info.DebtToEquity = getFloat64(fin, "debtToEquity")
		info.RevenuePerShare = getFloat64(fin, "revenuePerShare")
		info.ReturnOnAssets = getFloat64(fin, "returnOnAssets")
		info.ReturnOnEquity = getFloat64(fin, "returnOnEquity")
		info.GrossProfits = getInt64(fin, "grossProfits")
		info.FreeCashflow = getInt64(fin, "freeCashflow")
		info.OperatingCashflow = getInt64(fin, "operatingCashflow")
		info.EarningsGrowth = getFloat64(fin, "earningsGrowth")
		info.RevenueGrowth = getFloat64(fin, "revenueGrowth")
		info.GrossMargins = getFloat64(fin, "grossMargins")
		info.EbitdaMargins = getFloat64(fin, "ebitdaMargins")
		info.OperatingMargins = getFloat64(fin, "operatingMargins")
		info.FinancialCurrency = getString(fin, "financialCurrency")
	}

	return info
}

// Helper functions for safe type conversion

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getFloat64(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case int64:
			return float64(val)
		case int:
			return float64(val)
		case map[string]interface{}:
			// Handle Yahoo's format like {"raw": 123.45, "fmt": "123.45"}
			if raw, ok := val["raw"]; ok {
				if f, ok := raw.(float64); ok {
					return f
				}
			}
		}
	}
	return 0
}

func getInt64(m map[string]interface{}, key string) int64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return int64(val)
		case int64:
			return val
		case int:
			return int64(val)
		case map[string]interface{}:
			if raw, ok := val["raw"]; ok {
				if f, ok := raw.(float64); ok {
					return int64(f)
				}
			}
		}
	}
	return 0
}

func getInt(m map[string]interface{}, key string) int {
	return int(getInt64(m, key))
}

// joinModules joins module names with comma.
func joinModules(modules []string) string {
	result := ""
	for i, m := range modules {
		if i > 0 {
			result += ","
		}
		result += m
	}
	return result
}
