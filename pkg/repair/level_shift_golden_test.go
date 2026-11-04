package repair

import (
	"math"
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

// fakeHourly is a fake HourlyFetcher that serves fixture bars within the
// requested window and records every call.
type fakeHourly struct {
	bars  []models.Bar
	err   error
	calls []fakeHourlyCall
}

type fakeHourlyCall struct {
	start, end time.Time
}

func (f *fakeHourly) fetch(start, end time.Time) ([]models.Bar, error) {
	f.calls = append(f.calls, fakeHourlyCall{start, end})
	if f.err != nil {
		return nil, f.err
	}
	var out []models.Bar
	for _, b := range f.bars {
		if !b.Date.Before(start) && b.Date.Before(end) {
			out = append(out, b)
		}
	}
	return out, nil
}

// levelShiftOptions returns daily repair options for a New York ticker with
// only the level-shift repair enabled, a fake hourly source and a fixed
// clock.
func levelShiftOptions(ticker string, hourly *fakeHourly, now time.Time) Options {
	return Options{
		Ticker:         ticker,
		Interval:       "1d",
		Timezone:       "America/New_York",
		Currency:       "USD",
		FixLevelShifts: true,
		FetchHourly:    hourly.fetch,
		Now:            func() time.Time { return now },
	}
}

// fixtureNow puts the clock a week after the fixture's last row, as upstream's
// test moves its fixtures to recent dates.
func fixtureNow(bars []models.Bar) time.Time {
	return bars[len(bars)-1].Date.Add(7 * 24 * time.Hour)
}

// TestRepairLevelShiftSOXSGolden ports upstream
// test_repair_unexplained_level_shift: SOXS 1d prices before 2026-05-26 are
// 15x too high and volume 15x too low, with no split near that date. The 1h
// data measures the 15x and the split-path repair fixes it. Dividends are not
// touched.
func TestRepairLevelShiftSOXSGolden(t *testing.T) {
	bad := loadBarsCSV(t, "SOXS-1d-bad-level-shift.csv")
	fixed := loadBarsCSV(t, "SOXS-1d-bad-level-shift-fixed.csv")
	hourly := &fakeHourly{bars: loadBarsCSV(t, "SOXS-1h-bad-level-shift.csv")}
	now := fixtureNow(bad)

	got, err := New(levelShiftOptions("SOXS", hourly, now)).Repair(bad)
	if err != nil {
		t.Fatalf("Repair: %v", err)
	}
	if len(hourly.calls) != 1 {
		t.Fatalf("hourly fetches = %d, want 1", len(hourly.calls))
	}
	assertLevelShiftGolden(t, "SOXS", got, fixed, 1e-6)

	ny, _ := time.LoadLocation("America/New_York")
	shift := time.Date(2026, 5, 26, 0, 0, 0, 0, ny)
	for i := range got {
		want := got[i].Date.Before(shift)
		if got[i].Repaired != want {
			t.Errorf("row=%d (%s) Repaired=%v, want %v", i, got[i].Date.Format(time.DateOnly), got[i].Repaired, want)
		}
		if got[i].Dividends != bad[i].Dividends {
			t.Errorf("row=%d Dividends changed: %v -> %v", i, bad[i].Dividends, got[i].Dividends)
		}
	}
	if CountRepaired(bad) != 0 {
		t.Fatal("Repair mutated its input")
	}
}

// TestRepairLevelShiftKALAGenuineCrash ports upstream
// test_repair_unexplained_level_shift_false_positive: KALA fell 92% on
// 2025-09-29 after a failed trial and volume surged. It looks like a level
// shift, but the 1h data shows the same drop, so nothing changes.
func TestRepairLevelShiftKALAGenuineCrash(t *testing.T) {
	good := loadBarsCSV(t, "KALA-1d-no-bad-level-shift.csv")
	hourly := &fakeHourly{bars: loadBarsCSV(t, "KALA-1h-no-bad-level-shift.csv")}

	got, err := New(levelShiftOptions("KALA", hourly, fixtureNow(good))).Repair(good)
	if err != nil {
		t.Fatalf("Repair: %v", err)
	}
	if len(hourly.calls) != 1 {
		t.Fatalf("hourly fetches = %d, want 1", len(hourly.calls))
	}
	assertLevelShiftGolden(t, "KALA", got, good, 0)
	if n := CountRepaired(got); n != 0 {
		t.Fatalf("repaired %d rows, want 0", n)
	}
}

// assertLevelShiftGolden compares prices with relative tolerance rtol and
// volume exactly.
func assertLevelShiftGolden(t *testing.T, label string, got, want []models.Bar, rtol float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: got %d rows, want %d", label, len(got), len(want))
	}
	for i := range got {
		if !got[i].Date.Equal(want[i].Date) {
			t.Fatalf("%s row=%d date %v, want %v", label, i, got[i].Date, want[i].Date)
		}
		for _, c := range []struct {
			name      string
			got, want float64
		}{
			{"Open", got[i].Open, want[i].Open}, {"High", got[i].High, want[i].High},
			{"Low", got[i].Low, want[i].Low}, {"Close", got[i].Close, want[i].Close},
			{"AdjClose", got[i].AdjClose, want[i].AdjClose},
		} {
			if math.IsNaN(c.got) && math.IsNaN(c.want) {
				continue
			}
			if !closeTo(c.got, c.want, rtol) {
				t.Errorf("%s row=%d %s: got %v want %v", label, i, c.name, c.got, c.want)
			}
		}
		if got[i].Volume != want[i].Volume {
			t.Errorf("%s row=%d Volume: got %d want %d", label, i, got[i].Volume, want[i].Volume)
		}
	}
}
