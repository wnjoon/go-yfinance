package calendars

import (
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func TestNew(t *testing.T) {
	cal, err := New()
	if err != nil {
		t.Fatalf("Failed to create Calendars: %v", err)
	}
	defer cal.Close()

	if cal == nil {
		t.Fatal("Calendars should not be nil")
	}

	if cal.ownsClient != true {
		t.Error("ownsClient should be true when no custom client is provided")
	}

	// Default date range should be now to 7 days from now
	now := time.Now()
	if cal.start.Day() != now.Day() {
		t.Errorf("Expected start day %d, got %d", now.Day(), cal.start.Day())
	}

	expected := now.AddDate(0, 0, 7)
	if cal.end.Day() != expected.Day() {
		t.Errorf("Expected end day %d, got %d", expected.Day(), cal.end.Day())
	}
}

func TestNewWithClient(t *testing.T) {
	cal1, err := New()
	if err != nil {
		t.Fatalf("Failed to create first Calendars: %v", err)
	}

	cal2, err := New(WithClient(cal1.client))
	if err != nil {
		t.Fatalf("Failed to create second Calendars: %v", err)
	}

	if cal2.ownsClient != false {
		t.Error("ownsClient should be false when custom client is provided")
	}

	if cal2.client != cal1.client {
		t.Error("Client should be the same instance")
	}

	cal1.Close()
}

func TestNewWithDateRange(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	cal, err := New(WithDateRange(start, end))
	if err != nil {
		t.Fatalf("Failed to create Calendars: %v", err)
	}
	defer cal.Close()

	if cal.start != start {
		t.Errorf("Expected start %v, got %v", start, cal.start)
	}

	if cal.end != end {
		t.Errorf("Expected end %v, got %v", end, cal.end)
	}
}

func TestCalendarConfigs(t *testing.T) {
	tests := []struct {
		calType       models.CalendarType
		expectedField string
	}{
		{models.CalendarEarnings, "intradaymarketcap"},
		{models.CalendarIPO, "startdatetime"},
		{models.CalendarEconomicEvents, "startdatetime"},
		{models.CalendarSplits, "startdatetime"},
	}

	for _, tt := range tests {
		config, ok := calendarConfigs[tt.calType]
		if !ok {
			t.Errorf("Config for %s should exist", tt.calType)
			continue
		}

		if config.sortField != tt.expectedField {
			t.Errorf("Expected sortField '%s' for %s, got '%s'",
				tt.expectedField, tt.calType, config.sortField)
		}

		if len(config.includeFields) == 0 {
			t.Errorf("includeFields for %s should not be empty", tt.calType)
		}
	}
}

func TestBuildDateQuery(t *testing.T) {
	cal, err := New()
	if err != nil {
		t.Fatalf("Failed to create Calendars: %v", err)
	}
	defer cal.Close()

	q := cal.buildDateQuery(nil)

	if q.Operator != "AND" {
		t.Errorf("Expected operator 'AND', got '%s'", q.Operator)
	}

	if len(q.Operands) != 2 {
		t.Errorf("Expected 2 operands, got %d", len(q.Operands))
	}
}

func TestBuildDateQueryWithOptions(t *testing.T) {
	cal, err := New()
	if err != nil {
		t.Fatalf("Failed to create Calendars: %v", err)
	}
	defer cal.Close()

	opts := &models.CalendarOptions{
		Start: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC),
	}

	q := cal.buildDateQuery(opts)

	// Check that the query uses the options dates
	if len(q.Operands) != 2 {
		t.Errorf("Expected 2 operands, got %d", len(q.Operands))
	}
}

func TestMakeColumnIndex(t *testing.T) {
	columns := []string{"Symbol", "Company Name", "Price"}
	idx := makeColumnIndex(columns)

	if idx["Symbol"] != 0 {
		t.Errorf("Expected Symbol at index 0, got %d", idx["Symbol"])
	}

	if idx["Company Name"] != 1 {
		t.Errorf("Expected Company Name at index 1, got %d", idx["Company Name"])
	}

	if idx["Price"] != 2 {
		t.Errorf("Expected Price at index 2, got %d", idx["Price"])
	}

	if _, ok := idx["Missing"]; ok {
		t.Error("Missing key should not exist in index")
	}
}

func TestGetStringAt(t *testing.T) {
	colIdx := map[string]int{"Name": 0, "Type": 1}
	row := []interface{}{"Apple Inc.", "EQUITY"}

	// Valid string
	if got := getStringAt(row, colIdx, "Name"); got != "Apple Inc." {
		t.Errorf("Expected 'Apple Inc.', got '%s'", got)
	}

	// Missing column
	if got := getStringAt(row, colIdx, "Missing"); got != "" {
		t.Errorf("Expected empty string for missing column, got '%s'", got)
	}

	// Index out of range
	colIdx["OutOfRange"] = 10
	if got := getStringAt(row, colIdx, "OutOfRange"); got != "" {
		t.Errorf("Expected empty string for out of range index, got '%s'", got)
	}
}

func TestGetFloatAt(t *testing.T) {
	colIdx := map[string]int{"Float": 0, "Int": 1, "Int64": 2, "String": 3}
	row := []interface{}{3.14, 42, int64(1234567890), "text"}

	// float64
	if got := getFloatAt(row, colIdx, "Float"); got != 3.14 {
		t.Errorf("Expected 3.14, got %f", got)
	}

	// int
	if got := getFloatAt(row, colIdx, "Int"); got != 42.0 {
		t.Errorf("Expected 42.0, got %f", got)
	}

	// int64
	if got := getFloatAt(row, colIdx, "Int64"); got != 1234567890.0 {
		t.Errorf("Expected 1234567890.0, got %f", got)
	}

	// Non-numeric type
	if got := getFloatAt(row, colIdx, "String"); got != 0 {
		t.Errorf("Expected 0 for non-numeric, got %f", got)
	}

	// Missing column
	if got := getFloatAt(row, colIdx, "Missing"); got != 0 {
		t.Errorf("Expected 0 for missing column, got %f", got)
	}
}

func TestClearCache(t *testing.T) {
	cal, err := New()
	if err != nil {
		t.Fatalf("Failed to create Calendars: %v", err)
	}
	defer cal.Close()

	// Add something to cache manually
	cal.mu.Lock()
	cal.cache[models.CalendarEarnings] = []models.EarningsEvent{}
	cal.mu.Unlock()

	// Verify cache has data
	cal.mu.RLock()
	hasData := len(cal.cache) > 0
	cal.mu.RUnlock()
	if !hasData {
		t.Error("Cache should have data")
	}

	// Clear cache
	cal.ClearCache()

	// Verify cache is empty
	cal.mu.RLock()
	isEmpty := len(cal.cache) == 0
	cal.mu.RUnlock()
	if !isEmpty {
		t.Error("Cache should be empty after ClearCache")
	}
}

func TestDefaultCalendarOptions(t *testing.T) {
	opts := models.DefaultCalendarOptions()

	now := time.Now()
	if opts.Start.Day() != now.Day() {
		t.Errorf("Expected start day %d, got %d", now.Day(), opts.Start.Day())
	}

	expected := now.AddDate(0, 0, 7)
	if opts.End.Day() != expected.Day() {
		t.Errorf("Expected end day %d, got %d", expected.Day(), opts.End.Day())
	}

	if opts.Limit != 12 {
		t.Errorf("Expected limit 12, got %d", opts.Limit)
	}

	if opts.Offset != 0 {
		t.Errorf("Expected offset 0, got %d", opts.Offset)
	}
}

func TestCalendarTypes(t *testing.T) {
	tests := []struct {
		calType  models.CalendarType
		expected string
	}{
		{models.CalendarEarnings, "sp_earnings"},
		{models.CalendarIPO, "ipo_info"},
		{models.CalendarEconomicEvents, "economic_event"},
		{models.CalendarSplits, "splits"},
	}

	for _, tt := range tests {
		if string(tt.calType) != tt.expected {
			t.Errorf("CalendarType %v: expected '%s', got '%s'",
				tt.calType, tt.expected, string(tt.calType))
		}
	}
}

func TestParseEarnings(t *testing.T) {
	cal, err := New()
	if err != nil {
		t.Fatalf("Failed to create Calendars: %v", err)
	}
	defer cal.Close()

	columns := []string{"Symbol", "Company Name", "Market Cap (Intraday)", "EPS Estimate", "Reported EPS"}
	rows := [][]interface{}{
		{"AAPL", "Apple Inc.", 3000000000000.0, 1.50, 1.52},
		{"MSFT", "Microsoft Corporation", 2500000000000.0, 2.25, 2.30},
	}

	events := cal.parseEarnings(rows, columns)

	if len(events) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(events))
	}

	if events[0].Symbol != "AAPL" {
		t.Errorf("Expected symbol 'AAPL', got '%s'", events[0].Symbol)
	}

	if events[0].CompanyName != "Apple Inc." {
		t.Errorf("Expected company 'Apple Inc.', got '%s'", events[0].CompanyName)
	}

	if events[0].EPSEstimate != 1.50 {
		t.Errorf("Expected EPS estimate 1.50, got %f", events[0].EPSEstimate)
	}

	if events[1].Symbol != "MSFT" {
		t.Errorf("Expected symbol 'MSFT', got '%s'", events[1].Symbol)
	}
}

func TestParseEarningsEmpty(t *testing.T) {
	cal, err := New()
	if err != nil {
		t.Fatalf("Failed to create Calendars: %v", err)
	}
	defer cal.Close()

	// Empty rows
	events := cal.parseEarnings(nil, nil)
	if len(events) != 0 {
		t.Errorf("Expected empty events, got %d", len(events))
	}

	// Row with empty symbol should be skipped
	columns := []string{"Symbol", "Company Name"}
	rows := [][]interface{}{
		{"", "Unknown Company"},
	}
	events = cal.parseEarnings(rows, columns)
	if len(events) != 0 {
		t.Errorf("Expected empty events for empty symbol, got %d", len(events))
	}
}

func TestParseIPOs(t *testing.T) {
	cal, err := New()
	if err != nil {
		t.Fatalf("Failed to create Calendars: %v", err)
	}
	defer cal.Close()

	columns := []string{"Symbol", "Company Name", "Exchange Short Name", "Price From", "Price To"}
	rows := [][]interface{}{
		{"NEWCO", "New Company Inc.", "NYSE", 15.0, 18.0},
	}

	events := cal.parseIPOs(rows, columns)

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	if events[0].Symbol != "NEWCO" {
		t.Errorf("Expected symbol 'NEWCO', got '%s'", events[0].Symbol)
	}

	if events[0].Exchange != "NYSE" {
		t.Errorf("Expected exchange 'NYSE', got '%s'", events[0].Exchange)
	}

	if events[0].PriceFrom != 15.0 {
		t.Errorf("Expected price from 15.0, got %f", events[0].PriceFrom)
	}

	if events[0].PriceTo != 18.0 {
		t.Errorf("Expected price to 18.0, got %f", events[0].PriceTo)
	}
}

func TestParseEconomicEvents(t *testing.T) {
	cal, err := New()
	if err != nil {
		t.Fatalf("Failed to create Calendars: %v", err)
	}
	defer cal.Close()

	columns := []string{"Event", "Country Code", "Market Expectation", "Actual"}
	rows := [][]interface{}{
		{"GDP Release", "US", 2.5, 2.7},
	}

	events := cal.parseEconomicEvents(rows, columns)

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	if events[0].Event != "GDP Release" {
		t.Errorf("Expected event 'GDP Release', got '%s'", events[0].Event)
	}

	if events[0].Region != "US" {
		t.Errorf("Expected region 'US', got '%s'", events[0].Region)
	}

	if events[0].Expected != 2.5 {
		t.Errorf("Expected expectation 2.5, got %f", events[0].Expected)
	}

	if events[0].Actual != 2.7 {
		t.Errorf("Expected actual 2.7, got %f", events[0].Actual)
	}
}

func TestParseSplits(t *testing.T) {
	cal, err := New()
	if err != nil {
		t.Fatalf("Failed to create Calendars: %v", err)
	}
	defer cal.Close()

	columns := []string{"Symbol", "Company Name", "Old Share Worth", "New Share Worth", "Optionable?"}
	rows := [][]interface{}{
		{"XYZ", "XYZ Corp", 1.0, 4.0, "Yes"},
	}

	events := cal.parseSplits(rows, columns)

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	if events[0].Symbol != "XYZ" {
		t.Errorf("Expected symbol 'XYZ', got '%s'", events[0].Symbol)
	}

	if events[0].OldShareWorth != 1.0 {
		t.Errorf("Expected old share worth 1.0, got %f", events[0].OldShareWorth)
	}

	if events[0].NewShareWorth != 4.0 {
		t.Errorf("Expected new share worth 4.0, got %f", events[0].NewShareWorth)
	}

	if events[0].Ratio != "4:1" {
		t.Errorf("Expected ratio '4:1', got '%s'", events[0].Ratio)
	}

	if events[0].Optionable != true {
		t.Errorf("Expected optionable true, got %v", events[0].Optionable)
	}
}

// Integration tests (require network access)
// Run with: go test -v -run Integration

func TestEarningsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cal, err := New()
	if err != nil {
		t.Fatalf("Failed to create Calendars: %v", err)
	}
	defer cal.Close()

	earnings, err := cal.Earnings(nil)
	if err != nil {
		t.Fatalf("Earnings() failed: %v", err)
	}

	t.Logf("Found %d earnings events", len(earnings))

	// Print first few results
	for i, e := range earnings {
		if i >= 5 {
			break
		}
		t.Logf("%s (%s): Est %.2f, Actual %.2f",
			e.Symbol, e.CompanyName, e.EPSEstimate, e.EPSActual)
	}
}

func TestEarningsWithOptionsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cal, err := New()
	if err != nil {
		t.Fatalf("Failed to create Calendars: %v", err)
	}
	defer cal.Close()

	opts := &models.CalendarOptions{
		Limit: 5,
	}

	earnings, err := cal.Earnings(opts)
	if err != nil {
		t.Fatalf("Earnings() failed: %v", err)
	}

	if len(earnings) > 5 {
		t.Errorf("Expected at most 5 results, got %d", len(earnings))
	}

	t.Logf("Found %d earnings events with limit 5", len(earnings))
}

func TestIPOsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Use wider date range for IPOs as they may be sparse
	start := time.Now().AddDate(0, -1, 0) // 1 month ago
	end := time.Now().AddDate(0, 1, 0)    // 1 month ahead

	cal, err := New(WithDateRange(start, end))
	if err != nil {
		t.Fatalf("Failed to create Calendars: %v", err)
	}
	defer cal.Close()

	ipos, err := cal.IPOs(nil)
	if err != nil {
		t.Fatalf("IPOs() failed: %v", err)
	}

	t.Logf("Found %d IPO events", len(ipos))

	// Print first few results
	for i, ipo := range ipos {
		if i >= 5 {
			break
		}
		t.Logf("%s (%s): $%.2f-$%.2f on %s",
			ipo.Symbol, ipo.CompanyName, ipo.PriceFrom, ipo.PriceTo, ipo.Exchange)
	}
}

func TestEconomicEventsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cal, err := New()
	if err != nil {
		t.Fatalf("Failed to create Calendars: %v", err)
	}
	defer cal.Close()

	events, err := cal.EconomicEvents(nil)
	if err != nil {
		t.Fatalf("EconomicEvents() failed: %v", err)
	}

	t.Logf("Found %d economic events", len(events))

	// Print first few results
	for i, e := range events {
		if i >= 5 {
			break
		}
		t.Logf("%s (%s): Expected %.2f, Actual %.2f",
			e.Event, e.Region, e.Expected, e.Actual)
	}
}

func TestSplitsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Use wider date range for splits as they may be sparse
	start := time.Now().AddDate(0, -1, 0) // 1 month ago
	end := time.Now().AddDate(0, 1, 0)    // 1 month ahead

	cal, err := New(WithDateRange(start, end))
	if err != nil {
		t.Fatalf("Failed to create Calendars: %v", err)
	}
	defer cal.Close()

	splits, err := cal.Splits(nil)
	if err != nil {
		t.Fatalf("Splits() failed: %v", err)
	}

	t.Logf("Found %d split events", len(splits))

	// Print first few results
	for i, s := range splits {
		if i >= 5 {
			break
		}
		t.Logf("%s (%s): %s",
			s.Symbol, s.CompanyName, s.Ratio)
	}
}

func TestCalendarCacheIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cal, err := New()
	if err != nil {
		t.Fatalf("Failed to create Calendars: %v", err)
	}
	defer cal.Close()

	// First call - fetch from API
	earnings1, err := cal.Earnings(&models.CalendarOptions{Limit: 3})
	if err != nil {
		t.Fatalf("First Earnings() failed: %v", err)
	}

	t.Logf("First call: %d events", len(earnings1))

	// Fetch economic events (different cache key)
	events, err := cal.EconomicEvents(&models.CalendarOptions{Limit: 3})
	if err != nil {
		t.Fatalf("EconomicEvents() failed: %v", err)
	}

	t.Logf("Economic events: %d events", len(events))

	// Clear cache
	cal.ClearCache()

	// Fetch again
	earnings2, err := cal.Earnings(&models.CalendarOptions{Limit: 3})
	if err != nil {
		t.Fatalf("Second Earnings() failed: %v", err)
	}

	t.Logf("After cache clear: %d events", len(earnings2))
}

func TestDateRangeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Custom date range
	start := time.Now().AddDate(0, -3, 0) // 3 months ago
	end := time.Now().AddDate(0, 3, 0)    // 3 months ahead

	cal, err := New(WithDateRange(start, end))
	if err != nil {
		t.Fatalf("Failed to create Calendars: %v", err)
	}
	defer cal.Close()

	earnings, err := cal.Earnings(&models.CalendarOptions{Limit: 50})
	if err != nil {
		t.Fatalf("Earnings() failed: %v", err)
	}

	t.Logf("Found %d earnings events in 6-month range", len(earnings))
}
