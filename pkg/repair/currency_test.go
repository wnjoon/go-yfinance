package repair

import (
	"math"
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func penceBar(day int, open, high, low, close float64, div float64) models.Bar {
	return models.Bar{
		Date:      time.Date(2024, 1, day, 0, 0, 0, 0, time.UTC),
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		AdjClose:  close,
		Volume:    10000,
		Dividends: div,
	}
}

func closeTo(a, b, rtol float64) bool {
	if a == b {
		return true
	}
	return math.Abs(a-b) <= rtol*math.Max(math.Abs(a), math.Abs(b))
}

func TestStandardiseCurrencyGBpRoundTrip(t *testing.T) {
	bars := []models.Bar{penceBar(1, 6000, 6050, 5950, 6010, 50)}

	scaled, currency, pricesScaled := standardiseCurrency(bars, "GBp")
	if !pricesScaled {
		t.Fatal("expected pricesScaled=true for GBp")
	}
	if currency != "GBP" {
		t.Fatalf("expected standardised currency GBP, got %s", currency)
	}
	if !closeTo(scaled[0].Close, 60.10, 1e-12) {
		t.Errorf("expected Close 60.10 after standardise, got %v", scaled[0].Close)
	}
	if !closeTo(scaled[0].Dividends, 0.50, 1e-12) {
		t.Errorf("expected Dividends 0.50 after standardise, got %v", scaled[0].Dividends)
	}

	reverted := revertCurrency(scaled, "GBp")
	if !closeTo(reverted[0].Close, 6010, 1e-9) {
		t.Errorf("expected Close 6010 after revert, got %v", reverted[0].Close)
	}
	if !closeTo(reverted[0].Dividends, 50, 1e-9) {
		t.Errorf("expected Dividends 50 after revert, got %v", reverted[0].Dividends)
	}
	// Original input untouched
	if bars[0].Close != 6010 {
		t.Errorf("input bars mutated: Close=%v", bars[0].Close)
	}
}

func TestStandardiseCurrencyPassthrough(t *testing.T) {
	bars := []models.Bar{penceBar(1, 100, 105, 95, 102, 0)}

	for _, currency := range []string{"USD", "GBP", "ZAR", "ILS", "KWF", ""} {
		scaled, got, pricesScaled := standardiseCurrency(bars, currency)
		if pricesScaled {
			t.Errorf("%s: expected pricesScaled=false", currency)
		}
		if got != currency {
			t.Errorf("%s: currency changed to %s", currency, got)
		}
		if scaled[0].Close != 102 {
			t.Errorf("%s: prices changed: %v", currency, scaled[0].Close)
		}
	}
}

func TestCurrencySubUnitDivisor(t *testing.T) {
	if d := currencySubUnitDivisor("KWF"); d != 1000.0 {
		t.Errorf("KWF divisor: expected 1000, got %v", d)
	}
	if d := currencySubUnitDivisor("USD"); d != 100.0 {
		t.Errorf("USD divisor: expected 100, got %v", d)
	}
	if d := currencySubUnitDivisor("GBp"); d != 100.0 {
		t.Errorf("GBp divisor: expected 100, got %v", d)
	}
}

// TestRepairGBpCleanDataUnchanged locks upstream #2907's contract: repairing
// clean GBp data must not convert it to GBP.
func TestRepairGBpCleanDataUnchanged(t *testing.T) {
	opts := DefaultOptions()
	opts.Ticker = "XDEV.L"
	opts.Currency = "GBp"
	repairer := New(opts)

	closes := []float64{6000, 6010, 5990, 6020, 5985, 6015, 5995, 6030, 5980, 6005}
	bars := make([]models.Bar, len(closes))
	for i, c := range closes {
		bars[i] = penceBar(i+1, c-5, c+20, c-25, c, 0)
	}

	result, err := repairer.Repair(bars)
	if err != nil {
		t.Fatalf("Repair returned error: %v", err)
	}

	for i := range result {
		if !closeTo(result[i].Close, bars[i].Close, 1e-9) {
			t.Errorf("bar %d: Close changed %v -> %v (must stay in pence)",
				i, bars[i].Close, result[i].Close)
		}
		if !closeTo(result[i].Open, bars[i].Open, 1e-9) {
			t.Errorf("bar %d: Open changed %v -> %v", i, bars[i].Open, result[i].Open)
		}
	}
}

// TestRepairGBpUnitSwitchStaysInPence: a unit-switch error (early bars quoted
// in pounds) is repaired, and the output is still in pence — not GBP.
func TestRepairGBpUnitSwitchStaysInPence(t *testing.T) {
	opts := DefaultOptions()
	opts.Ticker = "ASAI.L"
	opts.Currency = "GBp"
	repairer := New(opts)

	// True prices in pence; first 5 bars erroneously quoted in pounds (100x low).
	truePence := []float64{6000, 6010, 5990, 6020, 5985, 6015, 5995, 6030, 5980, 6005}
	bars := make([]models.Bar, len(truePence))
	for i, c := range truePence {
		scale := 1.0
		if i < 5 {
			scale = 0.01
		}
		bars[i] = penceBar(i+1, (c-5)*scale, (c+20)*scale, (c-25)*scale, c*scale, 0)
	}

	result, err := repairer.Repair(bars)
	if err != nil {
		t.Fatalf("Repair returned error: %v", err)
	}

	for i := range result {
		if !closeTo(result[i].Close, truePence[i], 1e-6) {
			t.Errorf("bar %d: expected Close %v pence, got %v", i, truePence[i], result[i].Close)
		}
	}
	for i := 0; i < 5; i++ {
		if !result[i].Repaired {
			t.Errorf("bar %d: expected Repaired=true", i)
		}
	}
}
