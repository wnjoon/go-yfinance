package ticker

import (
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func TestParseDividendEventsPreservesEventRows(t *testing.T) {
	result := &models.ChartResult{
		Timestamp: []int64{1704067200},
		Events: &models.ChartEvents{
			Dividends: map[string]models.DividendEvent{
				"1704153600": {Date: 1704153600, Amount: 0.24, Currency: "USD"},
				"1704067200": {Date: 1704067200, Amount: 0},
			},
		},
	}

	dividends := parseDividendEvents(result)
	if len(dividends) != 1 {
		t.Fatalf("Expected one positive dividend event, got %d", len(dividends))
	}
	if dividends[0].Date != time.Unix(1704153600, 0).UTC() {
		t.Fatalf("Expected dividend event date from events map, got %s", dividends[0].Date)
	}
	if dividends[0].Amount != 0.24 {
		t.Errorf("Expected dividend amount 0.24, got %v", dividends[0].Amount)
	}
	if dividends[0].Currency != "USD" {
		t.Errorf("Expected dividend currency USD, got %q", dividends[0].Currency)
	}
}

func TestParseSplitEvents(t *testing.T) {
	result := &models.ChartResult{
		Events: &models.ChartEvents{
			Splits: map[string]models.SplitEvent{
				"old": {Date: 946684800, Numerator: 2, Denominator: 1, SplitRatio: "2:1"},
				"new": {Date: 1704067200, Numerator: 3, Denominator: 2},
				"bad": {Date: 1704153600, Numerator: 1, Denominator: 0},
			},
		},
	}

	splits := parseSplitEvents(result)
	if len(splits) != 2 {
		t.Fatalf("Expected two valid split events, got %d", len(splits))
	}
	if splits[0].Ratio != "2:1" {
		t.Errorf("Expected provided split ratio 2:1, got %q", splits[0].Ratio)
	}
	if splits[1].Ratio != "3:2" {
		t.Errorf("Expected generated split ratio 3:2, got %q", splits[1].Ratio)
	}
}

func TestParseCapitalGainEvents(t *testing.T) {
	result := &models.ChartResult{
		Events: &models.ChartEvents{
			CapitalGains: map[string]models.CapitalGainEvent{
				"1704067200": {Date: 1704067200, Amount: 0.12},
				"1704153600": {Date: 1704153600, Amount: 0},
			},
		},
	}

	capitalGains := parseCapitalGainEvents(result)
	if len(capitalGains) != 1 {
		t.Fatalf("Expected one positive capital gain event, got %d", len(capitalGains))
	}
	if capitalGains[0].Date != time.Unix(1704067200, 0).UTC() {
		t.Fatalf("Expected capital gain event date from events map, got %s", capitalGains[0].Date)
	}
	if capitalGains[0].Amount != 0.12 {
		t.Errorf("Expected capital gain amount 0.12, got %v", capitalGains[0].Amount)
	}
}
