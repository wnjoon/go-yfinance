package models

import (
	"errors"
	"testing"
)

func TestTickersInfoResultHelpers(t *testing.T) {
	var nilResult *TickersInfoResult
	if nilResult.Get("AAPL") != nil || nilResult.HasErrors() || nilResult.SuccessCount() != 0 || nilResult.ErrorCount() != 0 {
		t.Error("nil TickersInfoResult helpers should return zero values")
	}

	r := &TickersInfoResult{
		Data:    map[string]*Info{"AAPL": {Symbol: "AAPL"}},
		Errors:  map[string]error{"BAD": errors.New("boom")},
		Symbols: []string{"AAPL"},
	}
	if r.Get("AAPL") == nil || r.Get("BAD") != nil {
		t.Error("Get mismatch")
	}
	if !r.HasErrors() || r.SuccessCount() != 1 || r.ErrorCount() != 1 {
		t.Errorf("counts = %v/%d/%d", r.HasErrors(), r.SuccessCount(), r.ErrorCount())
	}
	if (&TickersInfoResult{}).Get("AAPL") != nil {
		t.Error("Get on empty result should be nil")
	}
}

func TestTickersCalendarResultHelpers(t *testing.T) {
	var nilResult *TickersCalendarResult
	if nilResult.Get("AAPL") != nil || nilResult.HasErrors() || nilResult.SuccessCount() != 0 || nilResult.ErrorCount() != 0 {
		t.Error("nil TickersCalendarResult helpers should return zero values")
	}

	r := &TickersCalendarResult{
		Data:    map[string]*Calendar{"AAPL": {}},
		Errors:  map[string]error{},
		Symbols: []string{"AAPL"},
	}
	if r.Get("AAPL") == nil || r.Get("MSFT") != nil {
		t.Error("Get mismatch")
	}
	if r.HasErrors() || r.SuccessCount() != 1 || r.ErrorCount() != 0 {
		t.Errorf("counts = %v/%d/%d", r.HasErrors(), r.SuccessCount(), r.ErrorCount())
	}
}
