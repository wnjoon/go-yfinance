package ticker

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/wnjoon/go-yfinance/internal/endpoints"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

// cachedExpirations stores expiration dates after first fetch
type optionsCache struct {
	expirations map[string]int64 // date string -> unix timestamp
	strikes     []float64
}

// Options returns all available expiration dates for options.
func (t *Ticker) Options() ([]time.Time, error) {
	if t.optionsCache != nil && len(t.optionsCache.expirations) > 0 {
		return t.getExpirationTimes(), nil
	}

	resp, err := t.fetchOptions("")
	if err != nil {
		return nil, err
	}

	return t.parseExpirations(resp)
}

// OptionChain returns the option chain for a specific expiration date.
// If date is empty, returns the nearest expiration.
func (t *Ticker) OptionChain(date string) (*models.OptionChain, error) {
	// If no date specified, fetch default (nearest expiration)
	if date == "" {
		resp, err := t.fetchOptions("")
		if err != nil {
			return nil, err
		}
		return t.parseOptionChain(resp)
	}

	// Ensure we have expiration dates cached
	if t.optionsCache == nil || len(t.optionsCache.expirations) == 0 {
		_, err := t.Options()
		if err != nil {
			return nil, err
		}
	}

	// Look up the unix timestamp for the date
	timestamp, ok := t.optionsCache.expirations[date]
	if !ok {
		available := make([]string, 0, len(t.optionsCache.expirations))
		for k := range t.optionsCache.expirations {
			available = append(available, k)
		}
		return nil, fmt.Errorf("expiration date %s not found, available: %v", date, available)
	}

	resp, err := t.fetchOptions(fmt.Sprintf("%d", timestamp))
	if err != nil {
		return nil, err
	}

	return t.parseOptionChain(resp)
}

// OptionChainAtExpiry is an alias for OptionChain with a specific date.
func (t *Ticker) OptionChainAtExpiry(date time.Time) (*models.OptionChain, error) {
	return t.OptionChain(date.Format("2006-01-02"))
}

// fetchOptions fetches options data from Yahoo Finance API.
func (t *Ticker) fetchOptions(dateParam string) (*models.OptionChainResponse, error) {
	apiURL := fmt.Sprintf("%s/%s", endpoints.OptionsURL, t.symbol)

	params := url.Values{}
	if dateParam != "" {
		params.Set("date", dateParam)
	}

	params, err := t.auth.AddCrumbToParams(params)
	if err != nil {
		return nil, fmt.Errorf("failed to add crumb: %w", err)
	}

	var resp models.OptionChainResponse
	err = t.client.GetJSON(apiURL, params, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch options: %w", err)
	}

	if resp.OptionChain.Error != nil {
		return nil, fmt.Errorf("API error: %s - %s",
			resp.OptionChain.Error.Code,
			resp.OptionChain.Error.Description)
	}

	if len(resp.OptionChain.Result) == 0 {
		return nil, fmt.Errorf("no options data available for %s", t.symbol)
	}

	return &resp, nil
}

// parseExpirations extracts and caches expiration dates from the response.
func (t *Ticker) parseExpirations(resp *models.OptionChainResponse) ([]time.Time, error) {
	if len(resp.OptionChain.Result) == 0 {
		return nil, fmt.Errorf("no options data in response")
	}

	result := resp.OptionChain.Result[0]

	// Initialize cache
	t.optionsCache = &optionsCache{
		expirations: make(map[string]int64),
		strikes:     result.Strikes,
	}

	// Parse expiration dates
	times := make([]time.Time, 0, len(result.ExpirationDates))
	for _, ts := range result.ExpirationDates {
		expTime := time.Unix(ts, 0)
		dateStr := expTime.Format("2006-01-02")
		t.optionsCache.expirations[dateStr] = ts
		times = append(times, expTime)
	}

	return times, nil
}

// parseOptionChain parses the option chain from the API response.
func (t *Ticker) parseOptionChain(resp *models.OptionChainResponse) (*models.OptionChain, error) {
	if len(resp.OptionChain.Result) == 0 {
		return nil, fmt.Errorf("no options data in response")
	}

	result := resp.OptionChain.Result[0]

	// Cache expirations if not already cached
	if t.optionsCache == nil {
		t.optionsCache = &optionsCache{
			expirations: make(map[string]int64),
			strikes:     result.Strikes,
		}
		for _, ts := range result.ExpirationDates {
			expTime := time.Unix(ts, 0)
			dateStr := expTime.Format("2006-01-02")
			t.optionsCache.expirations[dateStr] = ts
		}
	}

	if len(result.Options) == 0 {
		return &models.OptionChain{
			Calls:      []models.Option{},
			Puts:       []models.Option{},
			Underlying: &result.Quote,
		}, nil
	}

	opt := result.Options[0]

	return &models.OptionChain{
		Calls:      opt.Calls,
		Puts:       opt.Puts,
		Underlying: &result.Quote,
		Expiration: time.Unix(opt.ExpirationDate, 0),
	}, nil
}

// getExpirationTimes returns cached expiration dates as time.Time slice.
func (t *Ticker) getExpirationTimes() []time.Time {
	if t.optionsCache == nil {
		return nil
	}

	times := make([]time.Time, 0, len(t.optionsCache.expirations))
	for _, ts := range t.optionsCache.expirations {
		times = append(times, time.Unix(ts, 0))
	}
	return times
}

// Strikes returns available strike prices for options.
func (t *Ticker) Strikes() ([]float64, error) {
	if t.optionsCache != nil && len(t.optionsCache.strikes) > 0 {
		return t.optionsCache.strikes, nil
	}

	// Fetch options to populate cache
	_, err := t.Options()
	if err != nil {
		return nil, err
	}

	if t.optionsCache == nil {
		return nil, fmt.Errorf("no strikes data available")
	}

	return t.optionsCache.strikes, nil
}

// OptionsJSON returns raw JSON response for debugging.
func (t *Ticker) OptionsJSON() ([]byte, error) {
	resp, err := t.fetchOptions("")
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(resp, "", "  ")
}
