package models

// SectorIndustryMapping maps sector names to their industries.
// Matches Python's SECTOR_INDUSTY_MAPPING from yfinance/const.py.
var SectorIndustryMapping = map[string][]string{
	"Basic Materials": {
		"Specialty Chemicals", "Gold", "Building Materials", "Copper", "Steel",
		"Agricultural Inputs", "Chemicals", "Other Industrial Metals & Mining",
		"Lumber & Wood Production", "Aluminum", "Other Precious Metals & Mining",
		"Coking Coal", "Paper & Paper Products", "Silver",
	},
	"Communication Services": {
		"Advertising Agencies", "Broadcasting", "Electronic Gaming & Multimedia",
		"Entertainment", "Internet Content & Information", "Publishing", "Telecom Services",
	},
	"Consumer Cyclical": {
		"Apparel Manufacturing", "Apparel Retail", "Auto & Truck Dealerships",
		"Auto Manufacturers", "Auto Parts", "Department Stores", "Footwear & Accessories",
		"Furnishings, Fixtures & Appliances", "Gambling", "Home Improvement Retail",
		"Internet Retail", "Leisure", "Lodging", "Luxury Goods",
		"Packaging & Containers", "Personal Services", "Recreational Vehicles",
		"Residential Construction", "Resorts & Casinos", "Restaurants",
		"Specialty Retail", "Textile Manufacturing", "Travel Services",
	},
	"Consumer Defensive": {
		"Beverages—Brewers", "Beverages—Non-Alcoholic", "Beverages—Wineries & Distilleries",
		"Confectioners", "Discount Stores", "Education & Training Services",
		"Farm Products", "Food Distribution", "Grocery Stores",
		"Household & Personal Products", "Packaged Foods", "Tobacco",
	},
	"Energy": {
		"Oil & Gas Drilling", "Oil & Gas E&P", "Oil & Gas Equipment & Services",
		"Oil & Gas Integrated", "Oil & Gas Midstream", "Oil & Gas Refining & Marketing",
		"Thermal Coal", "Uranium",
	},
	"Financial Services": {
		"Asset Management", "Banks—Diversified", "Banks—Regional", "Capital Markets",
		"Credit Services", "Financial Conglomerates", "Financial Data & Stock Exchanges",
		"Insurance Brokers", "Insurance—Diversified", "Insurance—Life",
		"Insurance—Property & Casualty", "Insurance—Reinsurance", "Insurance—Specialty",
		"Mortgage Finance", "Shell Companies",
	},
	"Healthcare": {
		"Biotechnology", "Diagnostics & Research", "Drug Manufacturers—General",
		"Drug Manufacturers—Specialty & Generic", "Health Information Services",
		"Healthcare Plans", "Medical Care Facilities", "Medical Devices",
		"Medical Instruments & Supplies", "Medical Distribution", "Pharmaceutical Retailers",
	},
	"Industrials": {
		"Aerospace & Defense", "Airlines", "Airports & Air Services",
		"Building Products & Equipment", "Business Equipment & Supplies", "Conglomerates",
		"Consulting Services", "Electrical Equipment & Parts", "Engineering & Construction",
		"Farm & Heavy Construction Machinery", "Industrial Distribution",
		"Infrastructure Operations", "Integrated Freight & Logistics", "Marine Shipping",
		"Metal Fabrication", "Pollution & Treatment Controls", "Railroads",
		"Rental & Leasing Services", "Security & Protection Services",
		"Specialty Business Services", "Specialty Industrial Machinery",
		"Staffing & Employment Services", "Tools & Accessories", "Trucking", "Waste Management",
	},
	"Real Estate": {
		"Real Estate—Development", "Real Estate Services", "Real Estate—Diversified",
		"REIT—Healthcare Facilities", "REIT—Hotel & Motel", "REIT—Industrial",
		"REIT—Office", "REIT—Residential", "REIT—Retail", "REIT—Mortgage",
		"REIT—Specialty", "REIT—Diversified",
	},
	"Technology": {
		"Communication Equipment", "Computer Hardware", "Consumer Electronics",
		"Electronic Components", "Electronics & Computer Distribution",
		"Information Technology Services", "Scientific & Technical Instruments",
		"Semiconductor Equipment & Materials", "Semiconductors",
		"Software—Application", "Software—Infrastructure", "Solar",
	},
	"Utilities": {
		"Utilities—Diversified", "Utilities—Independent Power Producers",
		"Utilities—Regulated Electric", "Utilities—Regulated Gas",
		"Utilities—Regulated Water", "Utilities—Renewable",
	},
}

// EquityScreenerExchangeMap maps region codes to valid exchange codes for equity screener.
// Matches Python's EQUITY_SCREENER_EQ_MAP["exchange"].
var EquityScreenerExchangeMap = map[string][]string{
	"ae": {"DFM"},
	"ar": {"BUE"},
	"at": {"VIE"},
	"au": {"ASX", "CXA"},
	"be": {"BRU"},
	"br": {"SAO"},
	"ca": {"CNQ", "NEO", "TOR", "VAN"},
	"ch": {"EBS"},
	"cl": {"SGO"},
	"cn": {"SHH", "SHZ"},
	"co": {"BVC"},
	"cz": {"PRA"},
	"de": {"BER", "DUS", "EUX", "FRA", "HAM", "HAN", "GER", "MUN", "STU"},
	"dk": {"CPH"},
	"ee": {"TAL"},
	"eg": {"CAI"},
	"es": {"MAD", "MCE"},
	"fi": {"HEL"},
	"fr": {"ENX", "PAR"},
	"gb": {"AQS", "CXE", "IOB", "LSE"},
	"gr": {"ATH"},
	"hk": {"HKG"},
	"hu": {"BUD"},
	"id": {"JKT"},
	"ie": {"ISE"},
	"il": {"TLV"},
	"in": {"BSE", "NSI"},
	"is": {"ICE"},
	"it": {"MDD", "MIL", "TLO"},
	"jp": {"FKA", "JPX", "OSA", "SAP"},
	"kr": {"KOE", "KSC"},
	"kw": {"KUW"},
	"lk": {"CSE"},
	"lt": {"LIT"},
	"lv": {"RIS"},
	"mx": {"MEX"},
	"my": {"KLS"},
	"nl": {"AMS", "DXE"},
	"no": {"OSL"},
	"nz": {"NZE"},
	"pe": {},
	"ph": {"PHP", "PHS"},
	"pk": {"KAR"},
	"pl": {"WSE"},
	"pt": {"LIS"},
	"qa": {"DOH"},
	"ro": {"BVB"},
	"ru": {"MCX"},
	"sa": {"SAU"},
	"se": {"STO"},
	"sg": {"SES"},
	"sr": {},
	"th": {"SET"},
	"tr": {"IST"},
	"tw": {"TAI", "TWO"},
	"us": {"ASE", "BTS", "CXI", "NAE", "NCM", "NGM", "NMS", "NYQ", "OEM", "OQB", "OQX", "PCX", "PNK", "YHD"},
	"ve": {"CCS"},
	"vn": {"VSE"},
	"za": {"JNB"},
}

// FundScreenerExchangeMap maps region codes to valid exchange codes for fund screener.
// Matches Python's FUND_SCREENER_EQ_MAP["exchange"].
var FundScreenerExchangeMap = map[string][]string{
	"ae": {"DFM"},
	"ar": {"BUE"},
	"at": {"VIE"},
	"au": {"ASX", "CXA"},
	"be": {"BRU"},
	"br": {"SAO"},
	"ca": {"CNQ", "NEO", "TOR", "VAN"},
	"ch": {"EBS"},
	"cl": {"SGO"},
	"co": {"BVC"},
	"cn": {"SHH", "SHZ"},
	"cz": {"PRA"},
	"de": {"BER", "DUS", "EUX", "FRA", "GER", "HAM", "HAN", "MUN", "STU"},
	"dk": {"CPH"},
	"ee": {"TAL"},
	"eg": {"CAI"},
	"es": {"BAR", "MAD", "MCE"},
	"fi": {"HEL"},
	"fr": {"ENX", "PAR"},
	"gb": {"CXE", "IOB", "LSE"},
	"gr": {"ATH"},
	"hk": {"HKG"},
	"hu": {"BUD"},
	"id": {"JKT"},
	"ie": {"ISE"},
	"il": {"TLV"},
	"in": {"BSE", "NSI"},
	"is": {"ICE"},
	"it": {"MIL"},
	"jp": {"FKA", "JPX", "OSA", "SAP"},
	"kr": {"KOE", "KSC"},
	"kw": {"KUW"},
	"lk": {"CSE"},
	"lt": {"LIT"},
	"lv": {"RIS"},
	"mx": {"MEX"},
	"my": {"KLS"},
	"nl": {"AMS"},
	"no": {"OSL"},
	"nz": {"NZE"},
	"pe": {""},
	"ph": {"PHP", "PHS"},
	"pk": {"KAR"},
	"pl": {"WSE"},
	"pt": {"LIS"},
	"qa": {"DOH"},
	"ro": {"BVB"},
	"ru": {"MCX"},
	"sa": {"SAU"},
	"se": {"STO"},
	"sg": {"SES"},
	"sr": {""},
	"th": {"SET"},
	"tr": {"IST"},
	"tw": {"TAI", "TWO"},
	"us": {"ASE", "NAS", "NCM", "NGM", "NMS", "NYQ", "OEM", "OGM", "OQB", "PNK", "WCB"},
	"ve": {"CCS"},
	"vn": {"VSE"},
	"za": {"JNB"},
}

// EquityScreenerSectors is the set of valid sector names for equity screener.
var EquityScreenerSectors = []string{
	"Basic Materials", "Industrials", "Communication Services", "Healthcare",
	"Real Estate", "Technology", "Energy", "Utilities", "Financial Services",
	"Consumer Defensive", "Consumer Cyclical",
}

// EquityScreenerPeerGroups is the set of valid peer group names for equity screener.
var EquityScreenerPeerGroups = []string{
	"US Fund Equity Energy", "US CE Convertibles", "EAA CE UK Large-Cap Equity",
	"EAA CE Other", "US Fund Financial", "India CE Multi-Cap",
	"US Fund Foreign Large Blend", "US Fund Consumer Cyclical",
	"EAA Fund Global Equity Income",
	"China Fund Sector Equity Financial and Real Estate",
	"US Fund Equity Precious Metals", "EAA Fund RMB Bond - Onshore",
	"China Fund QDII Greater China Equity", "US Fund Large Growth",
	"EAA Fund Germany Equity", "EAA Fund Hong Kong Equity",
	"EAA CE UK Small-Cap Equity", "US Fund Natural Resources",
	"US CE Preferred Stock", "India Fund Sector - Financial Services",
	"US Fund Diversified Emerging Mkts",
	"EAA Fund South Africa & Namibia Equity",
	"China Fund QDII Sector Equity", "EAA CE Sector Equity Biotechnology",
	"EAA Fund Switzerland Equity", "US Fund Large Value",
	"EAA Fund Asia ex-Japan Equity", "US Fund Health", "US Fund China Region",
	"EAA Fund Emerging Europe ex-Russia Equity",
	"EAA Fund Sector Equity Industrial Materials",
	"EAA Fund Japan Large-Cap Equity", "EAA Fund EUR Corporate Bond",
	"US Fund Technology", "EAA CE Global Large-Cap Blend Equity",
	"Mexico Fund Mexico Equity", "US Fund Trading--Leveraged Equity",
	"EAA Fund Sector Equity Consumer Goods & Services", "US Fund Large Blend",
	"EAA Fund Global Flex-Cap Equity",
	"EAA Fund EUR Aggressive Allocation - Global", "EAA Fund China Equity",
	"EAA Fund Global Large-Cap Growth Equity", "US CE Options-based",
	"EAA Fund Sector Equity Financial Services",
	"EAA Fund Europe Large-Cap Blend Equity",
	"EAA Fund China Equity - A Shares", "EAA Fund USD Corporate Bond",
	"EAA Fund Eurozone Large-Cap Equity",
	"China Fund Aggressive Allocation Fund",
	"EAA Fund Sector Equity Technology",
	"EAA Fund Global Emerging Markets Equity",
	"EAA Fund EUR Moderate Allocation - Global", "EAA Fund Other Bond",
	"EAA Fund Denmark Equity", "EAA Fund US Large-Cap Blend Equity",
	"India Fund Large-Cap", "Paper & Forestry", "Containers & Packaging",
	"US Fund Miscellaneous Region", "Energy Services", "EAA Fund Other Equity",
	"Homebuilders", "Construction Materials", "China Fund Equity Funds",
	"Steel", "Consumer Durables", "EAA Fund Global Large-Cap Blend Equity",
	"Transportation Infrastructure", "Precious Metals", "Building Products",
	"Traders & Distributors", "Electrical Equipment", "Auto Components",
	"Construction & Engineering", "Aerospace & Defense",
	"Refiners & Pipelines", "Diversified Metals", "Textiles & Apparel",
	"Industrial Conglomerates", "Household Products", "Commercial Services",
	"Food Retailers", "Semiconductors", "Media", "Automobiles",
	"Consumer Services", "Technology Hardware", "Transportation",
	"Telecommunication Services", "Oil & Gas Producers", "Machinery",
	"Retailing", "Healthcare", "Chemicals", "Food Products",
	"Diversified Financials", "Real Estate", "Insurance", "Utilities",
	"Pharmaceuticals", "Software & Services", "Banks",
}

// EquityScreenerFields defines valid field names by category for equity screener.
// After merging with CommonScreenerFields, matches Python's EQUITY_SCREENER_FIELDS.
var EquityScreenerFields = map[string][]string{
	"eq_fields": {"region", "sector", "peer_group", "industry", "exchange"},
	"price": {
		"lastclosemarketcap.lasttwelvemonths", "percentchange",
		"lastclose52weekhigh.lasttwelvemonths", "fiftytwowkpercentchange",
		"lastclose52weeklow.lasttwelvemonths", "intradaymarketcap",
		"eodprice", "intradaypricechange", "intradayprice",
	},
	"trading": {"beta", "avgdailyvol3m", "pctheldinsider", "pctheldinst", "dayvolume", "eodvolume"},
	"short_interest": {
		"short_percentage_of_shares_outstanding.value", "short_interest.value",
		"short_percentage_of_float.value", "days_to_cover_short.value",
		"short_interest_percentage_change.value",
	},
	"valuation": {
		"bookvalueshare.lasttwelvemonths", "lastclosemarketcaptotalrevenue.lasttwelvemonths",
		"lastclosetevtotalrevenue.lasttwelvemonths", "pricebookratio.quarterly",
		"peratio.lasttwelvemonths", "lastclosepricetangiblebookvalue.lasttwelvemonths",
		"lastclosepriceearnings.lasttwelvemonths", "pegratio_5y",
	},
	"profitability": {
		"consecutive_years_of_dividend_growth_count", "returnonassets.lasttwelvemonths",
		"returnonequity.lasttwelvemonths", "forward_dividend_per_share",
		"forward_dividend_yield", "returnontotalcapital.lasttwelvemonths",
	},
	"leverage": {
		"lastclosetevebit.lasttwelvemonths", "netdebtebitda.lasttwelvemonths",
		"totaldebtequity.lasttwelvemonths", "ltdebtequity.lasttwelvemonths",
		"ebitinterestexpense.lasttwelvemonths", "ebitdainterestexpense.lasttwelvemonths",
		"lastclosetevebitda.lasttwelvemonths", "totaldebtebitda.lasttwelvemonths",
	},
	"liquidity": {
		"quickratio.lasttwelvemonths",
		"altmanzscoreusingtheaveragestockinformationforaperiod.lasttwelvemonths",
		"currentratio.lasttwelvemonths",
		"operatingcashflowtocurrentliabilities.lasttwelvemonths",
	},
	"income_statement": {
		"totalrevenues.lasttwelvemonths", "netincomemargin.lasttwelvemonths",
		"grossprofit.lasttwelvemonths", "ebitda1yrgrowth.lasttwelvemonths",
		"dilutedepscontinuingoperations.lasttwelvemonths", "quarterlyrevenuegrowth.quarterly",
		"epsgrowth.lasttwelvemonths", "netincomeis.lasttwelvemonths",
		"ebitda.lasttwelvemonths", "dilutedeps1yrgrowth.lasttwelvemonths",
		"totalrevenues1yrgrowth.lasttwelvemonths", "operatingincome.lasttwelvemonths",
		"netincome1yrgrowth.lasttwelvemonths", "grossprofitmargin.lasttwelvemonths",
		"ebitdamargin.lasttwelvemonths", "ebit.lasttwelvemonths",
		"basicepscontinuingoperations.lasttwelvemonths",
		"netepsbasic.lasttwelvemonthsnetepsdiluted.lasttwelvemonths",
	},
	"balance_sheet": {
		"totalassets.lasttwelvemonths", "totalcommonsharesoutstanding.lasttwelvemonths",
		"totaldebt.lasttwelvemonths", "totalequity.lasttwelvemonths",
		"totalcurrentassets.lasttwelvemonths",
		"totalcashandshortterminvestments.lasttwelvemonths",
		"totalcommonequity.lasttwelvemonths",
		"totalcurrentliabilities.lasttwelvemonths", "totalsharesoutstanding",
	},
	"cash_flow": {
		"forward_dividend_yield", "leveredfreecashflow.lasttwelvemonths",
		"capitalexpenditure.lasttwelvemonths", "cashfromoperations.lasttwelvemonths",
		"leveredfreecashflow1yrgrowth.lasttwelvemonths",
		"unleveredfreecashflow.lasttwelvemonths",
		"cashfromoperations1yrgrowth.lasttwelvemonths",
	},
	"esg": {"esg_score", "environmental_score", "governance_score", "social_score", "highest_controversy"},
}

// FundScreenerFields defines valid field names by category for fund screener.
// After merging with CommonScreenerFields, matches Python's FUND_SCREENER_FIELDS.
var FundScreenerFields = map[string][]string{
	"eq_fields": {
		"categoryname", "performanceratingoverall", "initialinvestment",
		"annualreturnnavy1categoryrank", "riskratingoverall", "exchange",
	},
	"price": {"eodprice", "intradaypricechange", "intradayprice"},
}

// allEquityValidFields returns a flattened set of all valid equity screener field names.
func allEquityValidFields() map[string]bool {
	result := make(map[string]bool)
	for _, fields := range EquityScreenerFields {
		for _, f := range fields {
			result[f] = true
		}
	}
	return result
}

// allFundValidFields returns a flattened set of all valid fund screener field names.
func allFundValidFields() map[string]bool {
	result := make(map[string]bool)
	for _, fields := range FundScreenerFields {
		for _, f := range fields {
			result[f] = true
		}
	}
	return result
}

// allExchangeCodes returns a flattened set of all exchange codes from the given exchange map.
func allExchangeCodes(exchangeMap map[string][]string) map[string]bool {
	result := make(map[string]bool)
	for _, codes := range exchangeMap {
		for _, code := range codes {
			if code != "" {
				result[code] = true
			}
		}
	}
	return result
}

// allIndustries returns a flattened set of all industry names from SectorIndustryMapping.
func allIndustries() map[string]bool {
	result := make(map[string]bool)
	for _, industries := range SectorIndustryMapping {
		for _, ind := range industries {
			result[ind] = true
		}
	}
	return result
}

// regionCodes returns a set of all region codes from the given exchange map.
func regionCodes(exchangeMap map[string][]string) map[string]bool {
	result := make(map[string]bool)
	for region := range exchangeMap {
		result[region] = true
	}
	return result
}
