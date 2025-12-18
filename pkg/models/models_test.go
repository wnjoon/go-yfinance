package models

import (
	"testing"
	"time"
)

func TestDefaultHistoryParams(t *testing.T) {
	params := DefaultHistoryParams()

	if params.Period != "1mo" {
		t.Errorf("Default period should be '1mo', got '%s'", params.Period)
	}
	if params.Interval != "1d" {
		t.Errorf("Default interval should be '1d', got '%s'", params.Interval)
	}
	if params.PrePost {
		t.Error("PrePost should be false by default")
	}
	if !params.AutoAdjust {
		t.Error("AutoAdjust should be true by default")
	}
	if !params.Actions {
		t.Error("Actions should be true by default")
	}
}

func TestIsValidPeriod(t *testing.T) {
	validPeriods := []string{"1d", "5d", "1mo", "3mo", "6mo", "1y", "2y", "5y", "10y", "ytd", "max"}
	for _, p := range validPeriods {
		if !IsValidPeriod(p) {
			t.Errorf("Period '%s' should be valid", p)
		}
	}

	invalidPeriods := []string{"1w", "2mo", "invalid", ""}
	for _, p := range invalidPeriods {
		if IsValidPeriod(p) {
			t.Errorf("Period '%s' should be invalid", p)
		}
	}
}

func TestIsValidInterval(t *testing.T) {
	validIntervals := []string{"1m", "2m", "5m", "15m", "30m", "60m", "90m", "1h", "1d", "5d", "1wk", "1mo", "3mo"}
	for _, i := range validIntervals {
		if !IsValidInterval(i) {
			t.Errorf("Interval '%s' should be valid", i)
		}
	}

	invalidIntervals := []string{"10m", "4h", "2d", "invalid", ""}
	for _, i := range invalidIntervals {
		if IsValidInterval(i) {
			t.Errorf("Interval '%s' should be invalid", i)
		}
	}
}

func TestBarStruct(t *testing.T) {
	now := time.Now()
	bar := Bar{
		Date:      now,
		Open:      100.0,
		High:      105.0,
		Low:       99.0,
		Close:     103.0,
		AdjClose:  103.0,
		Volume:    1000000,
		Dividends: 0.5,
		Splits:    0,
	}

	if bar.Date != now {
		t.Error("Bar date mismatch")
	}
	if bar.Open != 100.0 {
		t.Error("Bar open mismatch")
	}
	if bar.High != 105.0 {
		t.Error("Bar high mismatch")
	}
	if bar.Low != 99.0 {
		t.Error("Bar low mismatch")
	}
	if bar.Close != 103.0 {
		t.Error("Bar close mismatch")
	}
	if bar.Volume != 1000000 {
		t.Error("Bar volume mismatch")
	}
}

func TestQuoteStruct(t *testing.T) {
	quote := Quote{
		Symbol:                     "AAPL",
		ShortName:                  "Apple Inc.",
		Exchange:                   "NMS",
		Currency:                   "USD",
		RegularMarketPrice:         150.0,
		RegularMarketChange:        2.5,
		RegularMarketChangePercent: 1.69,
		MarketState:                "REGULAR",
	}

	if quote.Symbol != "AAPL" {
		t.Error("Quote symbol mismatch")
	}
	if quote.RegularMarketPrice != 150.0 {
		t.Error("Quote price mismatch")
	}
	if quote.Currency != "USD" {
		t.Error("Quote currency mismatch")
	}
}

func TestInfoStruct(t *testing.T) {
	info := Info{
		Symbol:            "AAPL",
		ShortName:         "Apple Inc.",
		LongName:          "Apple Inc.",
		Sector:            "Technology",
		Industry:          "Consumer Electronics",
		FullTimeEmployees: 164000,
		Country:           "United States",
		MarketCap:         2500000000000,
	}

	if info.Symbol != "AAPL" {
		t.Error("Info symbol mismatch")
	}
	if info.Sector != "Technology" {
		t.Error("Info sector mismatch")
	}
	if info.MarketCap != 2500000000000 {
		t.Error("Info market cap mismatch")
	}
}

func TestValidPeriods(t *testing.T) {
	periods := ValidPeriods()
	if len(periods) != 11 {
		t.Errorf("Expected 11 valid periods, got %d", len(periods))
	}
}

func TestValidIntervals(t *testing.T) {
	intervals := ValidIntervals()
	if len(intervals) != 13 {
		t.Errorf("Expected 13 valid intervals, got %d", len(intervals))
	}
}
