// Package utils provides utility functions for yfinance.
package utils

// MICToYahooSuffix maps ISO 10383 Market Identifier Codes (MIC) to Yahoo Finance ticker suffixes.
// This allows converting standard exchange codes to the format used by Yahoo Finance.
//
// Reference:
//   - Yahoo Finance suffixes: https://help.yahoo.com/kb/finance-for-web/SLN2310.html
//   - ISO 10383 MIC codes: https://www.iso20022.org/market-identifier-codes
//
// Example: "XLON" (London Stock Exchange) maps to "L", so AAPL traded on LSE would be "AAPL.L"
var MICToYahooSuffix = map[string]string{
	// United States
	"XCBT": "CBT",  // Chicago Board of Trade
	"XCME": "CME",  // Chicago Mercantile Exchange
	"IFUS": "NYB",  // ICE Futures US
	"CECS": "CMX",  // COMEX (Commodities Exchange)
	"XNYM": "NYM",  // New York Mercantile Exchange
	"XNYS": "",     // New York Stock Exchange (no suffix for US)
	"XNAS": "",     // NASDAQ (no suffix for US)

	// Argentina
	"XBUE": "BA", // Buenos Aires Stock Exchange

	// Austria
	"XVIE": "VI", // Vienna Stock Exchange

	// Australia
	"XASX": "AX", // Australian Securities Exchange
	"XAUS": "XA", // National Stock Exchange of Australia

	// Belgium
	"XBRU": "BR", // Euronext Brussels

	// Brazil
	"BVMF": "SA", // B3 (Brasil Bolsa Balcão)

	// Canada
	"CNSX": "CN", // Canadian Securities Exchange
	"NEOE": "NE", // Aequitas NEO Exchange
	"XTSE": "TO", // Toronto Stock Exchange
	"XTSX": "V",  // TSX Venture Exchange

	// Chile
	"XSGO": "SN", // Santiago Stock Exchange

	// China
	"XSHG": "SS", // Shanghai Stock Exchange
	"XSHE": "SZ", // Shenzhen Stock Exchange

	// Colombia
	"XBOG": "CL", // Colombia Stock Exchange

	// Czech Republic
	"XPRA": "PR", // Prague Stock Exchange

	// Denmark
	"XCSE": "CO", // Nasdaq Copenhagen

	// Egypt
	"XCAI": "CA", // Egyptian Exchange

	// Estonia
	"XTAL": "TL", // Nasdaq Tallinn

	// Europe (Multi-national)
	"CEUX": "XD", // Cboe Europe
	"XEUR": "NX", // Euronext

	// Finland
	"XHEL": "HE", // Nasdaq Helsinki

	// France
	"XPAR": "PA", // Euronext Paris

	// Germany
	"XBER": "BE", // Berlin Stock Exchange
	"XBMS": "BM", // Boerse München
	"XDUS": "DU", // Düsseldorf Stock Exchange
	"XFRA": "F",  // Frankfurt Stock Exchange
	"XHAM": "HM", // Hamburg Stock Exchange
	"XHAN": "HA", // Hannover Stock Exchange
	"XMUN": "MU", // Munich Stock Exchange
	"XSTU": "SG", // Stuttgart Stock Exchange
	"XETR": "DE", // XETRA

	// Greece
	"XATH": "AT", // Athens Stock Exchange

	// Hong Kong
	"XHKG": "HK", // Hong Kong Stock Exchange

	// Hungary
	"XBUD": "BD", // Budapest Stock Exchange

	// Iceland
	"XICE": "IC", // Nasdaq Iceland

	// India
	"XBOM": "BO", // Bombay Stock Exchange
	"XNSE": "NS", // National Stock Exchange of India

	// Indonesia
	"XIDX": "JK", // Indonesia Stock Exchange

	// Ireland
	"XDUB": "IR", // Euronext Dublin

	// Israel
	"XTAE": "TA", // Tel Aviv Stock Exchange

	// Italy
	"MTAA": "MI", // Borsa Italiana
	"EUTL": "TI", // EuroTLX

	// Japan
	"XTKS": "T", // Tokyo Stock Exchange

	// Kuwait
	"XKFE": "KW", // Kuwait Stock Exchange

	// Latvia
	"XRIS": "RG", // Nasdaq Riga

	// Lithuania
	"XVIL": "VS", // Nasdaq Vilnius

	// Malaysia
	"XKLS": "KL", // Bursa Malaysia

	// Mexico
	"XMEX": "MX", // Mexican Stock Exchange

	// Netherlands
	"XAMS": "AS", // Euronext Amsterdam

	// New Zealand
	"XNZE": "NZ", // New Zealand Stock Exchange

	// Norway
	"XOSL": "OL", // Oslo Stock Exchange

	// Philippines
	"XPHS": "PS", // Philippine Stock Exchange

	// Poland
	"XWAR": "WA", // Warsaw Stock Exchange

	// Portugal
	"XLIS": "LS", // Euronext Lisbon

	// Qatar
	"XQAT": "QA", // Qatar Stock Exchange

	// Romania
	"XBSE": "RO", // Bucharest Stock Exchange

	// Singapore
	"XSES": "SI", // Singapore Exchange

	// South Africa
	"XJSE": "JO", // Johannesburg Stock Exchange

	// South Korea
	"XKRX": "KS", // Korea Exchange
	"KQKS": "KQ", // KOSDAQ

	// Spain
	"BMEX": "MC", // BME Spanish Exchanges

	// Saudi Arabia
	"XSAU": "SR", // Saudi Stock Exchange (Tadawul)

	// Sweden
	"XSTO": "ST", // Nasdaq Stockholm

	// Switzerland
	"XSWX": "SW", // SIX Swiss Exchange

	// Taiwan
	"ROCO": "TWO", // Taipei Exchange (TPEX)
	"XTAI": "TW",  // Taiwan Stock Exchange

	// Thailand
	"XBKK": "BK", // Stock Exchange of Thailand

	// Turkey
	"XIST": "IS", // Borsa Istanbul

	// United Arab Emirates
	"XDFM": "AE", // Dubai Financial Market

	// United Kingdom
	"AQXE": "AQ", // Aquis Exchange
	"XCHI": "XC", // Chi-X Europe
	"XLON": "L",  // London Stock Exchange
	"ILSE": "IL", // Irish Stock Exchange (legacy code)

	// Venezuela
	"XCAR": "CR", // Caracas Stock Exchange

	// Vietnam
	"XSTC": "VN", // Ho Chi Minh Stock Exchange
}

// YahooSuffixToMIC is the reverse mapping from Yahoo Finance suffix to MIC code.
// Note: Some Yahoo suffixes may map to multiple MIC codes. In such cases,
// the primary/most common exchange is used.
var YahooSuffixToMIC map[string]string

func init() {
	// Build reverse mapping
	YahooSuffixToMIC = make(map[string]string)
	for mic, suffix := range MICToYahooSuffix {
		if suffix != "" {
			// For duplicates, keep the first one (or could implement priority)
			if _, exists := YahooSuffixToMIC[suffix]; !exists {
				YahooSuffixToMIC[suffix] = mic
			}
		}
	}
}

// GetYahooSuffix returns the Yahoo Finance ticker suffix for a given MIC code.
// Returns an empty string if the MIC code is not found or maps to a US exchange.
func GetYahooSuffix(mic string) string {
	if suffix, ok := MICToYahooSuffix[mic]; ok {
		return suffix
	}
	return ""
}

// GetMIC returns the MIC code for a given Yahoo Finance ticker suffix.
// Returns an empty string if the suffix is not found.
func GetMIC(suffix string) string {
	if mic, ok := YahooSuffixToMIC[suffix]; ok {
		return mic
	}
	return ""
}

// FormatYahooTicker formats a base ticker symbol with the appropriate Yahoo Finance suffix
// for the given MIC code.
//
// Example:
//
//	FormatYahooTicker("AAPL", "XLON") returns "AAPL.L"
//	FormatYahooTicker("AAPL", "XNYS") returns "AAPL" (no suffix for US)
func FormatYahooTicker(baseTicker, mic string) string {
	suffix := GetYahooSuffix(mic)
	if suffix == "" {
		return baseTicker
	}
	return baseTicker + "." + suffix
}

// ParseYahooTicker parses a Yahoo Finance ticker into base ticker and suffix.
//
// Example:
//
//	ParseYahooTicker("AAPL.L") returns ("AAPL", "L")
//	ParseYahooTicker("AAPL") returns ("AAPL", "")
func ParseYahooTicker(ticker string) (baseTicker, suffix string) {
	for i := len(ticker) - 1; i >= 0; i-- {
		if ticker[i] == '.' {
			return ticker[:i], ticker[i+1:]
		}
	}
	return ticker, ""
}

// IsUSExchange returns true if the MIC code represents a US exchange.
func IsUSExchange(mic string) bool {
	switch mic {
	case "XCBT", "XCME", "IFUS", "CECS", "XNYM", "XNYS", "XNAS":
		return true
	default:
		return false
	}
}

// AllMICs returns all supported MIC codes.
func AllMICs() []string {
	mics := make([]string, 0, len(MICToYahooSuffix))
	for mic := range MICToYahooSuffix {
		mics = append(mics, mic)
	}
	return mics
}

// AllYahooSuffixes returns all supported Yahoo Finance suffixes (excluding empty for US).
func AllYahooSuffixes() []string {
	suffixes := make([]string, 0, len(YahooSuffixToMIC))
	for suffix := range YahooSuffixToMIC {
		suffixes = append(suffixes, suffix)
	}
	return suffixes
}
