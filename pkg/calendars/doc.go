// Package calendars provides Yahoo Finance economic calendar functionality.
//
// # Overview
//
// The calendars package allows retrieving various financial calendars including
// earnings announcements, IPOs, economic events, and stock splits.
// This is useful for tracking upcoming market events and financial releases.
//
// # Basic Usage
//
//	cal, err := calendars.New()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer cal.Close()
//
//	// Get earnings calendar
//	earnings, err := cal.Earnings(nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, e := range earnings {
//	    fmt.Printf("%s: %s, EPS Est: %.2f\n",
//	        e.Symbol, e.CompanyName, e.EPSEstimate)
//	}
//
// # Earnings Calendar
//
// Get upcoming and recent earnings announcements:
//
//	earnings, err := cal.Earnings(&models.CalendarOptions{
//	    Limit: 50,
//	})
//	for _, e := range earnings {
//	    fmt.Printf("%s: Est %.2f, Actual %.2f, Surprise %.2f%%\n",
//	        e.Symbol, e.EPSEstimate, e.EPSActual, e.SurprisePercent)
//	}
//
// # IPO Calendar
//
// Get upcoming and recent IPO information:
//
//	ipos, err := cal.IPOs(nil)
//	for _, ipo := range ipos {
//	    fmt.Printf("%s: %s, Price: $%.2f-$%.2f\n",
//	        ipo.Symbol, ipo.CompanyName, ipo.PriceFrom, ipo.PriceTo)
//	}
//
// # Economic Events
//
// Get economic releases and indicators:
//
//	events, err := cal.EconomicEvents(nil)
//	for _, e := range events {
//	    fmt.Printf("%s (%s): Expected %.2f, Actual %.2f\n",
//	        e.Event, e.Region, e.Expected, e.Actual)
//	}
//
// # Stock Splits
//
// Get stock split events:
//
//	splits, err := cal.Splits(nil)
//	for _, s := range splits {
//	    fmt.Printf("%s: %s (%s)\n",
//	        s.Symbol, s.CompanyName, s.Ratio)
//	}
//
// # Custom Date Range
//
// Specify a custom date range for calendar queries:
//
//	start := time.Now()
//	end := start.AddDate(0, 1, 0) // 1 month ahead
//
//	cal, err := calendars.New(calendars.WithDateRange(start, end))
//	// Or per-query:
//	earnings, err := cal.Earnings(&models.CalendarOptions{
//	    Start: start,
//	    End:   end,
//	    Limit: 100,
//	})
//
// # Calendar Options
//
// Configure calendar queries with CalendarOptions:
//
//	opts := &models.CalendarOptions{
//	    Start:  time.Now(),
//	    End:    time.Now().AddDate(0, 0, 30),
//	    Limit:  50,   // Max results (capped at 100)
//	    Offset: 0,    // Pagination offset
//	}
//	earnings, err := cal.Earnings(opts)
//
// # Custom Client
//
// Provide a custom HTTP client:
//
//	c, _ := client.New()
//	cal, _ := calendars.New(calendars.WithClient(c))
//
// # Thread Safety
//
// All Calendars methods are safe for concurrent use from multiple goroutines.
//
// # Python Compatibility
//
// This package implements the same functionality as Python yfinance's Calendars class:
//
//	Python                                  | Go
//	----------------------------------------|----------------------------------------
//	yf.Calendars()                          | calendars.New()
//	yf.Calendars(start, end)                | calendars.New(WithDateRange(start, end))
//	cal.get_earnings_calendar(limit)        | cal.Earnings(&CalendarOptions{Limit: n})
//	cal.get_ipo_info_calendar()             | cal.IPOs(nil)
//	cal.get_economic_events_calendar()      | cal.EconomicEvents(nil)
//	cal.get_splits_calendar()               | cal.Splits(nil)
//	cal.earnings_calendar                   | cal.Earnings(nil)
//	cal.ipo_info_calendar                   | cal.IPOs(nil)
//	cal.economic_events_calendar            | cal.EconomicEvents(nil)
//	cal.splits_calendar                     | cal.Splits(nil)
package calendars
