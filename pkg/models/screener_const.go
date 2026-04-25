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

// ETFScreenerExchangeMap maps region codes to valid exchange codes for ETF screener.
// Matches Python's ETF_SCREENER_EQ_MAP["exchange"] from yfinance v1.3.0.
var ETFScreenerExchangeMap = map[string][]string{
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

// ETFScreenerCategories is the set of valid categoryname values for ETF screener.
var ETFScreenerCategories = []string{
	"Allocation--15% to 30% Equity", "Allocation--30% to 50% Equity",
	"Allocation--50% to 70% Equity", "Allocation--70% to 85% Equity",
	"Allocation--85%+ Equity", "Bank Loan", "Bear Market", "China Region",
	"Commodities Agriculture", "Commodities Broad Basket", "Convertibles",
	"Corporate Bond", "Diversified Emerging Mkts", "Diversified Pacific/Asia",
	"Emerging Markets Bond", "Emerging-Markets Local-Currency Bond",
	"Energy Limited Partnership", "Equity Energy", "Equity Precious Metals",
	"Europe Stock", "Financial", "Foreign Large Blend", "Foreign Large Growth",
	"Foreign Large Value", "Foreign Small/Mid Blend", "Foreign Small/Mid Growth",
	"Foreign Small/Mid Value", "Global Real Estate", "Health", "High Yield Bond",
	"High Yield Muni", "Inflation-Protected Bond", "Infrastructure",
	"Intermediate Government", "Intermediate-Term Bond", "Japan Stock",
	"Large Blend", "Large Growth", "Large Value", "Long Government",
	"Long-Short Credit", "Long-Short Equity", "Long-Term Bond", "Managed Futures",
	"Market Neutral", "Mid-Cap Blend", "Mid-Cap Growth", "Mid-Cap Value",
	"Miscellaneous Region", "Multialternative", "Multicurrency",
	"Multisector Bond", "Muni California Intermediate", "Muni California Long",
	"Muni Massachusetts", "Muni Minnesota", "Muni National Interm",
	"Muni National Long", "Muni National Short", "Muni New Jersey",
	"Muni New York Intermediate", "Muni New York Long", "Muni Ohio",
	"Muni Pennsylvania", "Muni Single State Interm", "Muni Single State Long",
	"Muni Single State Short", "Natural Resources", "Nontraditional Bond",
	"Option Writing", "Other", "Other Allocation", "Pacific/Asia ex-Japan Stk",
	"Preferred Stock", "Real Estate", "Short Government", "Short-Term Bond",
	"Small Blend", "Small Growth", "Small Value", "Tactical Allocation",
	"Target-Date 2000-2010", "Target-Date 2015", "Target-Date 2020",
	"Target-Date 2025", "Target-Date 2030", "Target-Date 2035",
	"Target-Date 2040", "Target-Date 2045", "Target-Date 2050",
	"Target-Date 2055", "Target-Date 2060+", "Target-Date Retirement",
	"Technology", "Trading - Leveraged/Inverse Commodities",
	"Trading - Leveraged/Inverse Equity", "Trading--Inverse Equity",
	"Trading--Leveraged Equity", "Ultrashort Bond", "Utilities",
	"World Allocation", "World Bond", "World Stock",
}

// ETFScreenerFundFamilies is the set of valid fundfamilyname values for ETF screener.
var ETFScreenerFundFamilies = []string{
	"ALPS", "AMG Funds", "AQR Funds", "Aberdeen", "Alger", "AllianceBernstein",
	"Allianz Funds", "American Beacon", "American Century Investments",
	"American Funds", "Aquila", "Artisan", "BMO Funds", "BNY Mellon Funds",
	"Baird", "Barclays Funds", "Barings Funds", "Baron Capital Group",
	"BlackRock", "Brown Advisory Funds", "Calamos", "Calvert Investments",
	"Catalyst Mutual Funds", "Cohen & Steers", "Columbia",
	"Commerz Funds Solutions SA", "Commerzbank AG, Frankfurt am Main",
	"Davis Funds", "Delaware Investments", "Deutsche Asset Management",
	"Deutsche Bank AG", "Diamond Hill Funds", "Dimensional Fund Advisors",
	"Direxion Funds", "DoubleLine", "Dreyfus", "Dunham Funds", "Eagle Funds",
	"Eaton Vance", "Federated", "Fidelity Investments", "First Investors",
	"First Trust", "Flexshares Trust", "Franklin Templeton Investments", "GMO",
	"Gabelli", "Global X Funds", "Goldman Sachs", "Great-West Funds",
	"Guggenheim Investments", "GuideStone Funds", "HSBC", "Hancock Horizon",
	"Harbor", "Hartford Mutual Funds", "Henderson Global", "Hennessy",
	"Highland Funds", "ICON Funds", "Invesco", "Ivy Funds", "JPMorgan",
	"Janus", "John Hancock", "Lazard", "Legg Mason", "Lord Abbett", "MFS",
	"Madison Funds", "MainStay", "Manning & Napier", "Market Vectors",
	"MassMutual", "Matthews Asia Funds", "Morgan Stanley", "Nationwide",
	"Natixis Funds", "Neuberger Berman", "Northern Funds", "Nuveen",
	"OppenheimerFunds", "PNC Funds", "Pacific funds series trust", "Pax World",
	"Paydenfunds", "Pimco", "Pioneer Investments", "PowerShares",
	"Principal Funds", "ProFunds", "ProShares", "Prudential Investments",
	"Putnam", "RBC Global Asset Management.", "RidgeWorth", "Royce", "Russell",
	"Rydex Funds", "SEI", "SPDR State Street Global Advisors", "Salient Funds",
	"Saratoga", "Schwab Funds", "Sentinel", "Shelton Capital Management",
	"State Farm", "State Street Global Advisors (Chicago)", "Sterling Capital Funds",
	"SunAmerica", "T. Rowe Price", "TCW", "TIAA-CREF Asset Management",
	"Teton Westwood Funds", "Thornburg", "Thrivent", "Timothy Plan", "Touchstone",
	"Transamerica", "UBS", "UBS Group AG", "USAA", "VALIC", "Vanguard",
	"Vantagepoint Funds", "Victory", "Virtus", "Voya", "Waddell & Reed",
	"Wasatch", "Wells Fargo Funds", "William Blair", "WisdomTree", "iShares",
}

var (
	ETFScreenerEconomicMoats = []string{"Wide", "Narrow", "None"}
	ETFScreenerStewardship   = []string{"Exemplary", "Standard", "Poor"}
	ETFScreenerUncertainty   = []string{"Low", "Medium", "High", "Very High", "Extreme"}
	ETFScreenerMoatTrend     = []string{"Stable", "Positive", "Negative"}
	ETFScreenerRatingChange  = []string{"Upgrade", "Downgrade"}
)

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

// ETFScreenerFields defines valid field names by category for ETF screener.
// After merging with CommonScreenerFields, matches Python's ETF_SCREENER_FIELDS.
var ETFScreenerFields = map[string][]string{
	"eq_fields": {
		"categoryname", "fundfamilyname", "region", "primary_sector",
		"morningstar_economic_moat", "morningstar_stewardship",
		"morningstar_uncertainty", "morningstar_moat_trend",
		"morningstar_rating_change",
	},
	"fundamentals": {"fundnetassets", "ticker"},
	"feesandexpenses": {
		"annualreportgrossexpenseratio", "annualreportnetexpenseratio", "turnoverratio",
	},
	"historicalperformance": {
		"annualreturnnavy1", "annualreturnnavy1categoryrank",
		"annualreturnnavy3", "annualreturnnavy5",
	},
	"keystats": {
		"avgdailyvol3m", "dayvolume", "eodvolume", "fiftytwowkpercentchange",
		"percentchange",
	},
	"morningstar_rating": {
		"morningstar_last_close_price_to_fair_value", "morningstar_rating",
		"morningstar_rating_updated_time",
	},
	"portfoliostatistics": {"marketcapitalvaluelong"},
	"purchasedetails":     {"initialinvestment"},
	"trailingperformance": {
		"performanceratingoverall", "quarterendtrailingreturnytd",
		"riskratingoverall", "trailing_3m_return", "trailing_ytd_return",
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

// allETFValidFields returns a flattened set of all valid ETF screener field names.
func allETFValidFields() map[string]bool {
	result := make(map[string]bool)
	for _, fields := range ETFScreenerFields {
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
