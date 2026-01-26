package utils

import (
	"testing"
)

func TestGetYahooSuffix(t *testing.T) {
	tests := []struct {
		mic      string
		expected string
	}{
		{"XLON", "L"},      // London Stock Exchange
		{"XNYS", ""},       // NYSE (no suffix)
		{"XNAS", ""},       // NASDAQ (no suffix)
		{"XTKS", "T"},      // Tokyo Stock Exchange
		{"XHKG", "HK"},     // Hong Kong
		{"XASX", "AX"},     // Australia
		{"XTSE", "TO"},     // Toronto
		{"XPAR", "PA"},     // Paris
		{"XETR", "DE"},     // XETRA Germany
		{"XSAU", "SR"},     // Saudi Arabia
		{"INVALID", ""},    // Unknown MIC
	}

	for _, tt := range tests {
		t.Run(tt.mic, func(t *testing.T) {
			result := GetYahooSuffix(tt.mic)
			if result != tt.expected {
				t.Errorf("GetYahooSuffix(%q) = %q, want %q", tt.mic, result, tt.expected)
			}
		})
	}
}

func TestGetMIC(t *testing.T) {
	tests := []struct {
		suffix   string
		expected string
	}{
		{"L", "XLON"},   // London
		{"T", "XTKS"},   // Tokyo
		{"HK", "XHKG"},  // Hong Kong
		{"AX", "XASX"},  // Australia
		{"TO", "XTSE"},  // Toronto
		{"PA", "XPAR"},  // Paris
		{"DE", "XETR"},  // Germany
		{"SR", "XSAU"},  // Saudi Arabia
		{"ZZ", ""},      // Unknown suffix
	}

	for _, tt := range tests {
		t.Run(tt.suffix, func(t *testing.T) {
			result := GetMIC(tt.suffix)
			if result != tt.expected {
				t.Errorf("GetMIC(%q) = %q, want %q", tt.suffix, result, tt.expected)
			}
		})
	}
}

func TestFormatYahooTicker(t *testing.T) {
	tests := []struct {
		baseTicker string
		mic        string
		expected   string
	}{
		{"AAPL", "XNYS", "AAPL"},       // US NYSE - no suffix
		{"AAPL", "XNAS", "AAPL"},       // US NASDAQ - no suffix
		{"AAPL", "XLON", "AAPL.L"},     // London
		{"7203", "XTKS", "7203.T"},     // Toyota on Tokyo
		{"0700", "XHKG", "0700.HK"},    // Tencent on Hong Kong
		{"CBA", "XASX", "CBA.AX"},      // Commonwealth Bank on ASX
		{"TD", "XTSE", "TD.TO"},        // TD Bank on Toronto
		{"SAP", "XETR", "SAP.DE"},      // SAP on XETRA
		{"UNKNOWN", "INVALID", "UNKNOWN"}, // Invalid MIC - no suffix
	}

	for _, tt := range tests {
		name := tt.baseTicker + "_" + tt.mic
		t.Run(name, func(t *testing.T) {
			result := FormatYahooTicker(tt.baseTicker, tt.mic)
			if result != tt.expected {
				t.Errorf("FormatYahooTicker(%q, %q) = %q, want %q",
					tt.baseTicker, tt.mic, result, tt.expected)
			}
		})
	}
}

func TestParseYahooTicker(t *testing.T) {
	tests := []struct {
		ticker         string
		expectedBase   string
		expectedSuffix string
	}{
		{"AAPL", "AAPL", ""},
		{"AAPL.L", "AAPL", "L"},
		{"7203.T", "7203", "T"},
		{"0700.HK", "0700", "HK"},
		{"SAP.DE", "SAP", "DE"},
		{"BRK.A", "BRK", "A"},              // Berkshire class A
		{"BRK.B", "BRK", "B"},              // Berkshire class B
		{"TD.TO", "TD", "TO"},
		{"NO.DOTS", "NO", "DOTS"},          // Last dot wins
		{"A.B.C", "A.B", "C"},              // Multiple dots
	}

	for _, tt := range tests {
		t.Run(tt.ticker, func(t *testing.T) {
			base, suffix := ParseYahooTicker(tt.ticker)
			if base != tt.expectedBase || suffix != tt.expectedSuffix {
				t.Errorf("ParseYahooTicker(%q) = (%q, %q), want (%q, %q)",
					tt.ticker, base, suffix, tt.expectedBase, tt.expectedSuffix)
			}
		})
	}
}

func TestIsUSExchange(t *testing.T) {
	tests := []struct {
		mic      string
		expected bool
	}{
		{"XNYS", true},   // NYSE
		{"XNAS", true},   // NASDAQ
		{"XCBT", true},   // Chicago Board of Trade
		{"XCME", true},   // Chicago Mercantile Exchange
		{"XNYM", true},   // NYMEX
		{"XLON", false},  // London
		{"XHKG", false},  // Hong Kong
		{"XTKS", false},  // Tokyo
		{"INVALID", false},
	}

	for _, tt := range tests {
		t.Run(tt.mic, func(t *testing.T) {
			result := IsUSExchange(tt.mic)
			if result != tt.expected {
				t.Errorf("IsUSExchange(%q) = %v, want %v", tt.mic, result, tt.expected)
			}
		})
	}
}

func TestAllMICs(t *testing.T) {
	mics := AllMICs()

	// Should have a reasonable number of MICs
	if len(mics) < 50 {
		t.Errorf("Expected at least 50 MICs, got %d", len(mics))
	}

	// Check that known MICs are present
	knownMICs := []string{"XNYS", "XNAS", "XLON", "XTKS", "XHKG"}
	for _, known := range knownMICs {
		found := false
		for _, mic := range mics {
			if mic == known {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected MIC %q in AllMICs()", known)
		}
	}
}

func TestAllYahooSuffixes(t *testing.T) {
	suffixes := AllYahooSuffixes()

	// Should have suffixes (US exchanges have empty suffix, so less than MICs)
	if len(suffixes) < 40 {
		t.Errorf("Expected at least 40 suffixes, got %d", len(suffixes))
	}

	// Check that known suffixes are present
	knownSuffixes := []string{"L", "T", "HK", "AX", "TO"}
	for _, known := range knownSuffixes {
		found := false
		for _, suffix := range suffixes {
			if suffix == known {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected suffix %q in AllYahooSuffixes()", known)
		}
	}
}

func TestMICToYahooSuffixConsistency(t *testing.T) {
	// Test that the mapping is complete and consistent

	// Check all MICs have valid entries (even if empty string for US)
	for mic, suffix := range MICToYahooSuffix {
		if mic == "" {
			t.Error("Empty MIC code in mapping")
		}
		// suffix can be empty for US exchanges
		_ = suffix
	}

	// Check reverse mapping works for non-empty suffixes
	for mic, suffix := range MICToYahooSuffix {
		if suffix != "" {
			// The reverse lookup should give back a valid MIC
			// (may not be the same MIC due to duplicates)
			reverseMIC := GetMIC(suffix)
			if reverseMIC == "" {
				t.Errorf("Reverse lookup failed for suffix %q (from MIC %q)", suffix, mic)
			}
		}
	}
}

func TestRoundTrip(t *testing.T) {
	// Test formatting and parsing round-trip
	testCases := []struct {
		baseTicker string
		mic        string
	}{
		{"AAPL", "XLON"},
		{"7203", "XTKS"},
		{"0700", "XHKG"},
		{"CBA", "XASX"},
		{"TD", "XTSE"},
	}

	for _, tc := range testCases {
		t.Run(tc.baseTicker+"_"+tc.mic, func(t *testing.T) {
			// Format the ticker
			formatted := FormatYahooTicker(tc.baseTicker, tc.mic)

			// Parse it back
			parsedBase, parsedSuffix := ParseYahooTicker(formatted)

			// Base should match
			if parsedBase != tc.baseTicker {
				t.Errorf("Round-trip base mismatch: got %q, want %q", parsedBase, tc.baseTicker)
			}

			// Suffix should match what we expect from the MIC
			expectedSuffix := GetYahooSuffix(tc.mic)
			if parsedSuffix != expectedSuffix {
				t.Errorf("Round-trip suffix mismatch: got %q, want %q", parsedSuffix, expectedSuffix)
			}
		})
	}
}
