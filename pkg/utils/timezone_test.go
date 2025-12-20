package utils

import (
	"testing"
	"time"
)

func TestGetTimezone(t *testing.T) {
	tests := []struct {
		exchange string
		expected string
	}{
		{"NYQ", "America/New_York"},
		{"NMS", "America/New_York"},
		{"LSE", "Europe/London"},
		{"TYO", "Asia/Tokyo"},
		{"HKG", "Asia/Hong_Kong"},
		{"ASX", "Australia/Sydney"},
		{"TSX", "America/Toronto"},
		{"UNKNOWN", "America/New_York"}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.exchange, func(t *testing.T) {
			got := GetTimezone(tt.exchange)
			if got != tt.expected {
				t.Errorf("GetTimezone(%s) = %s, want %s", tt.exchange, got, tt.expected)
			}
		})
	}
}

func TestCacheTimezone(t *testing.T) {
	CacheTimezone("TEST", "Asia/Seoul")
	got := GetTimezone("TEST")
	if got != "Asia/Seoul" {
		t.Errorf("Expected cached timezone 'Asia/Seoul', got %s", got)
	}
}

func TestLoadLocation(t *testing.T) {
	// Valid timezone
	loc := LoadLocation("America/New_York")
	if loc == nil {
		t.Error("Expected non-nil location for America/New_York")
	}

	// Invalid timezone
	loc = LoadLocation("Invalid/Timezone")
	if loc != nil {
		t.Error("Expected nil location for invalid timezone")
	}
}

func TestIsValidTimezone(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"America/New_York", true},
		{"Europe/London", true},
		{"Asia/Tokyo", true},
		{"UTC", true},
		{"Invalid/Timezone", false},
		{"Not/Real", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidTimezone(tt.name)
			if got != tt.valid {
				t.Errorf("IsValidTimezone(%s) = %v, want %v", tt.name, got, tt.valid)
			}
		})
	}
}

func TestConvertToTimezone(t *testing.T) {
	// Create a UTC time
	utc := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)

	// Convert to New York time (should be 9:30 AM in winter)
	ny := ConvertToTimezone(utc, "America/New_York")
	if ny.Location().String() != "America/New_York" {
		t.Errorf("Expected location America/New_York, got %s", ny.Location().String())
	}

	// Invalid timezone should return original time
	invalid := ConvertToTimezone(utc, "Invalid/Timezone")
	if !invalid.Equal(utc) {
		t.Error("Expected invalid timezone to return original time")
	}
}

func TestParseTimestamp(t *testing.T) {
	// Unix timestamp for 2024-01-15 14:30:00 UTC
	timestamp := int64(1705329000)

	result := ParseTimestamp(timestamp, "America/New_York")
	if result.Location().String() != "America/New_York" {
		t.Errorf("Expected location America/New_York, got %s", result.Location().String())
	}
}

func TestMarketIsOpen(t *testing.T) {
	// This is a basic smoke test since actual result depends on current time
	// Just verify it doesn't panic
	_ = MarketIsOpen("NYQ")
	_ = MarketIsOpen("LSE")
	_ = MarketIsOpen("UNKNOWN")
}
