package models

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestParseEPSMethodology(t *testing.T) {
	valid := map[string]EPSMethodology{
		"":           EPSMethodologyDefault,
		"gaap":       EPSMethodologyGAAP,
		"GAAP":       EPSMethodologyGAAP,
		"Gaap":       EPSMethodologyGAAP,
		"normalized": EPSMethodologyNormalized,
		"Normalized": EPSMethodologyNormalized,
		"NORMALIZED": EPSMethodologyNormalized,
	}
	for in, want := range valid {
		got, err := ParseEPSMethodology(in)
		if err != nil {
			t.Fatalf("ParseEPSMethodology(%q) error: %v", in, err)
		}
		if got != want {
			t.Fatalf("ParseEPSMethodology(%q) = %q, want %q", in, got, want)
		}
	}

	// "nongaap" is Yahoo's label, not an accepted request value (as in Python).
	for _, in := range []string{"ifrs", "nongaap", "non-gaap", " gaap", "gaap ", "default", "none"} {
		got, err := ParseEPSMethodology(in)
		if err == nil {
			t.Fatalf("ParseEPSMethodology(%q) = %q, want error", in, got)
		}
		if !errors.Is(err, ErrInvalidEPSMethodology) {
			t.Fatalf("ParseEPSMethodology(%q) error %v does not wrap ErrInvalidEPSMethodology", in, err)
		}
		if !strings.Contains(err.Error(), in) {
			t.Fatalf("error %q should quote the rejected value %q", err, in)
		}
	}
}

func TestEPSMethodologyJSONOmitEmpty(t *testing.T) {
	cases := []struct {
		name string
		v    interface{}
	}{
		{"EarningsEstimate", EarningsEstimate{Period: "0q"}},
		{"EPSTrend", EPSTrend{Period: "0q"}},
		{"EPSRevision", EPSRevision{Period: "0q"}},
		{"EarningsHistoryItem", EarningsHistoryItem{Period: "-1q"}},
	}
	for _, tc := range cases {
		b, err := json.Marshal(tc.v)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "methodology") {
			t.Fatalf("%s: empty Methodology must be omitted, got %s", tc.name, b)
		}
	}

	b, err := json.Marshal(EarningsEstimate{Period: "0q", Methodology: "gaap"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"methodology":"gaap"`) {
		t.Fatalf("Methodology not serialized: %s", b)
	}
}
