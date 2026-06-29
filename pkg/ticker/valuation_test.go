package ticker

import (
	"reflect"
	"testing"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

const valuationMeasuresTimeseries = `{
	"timeseries": {
		"result": [
			{
				"meta": {"type": ["quarterlyMarketCap"]},
				"timestamp": [1767139200, 1759190400],
				"quarterlyMarketCap": [
					{"asOfDate": "2025-12-31", "reportedValue": {"raw": 4000000000000}},
					{"asOfDate": "2025-09-30", "reportedValue": {"raw": 3760000000000}}
				]
			},
			{
				"meta": {"type": ["trailingMarketCap"]},
				"timestamp": [1780617600],
				"trailingMarketCap": [
					{"asOfDate": "2026-06-05", "reportedValue": {"raw": 4514000000000}}
				]
			},
			{
				"meta": {"type": ["quarterlyPeRatio"]},
				"timestamp": [1759190400],
				"quarterlyPeRatio": [
					{"asOfDate": "2025-09-30", "reportedValue": {"raw": 38.64}}
				]
			},
			{
				"meta": {"type": ["trailingPeRatio"]},
				"timestamp": [1780617600],
				"trailingPeRatio": [
					{"asOfDate": "2026-06-05", "reportedValue": {"raw": 37.21}}
				]
			}
		]
	}
}`

func TestParseValuationMeasuresTimeseries(t *testing.T) {
	measures, err := parseValuationMeasuresTimeseries(valuationMeasuresTimeseries, "quarterly")
	if err != nil {
		t.Fatalf("parseValuationMeasuresTimeseries returned error: %v", err)
	}

	if measures.Empty() {
		t.Fatal("Expected valuation measures to be non-empty")
	}
	if len(measures.Rows) != 9 {
		t.Fatalf("Expected 9 rows, got %d", len(measures.Rows))
	}
	expectedColumns := []string{"Current", "12/31/2025", "9/30/2025"}
	if !reflect.DeepEqual(measures.Columns, expectedColumns) {
		t.Fatalf("Expected columns %v, got %v", expectedColumns, measures.Columns)
	}
	if got, ok := measures.FloatValue("Market Cap", "Current"); !ok || got != 4514000000000 {
		t.Errorf("Expected Market Cap Current 4514000000000, got %v (ok=%v)", got, ok)
	}
	if got, ok := measures.FloatValue("Market Cap", "12/31/2025"); !ok || got != 4000000000000 {
		t.Errorf("Expected Market Cap 12/31/2025 4000000000000, got %v (ok=%v)", got, ok)
	}
	if got, ok := measures.FloatValue("Trailing P/E", "9/30/2025"); !ok || got != 38.64 {
		t.Errorf("Expected Trailing P/E 9/30/2025 38.64, got %v (ok=%v)", got, ok)
	}
}

func TestParseValuationMeasuresListsAllMeasures(t *testing.T) {
	measures, err := parseValuationMeasuresTimeseries(valuationMeasuresTimeseries, "quarterly")
	if err != nil {
		t.Fatalf("parseValuationMeasuresTimeseries returned error: %v", err)
	}

	if len(measures.Rows) != 9 {
		t.Fatalf("Expected all 9 valuation rows, got %d", len(measures.Rows))
	}
	if _, ok := measures.FloatValue("PEG Ratio (5yr expected)", "Current"); ok {
		t.Fatal("Expected missing PEG Ratio Current cell")
	}
	if got, ok := measures.Value("PEG Ratio (5yr expected)", "Current"); !ok || got != "" {
		t.Fatalf("Expected empty compatibility string for missing PEG Ratio Current, got %q (ok=%v)", got, ok)
	}
}

func TestParseValuationMeasuresEmpty(t *testing.T) {
	measures, err := parseValuationMeasuresTimeseries(`{"timeseries":{"result":[]}}`, "quarterly")
	if err != nil {
		t.Fatalf("parseValuationMeasuresTimeseries returned error: %v", err)
	}
	if !measures.Empty() {
		t.Fatal("Expected empty valuation measures")
	}
}

func TestParseValuationMeasuresFreqYearly(t *testing.T) {
	payload := `{"timeseries":{"result":[
		{"meta":{"type":["annualMarketCap"]},"timestamp":[1],
		 "annualMarketCap":[{"asOfDate":"2025-09-30","reportedValue":{"raw":3760000000000}}]},
		{"meta":{"type":["trailingMarketCap"]},"timestamp":[2],
		 "trailingMarketCap":[{"asOfDate":"2026-06-05","reportedValue":{"raw":4510000000000}}]}
	]}}`

	measures, err := parseValuationMeasuresTimeseries(payload, "annual")
	if err != nil {
		t.Fatalf("parseValuationMeasuresTimeseries returned error: %v", err)
	}

	expectedColumns := []string{"Current", "9/30/2025"}
	if !reflect.DeepEqual(measures.Columns, expectedColumns) {
		t.Fatalf("Expected columns %v, got %v", expectedColumns, measures.Columns)
	}
	if got, ok := measures.FloatValue("Market Cap", "Current"); !ok || got != 4510000000000 {
		t.Errorf("Expected current Market Cap, got %v (ok=%v)", got, ok)
	}
}

func TestValuationPeriodsSlicing(t *testing.T) {
	measures, err := parseValuationMeasuresTimeseries(valuationPeriodsPayload(), "quarterly")
	if err != nil {
		t.Fatalf("parseValuationMeasuresTimeseries returned error: %v", err)
	}

	two := 2
	sliced := sliceValuationPeriods(measures, &two)
	expectedColumns := []string{"Current", "12/31/2026", "9/30/2026"}
	if !reflect.DeepEqual(sliced.Columns, expectedColumns) {
		t.Fatalf("Expected columns %v, got %v", expectedColumns, sliced.Columns)
	}

	zero := 0
	currentOnly := sliceValuationPeriods(measures, &zero)
	if !reflect.DeepEqual(currentOnly.Columns, []string{"Current"}) {
		t.Fatalf("Expected Current only, got %v", currentOnly.Columns)
	}

	all := sliceValuationPeriods(measures, nil)
	if len(all.Columns) != 7 {
		t.Fatalf("Expected all 7 columns, got %d", len(all.Columns))
	}
}

func TestGetValuationMeasuresCacheAndPeriods(t *testing.T) {
	tkr, err := New("AAPL")
	if err != nil {
		t.Fatalf("Failed to create ticker: %v", err)
	}

	measures, err := parseValuationMeasuresTimeseries(valuationPeriodsPayload(), "quarterly")
	if err != nil {
		t.Fatalf("parseValuationMeasuresTimeseries returned error: %v", err)
	}
	tkr.valuationCache = map[string]*models.ValuationMeasures{"quarterly": measures}

	two := 2
	first, err := tkr.GetValuationMeasures("quarterly", &two)
	if err != nil {
		t.Fatalf("GetValuationMeasures returned error: %v", err)
	}
	five := 5
	second, err := tkr.GetValuationMeasures("quarterly", &five)
	if err != nil {
		t.Fatalf("GetValuationMeasures returned error: %v", err)
	}

	if len(first.Columns) != 3 {
		t.Fatalf("Expected Current + 2 period columns, got %v", first.Columns)
	}
	if len(second.Columns) != 6 {
		t.Fatalf("Expected Current + 5 period columns, got %v", second.Columns)
	}
}

func TestGetValuationMeasuresInvalidInputs(t *testing.T) {
	tkr, err := New("AAPL")
	if err != nil {
		t.Fatalf("Failed to create ticker: %v", err)
	}

	if _, err := tkr.GetValuationMeasures("bogus", nil); err == nil {
		t.Fatal("Expected invalid freq error")
	}

	negative := -1
	if _, err := tkr.GetValuationMeasures("quarterly", &negative); err == nil {
		t.Fatal("Expected invalid periods error")
	}
}

func valuationPeriodsPayload() string {
	return `{"timeseries":{"result":[
		{"meta":{"type":["quarterlyMarketCap"]},"timestamp":[1,2,3,4,5,6],
		 "quarterlyMarketCap":[
			{"asOfDate":"2026-12-31","reportedValue":{"raw":5000000000000}},
			{"asOfDate":"2026-09-30","reportedValue":{"raw":4900000000000}},
			{"asOfDate":"2026-06-30","reportedValue":{"raw":4800000000000}},
			{"asOfDate":"2026-03-31","reportedValue":{"raw":4700000000000}},
			{"asOfDate":"2025-12-31","reportedValue":{"raw":4600000000000}},
			{"asOfDate":"2025-09-30","reportedValue":{"raw":4500000000000}}
		 ]},
		{"meta":{"type":["trailingMarketCap"]},"timestamp":[99],
		 "trailingMarketCap":[{"asOfDate":"2026-06-05","reportedValue":{"raw":5100000000000}}]}
	]}}`
}
