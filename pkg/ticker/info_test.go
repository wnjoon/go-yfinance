package ticker

import (
	"errors"
	"testing"

	"github.com/wnjoon/go-yfinance/pkg/client"
)

func TestParseInfoResponseSparsePayloads(t *testing.T) {
	tkr, err := New("MSFT")
	if err != nil {
		t.Fatalf("Failed to create ticker: %v", err)
	}

	tests := []struct {
		name        string
		body        string
		expectError bool
	}{
		{
			name: "empty result",
			body: `{"quoteSummary":{"result":[],"error":null}}`,
		},
		{
			name: "null result",
			body: `{"quoteSummary":{"result":null,"error":null}}`,
		},
		{
			name: "missing result",
			body: `{"quoteSummary":{"error":null}}`,
		},
		{
			name: "missing quoteSummary envelope",
			body: `{}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			info, err := tkr.parseInfoResponse(tc.body)
			if err == nil {
				t.Fatalf("Expected not-found error, got info=%+v", info)
			}
			if !errors.Is(err, client.ErrNotFound) {
				t.Fatalf("Expected not-found error, got %v", err)
			}
		})
	}
}

func TestParseInfoResponseYahooError(t *testing.T) {
	tkr, err := New("MSFT")
	if err != nil {
		t.Fatalf("Failed to create ticker: %v", err)
	}

	_, err = tkr.parseInfoResponse(`{"quoteSummary":{"result":null,"error":{"code":"Not Found","description":"No fundamentals data found"}}}`)
	if err == nil {
		t.Fatal("Expected Yahoo API error")
	}
}

func TestParseInfoResponsePartialResult(t *testing.T) {
	tkr, err := New("MSFT")
	if err != nil {
		t.Fatalf("Failed to create ticker: %v", err)
	}

	info, err := tkr.parseInfoResponse(`{
		"quoteSummary": {
			"result": [{
				"quoteType": {
					"symbol": "MSFT",
					"longName": "Microsoft Corporation",
					"quoteType": "EQUITY"
				}
			}],
			"error": null
		}
	}`)
	if err != nil {
		t.Fatalf("Expected partial result to parse, got %v", err)
	}
	if info.Symbol != "MSFT" {
		t.Errorf("Expected symbol MSFT, got %s", info.Symbol)
	}
	if info.LongName != "Microsoft Corporation" {
		t.Errorf("Expected long name, got %q", info.LongName)
	}
	if info.QuoteType != "EQUITY" {
		t.Errorf("Expected quote type EQUITY, got %q", info.QuoteType)
	}
}

func TestParseTrailingPegRatioTimeseriesSparsePayloads(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "empty result",
			body: `{"timeseries":{"result":[]}}`,
		},
		{
			name: "missing result",
			body: `{"timeseries":{}}`,
		},
		{
			name: "missing key",
			body: `{"timeseries":{"result":[{}]}}`,
		},
		{
			name: "empty key list",
			body: `{"timeseries":{"result":[{"trailingPegRatio":[]}]}}`,
		},
		{
			name: "missing reported value",
			body: `{"timeseries":{"result":[{"trailingPegRatio":[{}]}]}}`,
		},
		{
			name: "missing raw value",
			body: `{"timeseries":{"result":[{"trailingPegRatio":[{"reportedValue":{}}]}]}}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			value, err := parseTrailingPegRatioTimeseries(tc.body)
			if err != nil {
				t.Fatalf("parseTrailingPegRatioTimeseries returned error: %v", err)
			}
			if value != nil {
				t.Fatalf("Expected nil value, got %v", *value)
			}
		})
	}
}

func TestParseTrailingPegRatioTimeseriesLatestValue(t *testing.T) {
	value, err := parseTrailingPegRatioTimeseries(`{
		"timeseries": {
			"result": [{
				"trailingPegRatio": [
					{"asOfDate": "2026-01-01", "reportedValue": {"raw": 1.25}},
					{"asOfDate": "2026-02-01", "reportedValue": {"raw": 1.5}}
				]
			}]
		}
	}`)
	if err != nil {
		t.Fatalf("parseTrailingPegRatioTimeseries returned error: %v", err)
	}
	if value == nil || *value != 1.5 {
		t.Fatalf("Expected latest trailing PEG 1.5, got %v", value)
	}
}
