package repair

import (
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func TestRepairInconsistentOHLCCloseAboveHigh(t *testing.T) {
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 98, Close: 110, AdjClose: 110},
	}

	result := repairInconsistentOHLC(bars)

	bar := result[0]
	if !bar.Repaired {
		t.Fatal("expected Repaired=true for Close > High")
	}
	if bar.Close > bar.High || bar.Close < bar.Low {
		t.Errorf("Close %v still outside [Low %v, High %v]", bar.Close, bar.Low, bar.High)
	}
	if bar.Open != 100 || bar.Low != 98 {
		t.Errorf("good values must stay untouched: Open=%v Low=%v", bar.Open, bar.Low)
	}
}

func TestRepairInconsistentOHLCOpenBelowLow(t *testing.T) {
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 50, High: 105, Low: 98, Close: 102, AdjClose: 102},
	}

	result := repairInconsistentOHLC(bars)

	bar := result[0]
	if !bar.Repaired {
		t.Fatal("expected Repaired=true for Open < Low")
	}
	if bar.Open > bar.High || bar.Open < bar.Low {
		t.Errorf("Open %v still outside [Low %v, High %v]", bar.Open, bar.Low, bar.High)
	}
	if bar.Close != 102 || bar.High != 105 {
		t.Errorf("good values must stay untouched: Close=%v High=%v", bar.Close, bar.High)
	}
}

func TestRepairInconsistentOHLCConsistentUntouched(t *testing.T) {
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 98, Close: 102, AdjClose: 102},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Open: 0, High: 0, Low: 0, Close: 0, AdjClose: 0},
	}

	result := repairInconsistentOHLC(bars)

	for i, bar := range result {
		if bar.Repaired {
			t.Errorf("bar %d: expected no repair", i)
		}
	}
	if result[0].Close != 102 {
		t.Errorf("consistent bar changed: Close=%v", result[0].Close)
	}
}
