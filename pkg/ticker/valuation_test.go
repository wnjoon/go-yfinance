package ticker

import "testing"

const valuationMeasuresHTML = `<html><body>
<table>
	<tr><td></td><td>Current</td><td>12/31/2025</td><td>9/30/2025</td></tr>
	<tr><td>Market Cap</td><td>3.76T</td><td>4.00T</td><td>3.76T</td></tr>
	<tr><td>Enterprise Value</td><td>3.78T</td><td>4.04T</td><td>3.81T</td></tr>
	<tr><td>Trailing P/E</td><td>32.39</td><td>36.44</td><td>38.64</td></tr>
	<tr><td>Forward P/E</td><td>29.76</td><td>32.79</td><td>31.65</td></tr>
	<tr><td>PEG Ratio (5yr expected)</td><td>2.27</td><td>2.75</td><td>2.44</td></tr>
	<tr><td>Price/Sales</td><td>8.77</td><td>9.80</td><td>9.41</td></tr>
	<tr><td>Price/Book</td><td>42.60</td><td>54.21</td><td>57.14</td></tr>
	<tr><td>Enterprise Value/Revenue</td><td>8.68</td><td>9.71</td><td>9.32</td></tr>
	<tr><td>Enterprise Value/EBITDA</td><td>24.73</td><td>27.92</td><td>26.87</td></tr>
</table>
</body></html>`

func TestParseValuationMeasuresHTML(t *testing.T) {
	measures, err := parseValuationMeasuresHTML(valuationMeasuresHTML)
	if err != nil {
		t.Fatalf("parseValuationMeasuresHTML returned error: %v", err)
	}

	if measures.Empty() {
		t.Fatal("Expected valuation measures to be non-empty")
	}
	if len(measures.Rows) != 9 {
		t.Fatalf("Expected 9 rows, got %d", len(measures.Rows))
	}
	if got, want := measures.Columns[0], "Current"; got != want {
		t.Errorf("Expected first column %q, got %q", want, got)
	}
	if got, ok := measures.Value("Market Cap", "Current"); !ok || got != "3.76T" {
		t.Errorf("Expected Market Cap Current 3.76T, got %q (ok=%v)", got, ok)
	}
	if got, ok := measures.Value("Forward P/E", "12/31/2025"); !ok || got != "32.79" {
		t.Errorf("Expected Forward P/E 12/31/2025 32.79, got %q (ok=%v)", got, ok)
	}
}

func TestParseValuationMeasuresHTMLNoTable(t *testing.T) {
	measures, err := parseValuationMeasuresHTML("<html><body><p>No tables here</p></body></html>")
	if err != nil {
		t.Fatalf("parseValuationMeasuresHTML returned error: %v", err)
	}
	if !measures.Empty() {
		t.Fatal("Expected empty valuation measures")
	}
}
