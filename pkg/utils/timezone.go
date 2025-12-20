// Package utils provides utility functions for go-yfinance.
package utils

import (
	"time"

	"github.com/wnjoon/go-yfinance/pkg/cache"
)

// Timezone cache key prefix
const tzCachePrefix = "tz:"

// Common exchange timezones
var exchangeTimezones = map[string]string{
	// US exchanges
	"NYQ":  "America/New_York", // NYSE
	"NMS":  "America/New_York", // NASDAQ
	"NGM":  "America/New_York", // NASDAQ Global Market
	"NCM":  "America/New_York", // NASDAQ Capital Market
	"NYS":  "America/New_York", // NYSE
	"PCX":  "America/New_York", // NYSE Arca
	"ASE":  "America/New_York", // NYSE American
	"BTS":  "America/New_York", // BATS
	"PNK":  "America/New_York", // Pink Sheets
	"OTC":  "America/New_York", // OTC
	"OTCM": "America/New_York", // OTC Markets

	// European exchanges
	"LSE": "Europe/London",    // London
	"IOB": "Europe/London",    // LSE International Order Book
	"FRA": "Europe/Berlin",    // Frankfurt
	"ETR": "Europe/Berlin",    // XETRA
	"PAR": "Europe/Paris",     // Paris
	"AMS": "Europe/Amsterdam", // Amsterdam
	"BRU": "Europe/Brussels",  // Brussels
	"MIL": "Europe/Rome",      // Milan
	"MCE": "Europe/Madrid",    // Madrid
	"SIX": "Europe/Zurich",    // Swiss Exchange
	"VIE": "Europe/Vienna",    // Vienna

	// Asian exchanges
	"TYO": "Asia/Tokyo",         // Tokyo
	"JPX": "Asia/Tokyo",         // Japan Exchange
	"HKG": "Asia/Hong_Kong",     // Hong Kong
	"SHH": "Asia/Shanghai",      // Shanghai
	"SHZ": "Asia/Shanghai",      // Shenzhen
	"KSC": "Asia/Seoul",         // Korea (KOSPI)
	"KOE": "Asia/Seoul",         // Korea (KOSDAQ)
	"SGX": "Asia/Singapore",     // Singapore
	"BOM": "Asia/Kolkata",       // Mumbai BSE
	"NSI": "Asia/Kolkata",       // Mumbai NSE
	"TAI": "Asia/Taipei",        // Taiwan
	"TWO": "Asia/Taipei",        // Taiwan OTC
	"JKT": "Asia/Jakarta",       // Jakarta
	"KLS": "Asia/Kuala_Lumpur",  // Kuala Lumpur
	"BKK": "Asia/Bangkok",       // Bangkok

	// Oceania
	"ASX": "Australia/Sydney", // Australian Securities Exchange
	"NZE": "Pacific/Auckland", // New Zealand

	// Americas (non-US)
	"TSX": "America/Toronto",     // Toronto
	"CVE": "America/Toronto",     // TSX Venture
	"NEO": "America/Toronto",     // NEO Exchange
	"MEX": "America/Mexico_City", // Mexico
	"SAO": "America/Sao_Paulo",   // Brazil
	"BUE": "America/Argentina/Buenos_Aires", // Buenos Aires

	// Crypto - 24/7
	"CCC": "UTC", // Crypto
	"CME": "America/Chicago", // CME (futures)

	// Middle East / Africa
	"TLV": "Asia/Jerusalem",       // Tel Aviv
	"JSE": "Africa/Johannesburg",  // Johannesburg
	"DFM": "Asia/Dubai",           // Dubai
}

// GetTimezone returns the timezone for a given exchange code.
// It first checks the cache, then falls back to the known exchange timezones.
// Returns "America/New_York" as default if exchange is unknown.
func GetTimezone(exchange string) string {
	// Check cache first
	cacheKey := tzCachePrefix + exchange
	if tz, ok := cache.GetGlobalString(cacheKey); ok {
		return tz
	}

	// Look up in known exchanges
	if tz, ok := exchangeTimezones[exchange]; ok {
		cache.SetGlobal(cacheKey, tz)
		return tz
	}

	// Default to New York
	return "America/New_York"
}

// CacheTimezone stores a timezone for an exchange in the cache.
func CacheTimezone(exchange, timezone string) {
	cache.SetGlobal(tzCachePrefix+exchange, timezone)
}

// LoadLocation loads a timezone location by name.
// Returns nil if the timezone is invalid.
func LoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil
	}
	return loc
}

// IsValidTimezone checks if a timezone name is valid.
func IsValidTimezone(name string) bool {
	_, err := time.LoadLocation(name)
	return err == nil
}

// ConvertToTimezone converts a UTC time to the specified timezone.
func ConvertToTimezone(t time.Time, timezone string) time.Time {
	loc := LoadLocation(timezone)
	if loc == nil {
		return t
	}
	return t.In(loc)
}

// ParseTimestamp parses a Unix timestamp and converts to the specified timezone.
func ParseTimestamp(timestamp int64, timezone string) time.Time {
	t := time.Unix(timestamp, 0).UTC()
	return ConvertToTimezone(t, timezone)
}

// MarketIsOpen checks if a market is currently open based on typical trading hours.
// This is a simplified check and doesn't account for holidays.
func MarketIsOpen(exchange string) bool {
	tz := GetTimezone(exchange)
	loc := LoadLocation(tz)
	if loc == nil {
		return false
	}

	now := time.Now().In(loc)

	// Check if weekend
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return false
	}

	// Typical market hours: 9:30 AM - 4:00 PM local time
	hour := now.Hour()
	minute := now.Minute()
	timeValue := hour*60 + minute

	// 9:30 AM = 570 minutes, 4:00 PM = 960 minutes
	return timeValue >= 570 && timeValue < 960
}
