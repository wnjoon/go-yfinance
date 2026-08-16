package repair

import (
	"math"
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func volBar(day int, price float64, volume int64) models.Bar {
	return models.Bar{
		Date:     time.Date(2024, 1, day, 0, 0, 0, 0, time.UTC),
		Open:     price * 0.999,
		High:     price * 1.004,
		Low:      price * 0.995,
		Close:    price,
		AdjClose: price,
		Volume:   volume,
	}
}

func TestDenoiseVolume(t *testing.T) {
	// Zeros are filled from neighbours, spikes are flattened by the median.
	vol := []float64{1000, 0, 1000, 1000, 100000, 1000, 1000, 0, 1000}
	out := denoiseVolume(vol)

	for i, v := range out {
		if v != 1000 {
			t.Errorf("index %d: expected denoised 1000, got %v", i, v)
		}
	}
}

func TestDenoiseVolumeEmpty(t *testing.T) {
	if out := denoiseVolume(nil); out != nil {
		t.Errorf("expected nil for empty input, got %v", out)
	}
}

func TestUnitSwitchRepairedWithFlatVolume(t *testing.T) {
	repairer := New(DefaultOptions())

	// Prices switch unit (x100) at index 5; volume stays flat -> unit switch.
	prices := []float64{1.00, 1.01, 0.99, 1.02, 0.995, 100.5, 99.0, 102.0, 99.5, 100.0}
	bars := make([]models.Bar, len(prices))
	for i, p := range prices {
		bars[i] = volBar(i+1, p, 100000)
	}

	result := repairer.fixPricesSuddenChange(bars, 100.0)

	for i := 0; i < 5; i++ {
		if !result[i].Repaired {
			t.Errorf("bar %d: expected Repaired=true (unit switch with flat volume)", i)
		}
		if result[i].Close < 90 {
			t.Errorf("bar %d: expected Close scaled x100, got %v", i, result[i].Close)
		}
	}
}

func TestUnitSwitchSkippedWhenVolumeMirrorsPrice(t *testing.T) {
	repairer := New(DefaultOptions())

	// Same price pattern, but volume collapses x100 at the boundary — the
	// signature of a real reverse split, not a unit switch (upstream #2943).
	prices := []float64{1.00, 1.01, 0.99, 1.02, 0.995, 100.5, 99.0, 102.0, 99.5, 100.0}
	bars := make([]models.Bar, len(prices))
	for i, p := range prices {
		volume := int64(100000)
		if i >= 5 {
			volume = 1000
		}
		bars[i] = volBar(i+1, p, volume)
	}

	result := repairer.fixPricesSuddenChange(bars, 100.0)

	for i := range result {
		if result[i].Repaired {
			t.Errorf("bar %d: expected no repair (volume mirrors price)", i)
		}
		if !closeTo(result[i].Close, bars[i].Close, 1e-12) {
			t.Errorf("bar %d: Close changed %v -> %v", i, bars[i].Close, result[i].Close)
		}
	}
}

func TestFixPricesSuddenChangeAllZeroVolume(t *testing.T) {
	repairer := New(DefaultOptions())

	prices := []float64{1.00, 1.01, 0.99, 100.5, 99.0, 102.0}
	bars := make([]models.Bar, len(prices))
	for i, p := range prices {
		bars[i] = volBar(i+1, p, 0)
	}

	result := repairer.fixPricesSuddenChange(bars, 100.0)

	for i := range result {
		if result[i].Repaired {
			t.Errorf("bar %d: expected no repair without volume data", i)
		}
	}
}

func TestRepairSkipsFXTicker(t *testing.T) {
	opts := DefaultOptions()
	opts.Ticker = "EURUSD=X"
	opts.QuoteType = QuoteTypeCurrency
	repairer := New(opts)

	// Unit-switch-looking pattern that would be "repaired" for an equity.
	prices := []float64{1.00, 1.01, 0.99, 1.02, 0.995, 100.5, 99.0, 102.0, 99.5, 100.0}
	bars := make([]models.Bar, len(prices))
	for i, p := range prices {
		bars[i] = volBar(i+1, p, 100000)
	}

	result, err := repairer.Repair(bars)
	if err != nil {
		t.Fatalf("Repair returned error: %v", err)
	}

	for i := range result {
		if !closeTo(result[i].Close, bars[i].Close, 1e-12) {
			t.Errorf("bar %d: FX bars must not be unit/split repaired: %v -> %v",
				i, bars[i].Close, result[i].Close)
		}
	}
}

func unadjustedSplitBars(volumes []int64) []models.Bar {
	// 2:1 split at index 5; Yahoo failed to adjust the earlier prices.
	prices := []float64{100, 101, 99, 100.5, 99.5, 50, 50.5, 49.8}
	bars := make([]models.Bar, len(prices))
	for i, p := range prices {
		bars[i] = volBar(i+1, p, volumes[i])
	}
	bars[5].Splits = 2.0
	return bars
}

func TestRepairStockSplitsUnadjustedWithVolume(t *testing.T) {
	repairer := New(DefaultOptions())

	// Volume doubles at the split date — the mirror image of the price drop.
	bars := unadjustedSplitBars([]int64{1200, 1150, 1250, 1180, 1220, 2400, 2500, 2450})

	result := repairer.repairStockSplits(bars)

	for i := 0; i < 5; i++ {
		expected := bars[i].Close / 2
		if !closeTo(result[i].Close, expected, 1e-9) {
			t.Errorf("bar %d: expected Close %v after split repair, got %v",
				i, expected, result[i].Close)
		}
		if !result[i].Repaired {
			t.Errorf("bar %d: expected Repaired=true", i)
		}
	}
}

func TestRepairStockSplitsSkippedWithFlatVolume(t *testing.T) {
	repairer := New(DefaultOptions())

	// Same price pattern, but volume is flat across the split date: without
	// the volume signature the repair must not fire (upstream #2943).
	bars := unadjustedSplitBars([]int64{1200, 1150, 1250, 1180, 1220, 1210, 1190, 1230})

	result := repairer.repairStockSplits(bars)

	for i := range result {
		if !closeTo(result[i].Close, bars[i].Close, 1e-12) {
			t.Errorf("bar %d: Close changed %v -> %v", i, bars[i].Close, result[i].Close)
		}
		if result[i].Repaired {
			t.Errorf("bar %d: expected no repair", i)
		}
	}
}

func TestExpectedSplitChangeSigns(t *testing.T) {
	if got := expectedSplitChange(2.0); !closeTo(got, -0.5, 1e-12) {
		t.Errorf("2:1 split: expected -0.5, got %v", got)
	}
	if got := expectedSplitChange(0.25); !closeTo(got, 3.0, 1e-12) {
		t.Errorf("1:4 reverse split: expected 3.0, got %v", got)
	}
}

func TestVolumeChangeThresholdScalesWithChange(t *testing.T) {
	vol := []float64{1000, 1010, 990, 1020, 995, 1005}
	t100 := volumeChangeThreshold(vol, 100, "1d")
	t2 := volumeChangeThreshold(vol, 2, "1d")
	if t100 <= t2 {
		t.Errorf("threshold must grow with change size: t100=%v t2=%v", t100, t2)
	}
	if t2 <= 1 {
		t.Errorf("threshold must exceed 1, got %v", t2)
	}
	if math.IsNaN(t100) {
		t.Error("threshold must not be NaN")
	}
}
