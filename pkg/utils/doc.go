// Package utils provides utility functions for go-yfinance.
//
// # Overview
//
// The utils package contains helper functions for common operations
// such as timezone handling and exchange-specific utilities.
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
