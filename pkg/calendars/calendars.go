package calendars

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/wnjoon/go-yfinance/internal/endpoints"
	"github.com/wnjoon/go-yfinance/pkg/client"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

const dateFormat = "2006-01-02"

// calendarConfig defines the configuration for each calendar type.
type calendarConfig struct {
	sortField     string
	includeFields []string
}

var calendarConfigs = map[models.CalendarType]calendarConfig{
	models.CalendarEarnings: {
		sortField: "intradaymarketcap",
		includeFields: []string{
			"ticker", "companyshortname", "intradaymarketcap", "eventname",
			"startdatetime", "startdatetimetype", "epsestimate", "epsactual", "epssurprisepct",
		},
	},
	models.CalendarIPO: {
		sortField: "startdatetime",
		includeFields: []string{
			"ticker", "companyshortname", "exchange_short_name", "filingdate",
			"startdatetime", "amendeddate", "pricefrom", "priceto", "offerprice",
			"currencyname", "shares", "dealtype",
		},
	},
	models.CalendarEconomicEvents: {
		sortField: "startdatetime",
		includeFields: []string{
			"econ_release", "country_code", "startdatetime", "period",
			"after_release_actual", "consensus_estimate", "prior_release_actual",
			"originally_reported_actual",
		},
	},
	models.CalendarSplits: {
		sortField: "startdatetime",
		includeFields: []string{
			"ticker", "companyshortname", "startdatetime", "optionable",
			"old_share_worth", "share_worth",
		},
	},
}

// Calendars provides access to Yahoo Finance economic calendars.
//
// Calendars allows retrieving earnings, IPO, economic events, and stock splits data.
type Calendars struct {
	client     *client.Client
	auth       *client.AuthManager
	ownsClient bool

	start time.Time
	end   time.Time

	// Cached data
	mu    sync.RWMutex
	cache map[models.CalendarType]interface{}
}

// Option is a function that configures a Calendars instance.
type Option func(*Calendars)

// WithClient sets a custom HTTP client for the Calendars instance.
func WithClient(c *client.Client) Option {
	return func(cal *Calendars) {
		cal.client = c
		cal.ownsClient = false
	}
}

// WithDateRange sets the date range for calendar queries.
func WithDateRange(start, end time.Time) Option {
	return func(cal *Calendars) {
		cal.start = start
		cal.end = end
	}
}

// New creates a new Calendars instance.
//
// By default, the date range is from today to 7 days from now.
//
// Example:
//
//	cal, err := calendars.New()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer cal.Close()
//
//	earnings, err := cal.Earnings(nil)
//	for _, e := range earnings {
//	    fmt.Printf("%s: %s, EPS Est: %.2f\n", e.Symbol, e.CompanyName, e.EPSEstimate)
//	}
func New(opts ...Option) (*Calendars, error) {
	now := time.Now()
	cal := &Calendars{
		ownsClient: true,
		start:      now,
		end:        now.AddDate(0, 0, 7),
		cache:      make(map[models.CalendarType]interface{}),
	}

	for _, opt := range opts {
		opt(cal)
	}

	if cal.client == nil {
		c, err := client.New()
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}
		cal.client = c
	}

	cal.auth = client.NewAuthManager(cal.client)

	return cal, nil
}

// Close releases resources used by the Calendars instance.
func (c *Calendars) Close() {
	if c.ownsClient && c.client != nil {
		c.client.Close()
	}
}

// query represents a calendar query condition.
type query struct {
	Operator string        `json:"operator"`
	Operands []interface{} `json:"operands"`
}

// getDateRange returns the effective start and end dates.
func (c *Calendars) getDateRange(opts *models.CalendarOptions) (time.Time, time.Time) {
	start := c.start
	end := c.end
	if opts != nil {
		if !opts.Start.IsZero() {
			start = opts.Start
		}
		if !opts.End.IsZero() {
			end = opts.End
		}
	}
	return start, end
}

// buildDateQueries returns GTE and LTE queries for date range (for flattened queries).
func (c *Calendars) buildDateQueries(opts *models.CalendarOptions) (query, query) {
	start, end := c.getDateRange(opts)
	return query{Operator: "GTE", Operands: []interface{}{"startdatetime", start.Format(dateFormat)}},
		query{Operator: "LTE", Operands: []interface{}{"startdatetime", end.Format(dateFormat)}}
}

// buildDateQuery returns an AND query with date range (for root-level queries).
func (c *Calendars) buildDateQuery(opts *models.CalendarOptions) query {
	start, end := c.getDateRange(opts)
	return query{
		Operator: "AND",
		Operands: []interface{}{
			query{Operator: "GTE", Operands: []interface{}{"startdatetime", start.Format(dateFormat)}},
			query{Operator: "LTE", Operands: []interface{}{"startdatetime", end.Format(dateFormat)}},
		},
	}
}

// fetchCalendar fetches calendar data from the API.
func (c *Calendars) fetchCalendar(calType models.CalendarType, q query, opts *models.CalendarOptions) ([][]interface{}, []string, error) {
	config, ok := calendarConfigs[calType]
	if !ok {
		return nil, nil, fmt.Errorf("unknown calendar type: %s", calType)
	}

	limit := 12
	offset := 0
	if opts != nil {
		if opts.Limit > 0 {
			limit = opts.Limit
			if limit > 100 {
				limit = 100 // Yahoo Finance caps at 100
			}
		}
		offset = opts.Offset
	}

	body := map[string]interface{}{
		"sortType":      "DESC",
		"entityIdType":  string(calType),
		"sortField":     config.sortField,
		"includeFields": config.includeFields,
		"size":          limit,
		"offset":        offset,
		"query":         q,
	}

	params := url.Values{}
	params.Set("lang", "en-US")
	params.Set("region", "US")

	// Add crumb authentication
	params, err := c.auth.AddCrumbToParams(params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to add crumb: %w", err)
	}

	// Marshal body to JSON
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := c.client.PostJSON(endpoints.CalendarURL, params, bodyBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch calendar data: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, nil, client.HTTPStatusToError(resp.StatusCode, resp.Body)
	}

	var raw models.CalendarResponse
	if err := json.Unmarshal([]byte(resp.Body), &raw); err != nil {
		return nil, nil, fmt.Errorf("failed to parse calendar response: %w", err)
	}

	if raw.Finance.Error != nil {
		return nil, nil, fmt.Errorf("calendar API error: %s - %s",
			raw.Finance.Error.Code, raw.Finance.Error.Description)
	}

	if len(raw.Finance.Result) == 0 || len(raw.Finance.Result[0].Documents) == 0 {
		return nil, nil, nil
	}

	doc := raw.Finance.Result[0].Documents[0]

	// Extract column labels
	var columns []string
	for _, col := range doc.Columns {
		columns = append(columns, col.Label)
	}

	return doc.Rows, columns, nil
}

// Earnings retrieves the earnings calendar.
//
// Returns upcoming and recent earnings announcements with EPS estimates.
//
// Example:
//
//	earnings, err := cal.Earnings(nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, e := range earnings {
//	    fmt.Printf("%s: Est %.2f, Actual %.2f, Surprise %.2f%%\n",
//	        e.Symbol, e.EPSEstimate, e.EPSActual, e.SurprisePercent)
//	}
func (c *Calendars) Earnings(opts *models.CalendarOptions) ([]models.EarningsEvent, error) {
	gteQuery, lteQuery := c.buildDateQueries(opts)
	q := query{
		Operator: "AND",
		Operands: []interface{}{
			query{Operator: "EQ", Operands: []interface{}{"region", "us"}},
			query{
				Operator: "OR",
				Operands: []interface{}{
					query{Operator: "EQ", Operands: []interface{}{"eventtype", "EAD"}},
					query{Operator: "EQ", Operands: []interface{}{"eventtype", "ERA"}},
				},
			},
			gteQuery,
			lteQuery,
		},
	}

	rows, columns, err := c.fetchCalendar(models.CalendarEarnings, q, opts)
	if err != nil {
		return nil, err
	}

	return c.parseEarnings(rows, columns), nil
}

// parseEarnings parses earnings data from API response.
func (c *Calendars) parseEarnings(rows [][]interface{}, columns []string) []models.EarningsEvent {
	colIdx := makeColumnIndex(columns)
	var events []models.EarningsEvent

	for _, row := range rows {
		event := models.EarningsEvent{
			Symbol:          getStringAt(row, colIdx, "Symbol"),
			CompanyName:     getStringAt(row, colIdx, "Company Name"),
			MarketCap:       getFloatAt(row, colIdx, "Market Cap (Intraday)"),
			EventName:       getStringAt(row, colIdx, "Event Name"),
			Timing:          getStringAt(row, colIdx, "Timing"),
			EPSEstimate:     getFloatAt(row, colIdx, "EPS Estimate"),
			EPSActual:       getFloatAt(row, colIdx, "Reported EPS"),
			SurprisePercent: getFloatAt(row, colIdx, "Surprise (%)"),
		}

		// Parse event time
		if timeStr := getStringAt(row, colIdx, "Event Start Date"); timeStr != "" {
			if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
				event.EventTime = &t
			}
		}

		if event.Symbol != "" {
			events = append(events, event)
		}
	}

	return events
}

// IPOs retrieves the IPO calendar.
//
// Returns upcoming and recent IPO information.
//
// Example:
//
//	ipos, err := cal.IPOs(nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, ipo := range ipos {
//	    fmt.Printf("%s: %s, Price Range: $%.2f-$%.2f\n",
//	        ipo.Symbol, ipo.CompanyName, ipo.PriceFrom, ipo.PriceTo)
//	}
func (c *Calendars) IPOs(opts *models.CalendarOptions) ([]models.IPOEvent, error) {
	start := c.start
	end := c.end
	if opts != nil {
		if !opts.Start.IsZero() {
			start = opts.Start
		}
		if !opts.End.IsZero() {
			end = opts.End
		}
	}

	startStr := start.Format(dateFormat)
	endStr := end.Format(dateFormat)

	q := query{
		Operator: "OR",
		Operands: []interface{}{
			query{Operator: "GTELT", Operands: []interface{}{"startdatetime", startStr, endStr}},
			query{Operator: "GTELT", Operands: []interface{}{"filingdate", startStr, endStr}},
			query{Operator: "GTELT", Operands: []interface{}{"amendeddate", startStr, endStr}},
		},
	}

	rows, columns, err := c.fetchCalendar(models.CalendarIPO, q, opts)
	if err != nil {
		return nil, err
	}

	return c.parseIPOs(rows, columns), nil
}

// parseIPOs parses IPO data from API response.
func (c *Calendars) parseIPOs(rows [][]interface{}, columns []string) []models.IPOEvent {
	colIdx := makeColumnIndex(columns)
	var events []models.IPOEvent

	for _, row := range rows {
		event := models.IPOEvent{
			Symbol:      getStringAt(row, colIdx, "Symbol"),
			CompanyName: getStringAt(row, colIdx, "Company Name"),
			Exchange:    getStringAt(row, colIdx, "Exchange Short Name"),
			PriceFrom:   getFloatAt(row, colIdx, "Price From"),
			PriceTo:     getFloatAt(row, colIdx, "Price To"),
			OfferPrice:  getFloatAt(row, colIdx, "Price"),
			Currency:    getStringAt(row, colIdx, "Currency Name"),
			Shares:      int64(getFloatAt(row, colIdx, "Shares")),
			DealType:    getStringAt(row, colIdx, "Deal Type"),
		}

		// Parse dates
		if timeStr := getStringAt(row, colIdx, "Filing Date"); timeStr != "" {
			if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
				event.FilingDate = &t
			}
		}
		if timeStr := getStringAt(row, colIdx, "Date"); timeStr != "" {
			if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
				event.Date = &t
			}
		}
		if timeStr := getStringAt(row, colIdx, "Amended Date"); timeStr != "" {
			if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
				event.AmendedDate = &t
			}
		}

		if event.Symbol != "" || event.CompanyName != "" {
			events = append(events, event)
		}
	}

	return events
}

// EconomicEvents retrieves the economic events calendar.
//
// Returns upcoming and recent economic releases and indicators.
//
// Example:
//
//	events, err := cal.EconomicEvents(nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, e := range events {
//	    fmt.Printf("%s (%s): Expected %.2f, Actual %.2f\n",
//	        e.Event, e.Region, e.Expected, e.Actual)
//	}
func (c *Calendars) EconomicEvents(opts *models.CalendarOptions) ([]models.EconomicEvent, error) {
	q := c.buildDateQuery(opts)

	rows, columns, err := c.fetchCalendar(models.CalendarEconomicEvents, q, opts)
	if err != nil {
		return nil, err
	}

	return c.parseEconomicEvents(rows, columns), nil
}

// parseEconomicEvents parses economic events data from API response.
func (c *Calendars) parseEconomicEvents(rows [][]interface{}, columns []string) []models.EconomicEvent {
	colIdx := makeColumnIndex(columns)
	var events []models.EconomicEvent

	for _, row := range rows {
		event := models.EconomicEvent{
			Event:    getStringAt(row, colIdx, "Event"),
			Region:   getStringAt(row, colIdx, "Country Code"),
			Period:   getStringAt(row, colIdx, "Period"),
			Actual:   getFloatAt(row, colIdx, "Actual"),
			Expected: getFloatAt(row, colIdx, "Market Expectation"),
			Last:     getFloatAt(row, colIdx, "Prior to This"),
			Revised:  getFloatAt(row, colIdx, "Revised from"),
		}

		// Parse event time
		if timeStr := getStringAt(row, colIdx, "Event Time"); timeStr != "" {
			if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
				event.EventTime = &t
			}
		}

		if event.Event != "" {
			events = append(events, event)
		}
	}

	return events
}

// Splits retrieves the stock splits calendar.
//
// Returns upcoming and recent stock split events.
//
// Example:
//
//	splits, err := cal.Splits(nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, s := range splits {
//	    fmt.Printf("%s: %s, Old: %.0f, New: %.0f\n",
//	        s.Symbol, s.CompanyName, s.OldShareWorth, s.NewShareWorth)
//	}
func (c *Calendars) Splits(opts *models.CalendarOptions) ([]models.CalendarSplitEvent, error) {
	q := c.buildDateQuery(opts)

	rows, columns, err := c.fetchCalendar(models.CalendarSplits, q, opts)
	if err != nil {
		return nil, err
	}

	return c.parseSplits(rows, columns), nil
}

// parseSplits parses splits data from API response.
func (c *Calendars) parseSplits(rows [][]interface{}, columns []string) []models.CalendarSplitEvent {
	colIdx := makeColumnIndex(columns)
	var events []models.CalendarSplitEvent

	for _, row := range rows {
		event := models.CalendarSplitEvent{
			Symbol:        getStringAt(row, colIdx, "Symbol"),
			CompanyName:   getStringAt(row, colIdx, "Company Name"),
			OldShareWorth: getFloatAt(row, colIdx, "Old Share Worth"),
			NewShareWorth: getFloatAt(row, colIdx, "New Share Worth"),
		}

		// Parse optionable
		if optStr := getStringAt(row, colIdx, "Optionable?"); optStr != "" {
			event.Optionable = optStr == "Yes" || optStr == "true"
		}

		// Parse payable date
		if timeStr := getStringAt(row, colIdx, "Payable On"); timeStr != "" {
			if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
				event.PayableDate = &t
			}
		}

		// Calculate ratio if possible
		if event.OldShareWorth > 0 && event.NewShareWorth > 0 {
			ratio := event.NewShareWorth / event.OldShareWorth
			event.Ratio = fmt.Sprintf("%.0f:1", ratio)
		}

		if event.Symbol != "" {
			events = append(events, event)
		}
	}

	return events
}

// ClearCache clears all cached calendar data.
func (c *Calendars) ClearCache() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[models.CalendarType]interface{})
}

// Helper functions

func makeColumnIndex(columns []string) map[string]int {
	idx := make(map[string]int)
	for i, col := range columns {
		idx[col] = i
	}
	return idx
}

func getStringAt(row []interface{}, colIdx map[string]int, colName string) string {
	idx, ok := colIdx[colName]
	if !ok || idx >= len(row) {
		return ""
	}
	if s, ok := row[idx].(string); ok {
		return s
	}
	return ""
}

func getFloatAt(row []interface{}, colIdx map[string]int, colName string) float64 {
	idx, ok := colIdx[colName]
	if !ok || idx >= len(row) {
		return 0
	}
	switch v := row[idx].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return 0
}
