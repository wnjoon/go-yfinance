// Package utils provides utility functions for go-yfinance.
//
// # Overview
//
// The utils package contains helper functions for common operations
// such as timezone handling, MIC code mapping, and exchange-specific utilities.
//
// # MIC Code Mapping (v1.1.0)
//
// ISO 10383 Market Identifier Code (MIC) to Yahoo Finance suffix mapping:
//
//	suffix := utils.GetYahooSuffix("XLON")     // Returns "L"
//	mic := utils.GetMIC("T")                   // Returns "XTKS"
//	ticker := utils.FormatYahooTicker("7203", "XTKS")  // Returns "7203.T"
//	base, suffix := utils.ParseYahooTicker("AAPL.L")   // Returns "AAPL", "L"
//
// Check if an exchange is US-based:
//
//	if utils.IsUSExchange("XNYS") {
//	    // NYSE - no suffix needed
//	}
//
// List all supported exchanges:
//
//	mics := utils.AllMICs()           // []string{"XNYS", "XNAS", "XLON", ...}
//	suffixes := utils.AllYahooSuffixes()  // []string{"", "L", "T", ...}
//
// # Timezone Functions
//
// Exchange timezone lookup:
//
//	tz := utils.GetTimezone("NYQ")  // Returns "America/New_York"
//	tz := utils.GetTimezone("TYO")  // Returns "Asia/Tokyo"
//
// Timezone validation and conversion:
//
//	if utils.IsValidTimezone("America/New_York") {
//	    loc := utils.LoadLocation("America/New_York")
//	    t := utils.ConvertToTimezone(time.Now(), "America/New_York")
//	}
//
// # Exchange Mappings
//
// The package includes timezone mappings for major exchanges:
//   - US: NYSE (NYQ), NASDAQ (NMS, NGM, NCM), BATS, OTC
//   - Europe: London (LSE), Frankfurt (FRA), Paris (PAR), etc.
//   - Asia: Tokyo (TYO), Hong Kong (HKG), Shanghai (SHH), etc.
//   - Others: TSX (Toronto), ASX (Sydney), etc.
//
// # Market Hours
//
// Basic market hours check (simplified, no holiday support):
//
//	if utils.MarketIsOpen("NYQ") {
//	    fmt.Println("NYSE is open")
//	}
//
// # Thread Safety
//
// All utility functions are thread-safe.
package utils
