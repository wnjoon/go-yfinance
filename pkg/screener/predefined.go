package screener

import "github.com/wnjoon/go-yfinance/pkg/models"

// PredefinedQuery holds a predefined screener query with its sort configuration.
type PredefinedQuery struct {
	SortField string
	SortAsc   bool
	Query     models.ScreenerQueryBuilder
}

// PredefinedScreenerQueries maps predefined screener names to their query definitions.
// Matches Python's PREDEFINED_SCREENER_QUERIES from yfinance v1.2.0.
var PredefinedScreenerQueries map[string]PredefinedQuery

func init() {
	PredefinedScreenerQueries = map[string]PredefinedQuery{
		// --- Equity Screeners ---

		"aggressive_small_caps": {
			SortField: "eodvolume",
			SortAsc:   false, // "desc"
			Query: mustEquityQuery("and", []any{
				mustEquityQuery("is-in", []any{"exchange", "NMS", "NYQ"}),
				mustEquityQuery("lt", []any{"epsgrowth.lasttwelvemonths", 15}),
			}),
		},

		"day_gainers": {
			SortField: "percentchange",
			SortAsc:   false, // "DESC"
			Query: mustEquityQuery("and", []any{
				mustEquityQuery("gt", []any{"percentchange", 3}),
				mustEquityQuery("eq", []any{"region", "us"}),
				mustEquityQuery("gte", []any{"intradaymarketcap", 2000000000}),
				mustEquityQuery("gte", []any{"intradayprice", 5}),
				mustEquityQuery("gt", []any{"dayvolume", 15000}),
			}),
		},

		"day_losers": {
			SortField: "percentchange",
			SortAsc:   true, // "ASC"
			Query: mustEquityQuery("and", []any{
				mustEquityQuery("lt", []any{"percentchange", -2.5}),
				mustEquityQuery("eq", []any{"region", "us"}),
				mustEquityQuery("gte", []any{"intradaymarketcap", 2000000000}),
				mustEquityQuery("gte", []any{"intradayprice", 5}),
				mustEquityQuery("gt", []any{"dayvolume", 20000}),
			}),
		},

		"growth_technology_stocks": {
			SortField: "eodvolume",
			SortAsc:   false, // "desc"
			Query: mustEquityQuery("and", []any{
				mustEquityQuery("gte", []any{"quarterlyrevenuegrowth.quarterly", 25}),
				mustEquityQuery("gte", []any{"epsgrowth.lasttwelvemonths", 25}),
				mustEquityQuery("eq", []any{"sector", "Technology"}),
				mustEquityQuery("is-in", []any{"exchange", "NMS", "NYQ"}),
			}),
		},

		"most_actives": {
			SortField: "dayvolume",
			SortAsc:   false, // "DESC"
			Query: mustEquityQuery("and", []any{
				mustEquityQuery("eq", []any{"region", "us"}),
				mustEquityQuery("gte", []any{"intradaymarketcap", 2000000000}),
				mustEquityQuery("gt", []any{"dayvolume", 5000000}),
			}),
		},

		"most_shorted_stocks": {
			SortField: "short_percentage_of_shares_outstanding.value",
			SortAsc:   false, // "DESC"
			Query: mustEquityQuery("and", []any{
				mustEquityQuery("eq", []any{"region", "us"}),
				mustEquityQuery("gt", []any{"intradayprice", 1}),
				mustEquityQuery("gt", []any{"avgdailyvol3m", 200000}),
			}),
		},

		"small_cap_gainers": {
			SortField: "eodvolume",
			SortAsc:   false, // "desc"
			Query: mustEquityQuery("and", []any{
				mustEquityQuery("lt", []any{"intradaymarketcap", 2000000000}),
				mustEquityQuery("is-in", []any{"exchange", "NMS", "NYQ"}),
			}),
		},

		"undervalued_growth_stocks": {
			SortField: "eodvolume",
			SortAsc:   false, // "DESC"
			Query: mustEquityQuery("and", []any{
				mustEquityQuery("btwn", []any{"peratio.lasttwelvemonths", 0, 20}),
				mustEquityQuery("lt", []any{"pegratio_5y", 1}),
				mustEquityQuery("gte", []any{"epsgrowth.lasttwelvemonths", 25}),
				mustEquityQuery("is-in", []any{"exchange", "NMS", "NYQ"}),
			}),
		},

		"undervalued_large_caps": {
			SortField: "eodvolume",
			SortAsc:   false, // "desc"
			Query: mustEquityQuery("and", []any{
				mustEquityQuery("btwn", []any{"peratio.lasttwelvemonths", 0, 20}),
				mustEquityQuery("lt", []any{"pegratio_5y", 1}),
				mustEquityQuery("btwn", []any{"intradaymarketcap", 10000000000, 100000000000}),
				mustEquityQuery("is-in", []any{"exchange", "NMS", "NYQ"}),
			}),
		},

		// --- Fund Screeners ---

		"conservative_foreign_funds": {
			SortField: "fundnetassets",
			SortAsc:   false, // "DESC"
			Query: mustFundQuery("and", []any{
				mustFundQuery("is-in", []any{
					"categoryname",
					"Foreign Large Value", "Foreign Large Blend", "Foreign Large Growth",
					"Foreign Small/Mid Growth", "Foreign Small/Mid Blend", "Foreign Small/Mid Value",
				}),
				mustFundQuery("is-in", []any{"performanceratingoverall", 4, 5}),
				mustFundQuery("lt", []any{"initialinvestment", 100001}),
				mustFundQuery("lt", []any{"annualreturnnavy1categoryrank", 50}),
				mustFundQuery("is-in", []any{"riskratingoverall", 1, 2, 3}),
				mustFundQuery("eq", []any{"exchange", "NAS"}),
			}),
		},

		"high_yield_bond": {
			SortField: "fundnetassets",
			SortAsc:   false, // "DESC"
			Query: mustFundQuery("and", []any{
				mustFundQuery("is-in", []any{"performanceratingoverall", 4, 5}),
				mustFundQuery("lt", []any{"initialinvestment", 100001}),
				mustFundQuery("lt", []any{"annualreturnnavy1categoryrank", 50}),
				mustFundQuery("is-in", []any{"riskratingoverall", 1, 2, 3}),
				mustFundQuery("eq", []any{"categoryname", "High Yield Bond"}),
				mustFundQuery("eq", []any{"exchange", "NAS"}),
			}),
		},

		"portfolio_anchors": {
			SortField: "fundnetassets",
			SortAsc:   false, // "DESC"
			Query: mustFundQuery("and", []any{
				mustFundQuery("eq", []any{"categoryname", "Large Blend"}),
				mustFundQuery("is-in", []any{"performanceratingoverall", 4, 5}),
				mustFundQuery("lt", []any{"initialinvestment", 100001}),
				mustFundQuery("lt", []any{"annualreturnnavy1categoryrank", 50}),
				mustFundQuery("eq", []any{"exchange", "NAS"}),
			}),
		},

		"solid_large_growth_funds": {
			SortField: "fundnetassets",
			SortAsc:   false, // "DESC"
			Query: mustFundQuery("and", []any{
				mustFundQuery("eq", []any{"categoryname", "Large Growth"}),
				mustFundQuery("is-in", []any{"performanceratingoverall", 4, 5}),
				mustFundQuery("lt", []any{"initialinvestment", 100001}),
				mustFundQuery("lt", []any{"annualreturnnavy1categoryrank", 50}),
				mustFundQuery("eq", []any{"exchange", "NAS"}),
			}),
		},

		"solid_midcap_growth_funds": {
			SortField: "fundnetassets",
			SortAsc:   false, // "DESC"
			Query: mustFundQuery("and", []any{
				mustFundQuery("eq", []any{"categoryname", "Mid-Cap Growth"}),
				mustFundQuery("is-in", []any{"performanceratingoverall", 4, 5}),
				mustFundQuery("lt", []any{"initialinvestment", 100001}),
				mustFundQuery("lt", []any{"annualreturnnavy1categoryrank", 50}),
				mustFundQuery("eq", []any{"exchange", "NAS"}),
			}),
		},

		"top_mutual_funds": {
			SortField: "percentchange",
			SortAsc:   false, // "DESC"
			Query: mustFundQuery("and", []any{
				mustFundQuery("gt", []any{"intradayprice", 15}),
				mustFundQuery("is-in", []any{"performanceratingoverall", 4, 5}),
				mustFundQuery("gt", []any{"initialinvestment", 1000}),
				mustFundQuery("eq", []any{"exchange", "NAS"}),
			}),
		},
	}
}

// mustEquityQuery creates an EquityQuery, panicking on error.
// Only for use with known-valid predefined queries at init time.
func mustEquityQuery(op string, operands []any) *models.EquityQuery {
	q, err := models.NewEquityQuery(op, operands)
	if err != nil {
		panic("predefined equity query construction failed: " + err.Error())
	}
	return q
}

// mustFundQuery creates a FundQuery, panicking on error.
// Only for use with known-valid predefined queries at init time.
func mustFundQuery(op string, operands []any) *models.FundQuery {
	q, err := models.NewFundQuery(op, operands)
	if err != nil {
		panic("predefined fund query construction failed: " + err.Error())
	}
	return q
}
