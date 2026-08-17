package repair

import (
	"math"
	"testing"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

// TestApplyUnitSwitchCorrectionPreservesVolumeScalesDividend locks the
// cross-verification finding (high): a currency-unit switch does not change
// the share count, so Volume must stay untouched, while Dividends scales by
// the same factor as prices (upstream passes correct_dividend=True but never
// correct_volume=True for a unit switch).
func TestApplyUnitSwitchCorrectionPreservesVolumeScalesDividend(t *testing.T) {
	bars := []models.Bar{
		{Open: 1.0, High: 1.05, Low: 0.98, Close: 1.0, AdjClose: 1.0, Dividends: 0.02, Volume: 12345},
	}

	applyUnitSwitchCorrection(bars, 100.0)

	if bars[0].Volume != 12345 {
		t.Errorf("expected Volume untouched by unit-switch correction, got %d", bars[0].Volume)
	}
	if !closeTo(bars[0].Dividends, 2.0, 1e-9) {
		t.Errorf("expected Dividends scaled to 2.0, got %v", bars[0].Dividends)
	}
	if !closeTo(bars[0].Close, 100.0, 1e-9) {
		t.Errorf("expected Close scaled to 100.0, got %v", bars[0].Close)
	}
}

// TestApplySplitCorrectionScalesDividend locks that split repair scales
// Dividends by the same factor as prices, matching upstream's
// correct_dividend=True on the shared split-repair call.
func TestApplySplitCorrectionScalesDividend(t *testing.T) {
	bars := []models.Bar{
		{Open: 100, High: 105, Low: 95, Close: 102, AdjClose: 102, Dividends: 1.0, Volume: 1000},
		{Open: 54, High: 58, Low: 50, Close: 55, AdjClose: 55, Volume: 2400, Splits: 2.0},
	}

	result := applySplitCorrection(bars, 1, 2.0)

	if !closeTo(result[0].Dividends, 0.5, 1e-9) {
		t.Errorf("expected pre-split Dividends halved to 0.5, got %v", result[0].Dividends)
	}
}

// TestSplitVolumeConfirmsNoVetoWhenBoundaryUnusable locks the fix: when the
// boundary volume change cannot be computed (no positive volume on one
// side), upstream does not veto the split repair, unlike the all-zero-volume
// case which disables repair entirely.
func TestSplitVolumeConfirmsNoVetoWhenBoundaryUnusable(t *testing.T) {
	bars := make([]models.Bar, 6)
	for i := range bars {
		bars[i] = volBar(i+1, 100, 0)
	}
	// Only the post-boundary side has any volume; pre-boundary side is all
	// zero, so boundaryVolumeChange returns NaN.
	for i := 3; i < 6; i++ {
		bars[i].Volume = 1000
	}

	if !splitVolumeConfirms(bars, 3, 2.0, "1d") {
		t.Error("expected splitVolumeConfirms=true (no veto) when boundary volume is unusable")
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

// TestFixPricesSuddenChangeWeeklyNoiseMultiplier locks that the price-side
// noise tolerance widens for 1wk (x3) but NOT for 5d, matching upstream's
// interday definition {1d, 1wk, 1mo, 3mo}.
func TestFixPricesSuddenChangeWeeklyNoiseMultiplier(t *testing.T) {
	weekly := intervalNoiseMultiplier("1wk")
	fiveDay := intervalNoiseMultiplier("5d")
	daily := intervalNoiseMultiplier("1d")
	monthly := intervalNoiseMultiplier("1mo")

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

	result := repairer.fixPricesSuddenChange(bars, 100.0)
	for i, bar := range result {
		if math.IsNaN(bar.Close) || math.IsInf(bar.Close, 0) {
			t.Fatalf("bar %d: Close is NaN/Inf after repair", i)
		}
	}
}
