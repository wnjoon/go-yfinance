package ticker

import (
	"encoding/json"
	"testing"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

// msftEPSRevisionsPayload is an earningsTrend quoteSummary module captured from
// Yahoo for MSFT (2026-11). Note "downLast7Days" with a capital D, next to
// "upLast7days" / "upLast30days" / "downLast30days", and the empty
// "downLast90days" object Yahoo also sends.
const msftEPSRevisionsPayload = `{
  "earningsTrend": {
    "trend": [
      {
        "period": "0q",
        "epsRevisions": {
          "upLast7days": {"raw": 5, "fmt": "5", "longFmt": "5"},
          "upLast30days": {"raw": 8, "fmt": "8", "longFmt": "8"},
          "downLast30days": {"raw": 7, "fmt": "7", "longFmt": "7"},
          "downLast7Days": {"raw": 3, "fmt": "3", "longFmt": "3"},
          "downLast90days": {},
          "epsRevisionsCurrency": "USD"
        }
      },
      {
        "period": "+1q",
        "epsRevisions": {
          "upLast7days": {"raw": 2, "fmt": "2", "longFmt": "2"},
          "upLast30days": {"raw": 4, "fmt": "4", "longFmt": "4"},
          "downLast30days": {"raw": 1, "fmt": "1", "longFmt": "1"},
          "downLast7Days": {"raw": 0, "fmt": "0", "longFmt": "0"},
          "downLast90days": {},
          "epsRevisionsCurrency": "USD"
        }
      }
    ]
  }
}`

func decodeTestJSON(t *testing.T, raw string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("decode test payload: %v", err)
	}
	return m
}

func TestParseEPSRevisionsYahooPayload(t *testing.T) {
	module, ok := decodeTestJSON(t, msftEPSRevisionsPayload)["earningsTrend"].(map[string]interface{})
	if !ok {
		t.Fatal("test payload has no earningsTrend module")
	}

	got := parseEPSRevisions(module, models.EPSMethodologyGAAP)
	want := []models.EPSRevision{
		{Period: "0q", Currency: "USD", UpLast7Days: 5, UpLast30Days: 8, DownLast7Days: 3, DownLast30Days: 7, Methodology: "gaap"},
		{Period: "+1q", Currency: "USD", UpLast7Days: 2, UpLast30Days: 4, DownLast7Days: 0, DownLast30Days: 1, Methodology: "gaap"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d revisions, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("revision %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestParseEPSRevisionKeyCasing(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    models.EPSRevision
	}{
		{
			name:    "documented lowercase keys",
			payload: `{"upLast7days":{"raw":1},"upLast30days":{"raw":2},"downLast7days":{"raw":3},"downLast30days":{"raw":4}}`,
			want:    models.EPSRevision{Period: "0y", UpLast7Days: 1, UpLast30Days: 2, DownLast7Days: 3, DownLast30Days: 4},
		},
		{
			name:    "capital D on downLast7Days (current Yahoo payload)",
			payload: `{"upLast7days":{"raw":1},"upLast30days":{"raw":2},"downLast7Days":{"raw":3},"downLast30days":{"raw":4}}`,
			want:    models.EPSRevision{Period: "0y", UpLast7Days: 1, UpLast30Days: 2, DownLast7Days: 3, DownLast30Days: 4},
		},
		{
			name:    "all keys capitalised",
			payload: `{"upLast7Days":{"raw":1},"upLast30Days":{"raw":2},"downLast7Days":{"raw":3},"downLast30Days":{"raw":4}}`,
			want:    models.EPSRevision{Period: "0y", UpLast7Days: 1, UpLast30Days: 2, DownLast7Days: 3, DownLast30Days: 4},
		},
		{
			name:    "exact spelling wins over a case variant",
			payload: `{"downLast7days":{"raw":9},"downLast7Days":{"raw":3}}`,
			want:    models.EPSRevision{Period: "0y", DownLast7Days: 9},
		},
		{
			name:    "plain numbers instead of raw objects",
			payload: `{"upLast7days":6,"downLast7Days":2}`,
			want:    models.EPSRevision{Period: "0y", UpLast7Days: 6, DownLast7Days: 2},
		},
		{
			name:    "empty objects and missing keys give zero",
			payload: `{"upLast7days":{},"downLast7Days":{},"downLast90days":{}}`,
			want:    models.EPSRevision{Period: "0y"},
		},
		{
			name:    "currency is kept",
			payload: `{"downLast7Days":{"raw":1},"epsRevisionsCurrency":"EUR"}`,
			want:    models.EPSRevision{Period: "0y", Currency: "EUR", DownLast7Days: 1},
		},
		{
			name:    "unrelated key with similar prefix is ignored",
			payload: `{"downLast7DaysX":{"raw":5},"downLast90days":{"raw":8}}`,
			want:    models.EPSRevision{Period: "0y"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseEPSRevision("0y", decodeTestJSON(t, tt.payload))
			if got != tt.want {
				t.Errorf("parseEPSRevision() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestGetNestedIntFold(t *testing.T) {
	m := decodeTestJSON(t, `{"Alpha":{"raw":1},"ALPHA":{"raw":2},"beta":{"raw":3},"gamma":4,"delta":"x"}`)

	tests := []struct {
		key  string
		want int
	}{
		{"beta", 3},  // exact
		{"BETA", 3},  // folded
		{"gamma", 4}, // plain number
		{"alpha", 2}, // no exact match: sorted first of {"ALPHA","Alpha"} is "ALPHA"
		{"Alpha", 1}, // exact match wins
		{"delta", 0}, // non-numeric
		{"missing", 0},
	}
	for _, tt := range tests {
		if got := getNestedIntFold(m, tt.key); got != tt.want {
			t.Errorf("getNestedIntFold(%q) = %d, want %d", tt.key, got, tt.want)
		}
	}

	if got := getNestedIntFold(nil, "beta"); got != 0 {
		t.Errorf("getNestedIntFold(nil) = %d, want 0", got)
	}
}

func TestEPSRevisionJSONKeysUnchanged(t *testing.T) {
	// The exported JSON field names must not change with the parsing fix.
	b, err := json.Marshal(models.EPSRevision{Period: "0q", DownLast7Days: 3})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"upLast7days", "upLast30days", "downLast7days", "downLast30days"} {
		if _, ok := m[key]; !ok {
			t.Errorf("JSON key %q missing from %s", key, b)
		}
	}
	if m["downLast7days"] != float64(3) {
		t.Errorf("downLast7days = %v, want 3", m["downLast7days"])
	}
}
