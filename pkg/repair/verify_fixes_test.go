package repair

import (
	"math"
	"testing"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

// TestScaleBarUnitSwitchVsSplit locks the correction flags: a currency-unit
// switch does not change the share count, so Volume stays untouched while
// Dividends scale with prices (upstream correct_dividend=True only); split
// repair also rescales Volume, rounding half to even.
func TestScaleBarUnitSwitchVsSplit(t *testing.T) {
	unit := models.Bar{Open: 1.0, High: 1.05, Low: 0.98, Close: 1.0, AdjClose: 1.0, Dividends: 0.02, Volume: 12345}
	scaleBar(&unit, 100, 0.01, priceShiftFix{unitSwitch: true, correctDividend: true})
	if unit.Volume != 12345 || !closeTo(unit.Dividends, 2.0, 1e-9) || !closeTo(unit.Close, 100.0, 1e-9) {
		t.Errorf("unit switch: %+v", unit)
	}

	split := models.Bar{Open: 100, High: 105, Low: 95, Close: 102, AdjClose: 102, Dividends: 1.0, Volume: 1001}
	scaleBar(&split, 0.5, 2, priceShiftFix{correctVolume: true, correctDividend: true})
	if split.Volume != 2002 || !closeTo(split.Dividends, 0.5, 1e-9) {
		t.Errorf("split: %+v", split)
	}
	half := models.Bar{Volume: 5}
	scaleBar(&half, 2, 0.5, priceShiftFix{correctVolume: true})
	if half.Volume != 2 {
		t.Errorf("2.5 should round half to even, got %d", half.Volume)
	}
}

// TestSplitDetectionRejectsModeratePriceDrop locks the threshold-formula fix
// (finding #2): a ~30% drop with low background volatility must not be
// mistaken for an unadjusted 2:1 split (unadjusted implies ~50% drop, and the
// upstream formula 1+(splitMax-1+pct)*0.6 requires the ratio to fall below
// roughly 1/1.6 = 0.625, not merely below 0.75 as the old distance formula
// allowed).
func TestSplitDetectionRejectsModeratePriceDrop(t *testing.T) {
	repairer := New(DefaultOptions())

	prices := []float64{100, 100.5, 99.5, 100.2, 99.8, 70, 70.5, 69.8}
	bars := make([]models.Bar, len(prices))
	for i, p := range prices {
		bars[i] = volBar(i+1, p, 1200)
	}
	bars[5].Splits = 2.0
	// Volume doubles at the boundary too, so a weaker threshold would
	// otherwise treat this 30% drop as confirmed.
	bars[5].Volume = 2400
	bars[6].Volume = 2400
	bars[7].Volume = 2400

	result := repairer.repairStockSplits(bars)

	for i := range result {
		if result[i].Repaired {
			t.Errorf("bar %d: a 30%% drop must not be repaired as an unadjusted 2:1 split", i)
		}
		if !closeTo(result[i].Close, bars[i].Close, 1e-12) {
			t.Errorf("bar %d: Close changed %v -> %v", i, bars[i].Close, result[i].Close)
		}
	}
}

// TestScaleLargestChangeIntervals locks that the price-side
// noise tolerance widens for 1wk (x3) but NOT for 5d, matching upstream's
// interday definition {1d, 1wk, 1mo, 3mo}.
func TestScaleLargestChangeIntervals(t *testing.T) {
	weekly := scaleLargestChange(1, "1wk")
	fiveDay := scaleLargestChange(1, "5d")
	daily := scaleLargestChange(1, "1d")
	monthly := scaleLargestChange(1, "1mo")

	if weekly != 3 {
		t.Errorf("expected 1wk multiplier 3, got %v", weekly)
	}
	if fiveDay != 1 {
		t.Errorf("expected 5d multiplier 1 (not interday upstream), got %v", fiveDay)
	}
	if daily != 1 {
		t.Errorf("expected 1d multiplier 1, got %v", daily)
	}
	if monthly != 6 {
		t.Errorf("expected 1mo multiplier 6, got %v", monthly)
	}
}

// TestFixPricesSuddenChangeUsesPopulationStd is a smoke check that std
// computation does not produce NaN/Inf with a tiny sample, guarding the
// ddof=0 (population) switch made to match upstream's np.std default.
func TestFixPricesSuddenChangeUsesPopulationStd(t *testing.T) {
	repairer := New(DefaultOptions())
	prices := []float64{1.0, 1.01, 0.99, 100.5, 99.0, 102.0}
	bars := make([]models.Bar, len(prices))
	for i, p := range prices {
		bars[i] = volBar(i+1, p, 100000)
	}

	result := repairer.repairUnitSwitch(bars)
	for i, bar := range result {
		if math.IsNaN(bar.Close) || math.IsInf(bar.Close, 0) {
			t.Fatalf("bar %d: Close is NaN/Inf after repair", i)
		}
	}
}
