package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/wnjoon/go-yfinance/internal/endpoints"
)

// AuthStrategy represents the authentication strategy.
type AuthStrategy int

const (
	// StrategyBasic uses fc.yahoo.com for cookie acquisition.
	StrategyBasic AuthStrategy = iota
	// StrategyCSRF uses guce.yahoo.com consent flow for cookie acquisition.
	StrategyCSRF
)

// AuthManager handles Yahoo Finance authentication (Cookie + Crumb).
type AuthManager struct {
	client   *Client
	mu       sync.RWMutex
	cookie   string
	crumb    string
	strategy AuthStrategy
	expiry   time.Time
	user     map[string]interface{}
}

// NewAuthManager creates a new AuthManager with the given client.
func NewAuthManager(client *Client) *AuthManager {
	return &AuthManager{
		client:   client,
		strategy: StrategyBasic,
	}
}

// SetLoginCookies sets manually retrieved Yahoo Finance login cookies.
func (a *AuthManager) SetLoginCookies(cookieT, cookieY string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.client.SetCookies(map[string]string{
		"T": cookieT,
		"Y": cookieY,
	})
	a.cookie = "T,Y"
	a.user = nil
}

// CheckLogin checks whether the current Yahoo cookies represent a logged-in user.
func (a *AuthManager) CheckLogin() (bool, error) {
	a.mu.RLock()
	if a.user != nil {
		a.mu.RUnlock()
		return true, nil
	}
	a.mu.RUnlock()

	resp, err := a.client.Get(endpoints.RootURL, nil)
	if err != nil {
		return false, err
	}

	user, ok, err := parseLoginUser(resp.Body)
	if err != nil || !ok {
		return ok, err
	}

	a.mu.Lock()
	a.user = user
	a.mu.Unlock()
	return true, nil
}

// User returns the cached logged-in Yahoo user payload, if available.
func (a *AuthManager) User() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.user == nil {
		return nil
	}
	user := make(map[string]interface{}, len(a.user))
	for k, v := range a.user {
		user[k] = v
	}
	return user
}

// GetCrumb returns the current crumb, fetching it if necessary.
func (a *AuthManager) GetCrumb() (string, error) {
	a.mu.RLock()
	if a.crumb != "" && time.Now().Before(a.expiry) {
		crumb := a.crumb
		a.mu.RUnlock()
		return crumb, nil
	}
	a.mu.RUnlock()

	return a.refreshAuth()
}

// refreshAuth fetches new cookie and crumb.
func (a *AuthManager) refreshAuth() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Double-check after acquiring write lock
	if a.crumb != "" && time.Now().Before(a.expiry) {
		return a.crumb, nil
	}

	var err error
	if a.strategy == StrategyBasic {
		err = a.fetchBasic()
		if err != nil {
			// Fallback to CSRF strategy
			a.strategy = StrategyCSRF
			err = a.fetchCSRF()
		}
	} else {
		err = a.fetchCSRF()
		if err != nil {
			// Fallback to Basic strategy
			a.strategy = StrategyBasic
			err = a.fetchBasic()
		}
	}

	if err != nil {
		return "", fmt.Errorf("authentication failed: %w", err)
	}

	return a.crumb, nil
}

// fetchBasic implements the basic authentication strategy.
// 1. GET https://fc.yahoo.com -> captures cookies
// 2. GET https://query2.finance.yahoo.com/v1/test/getcrumb -> gets crumb
func (a *AuthManager) fetchBasic() error {
	// Step 1: Get cookie from fc.yahoo.com
	resp, err := a.client.Get(endpoints.CookieURL, nil)
	if err != nil {
		return fmt.Errorf("failed to get cookie: %w", err)
	}

	// Extract cookies from response headers
	a.extractCookies(resp.Headers)

	// Step 2: Get crumb
	resp, err = a.client.Get(endpoints.CrumbURL, nil)
	if err != nil {
		return fmt.Errorf("failed to get crumb: %w", err)
	}

	if resp.StatusCode == 429 || strings.Contains(resp.Body, "Too Many Requests") {
		return fmt.Errorf("rate limited")
	}

	if resp.Body == "" || strings.Contains(resp.Body, "<html>") {
		return fmt.Errorf("invalid crumb response")
	}

	a.crumb = strings.TrimSpace(resp.Body)
	a.expiry = time.Now().Add(1 * time.Hour) // Crumb typically valid for ~1 hour

	return nil
}

// fetchCSRF implements the CSRF consent-based authentication strategy.
// This is used when basic strategy fails (e.g., for EU users).
func (a *AuthManager) fetchCSRF() error {
	// Step 1: Get consent page
	resp, err := a.client.Get(endpoints.ConsentURL, nil)
	if err != nil {
		return fmt.Errorf("failed to get consent page: %w", err)
	}

	// Extract CSRF token and session ID from HTML
	csrfToken := extractInputValue(resp.Body, "csrfToken")
	sessionID := extractInputValue(resp.Body, "sessionId")

	if csrfToken == "" || sessionID == "" {
		return fmt.Errorf("failed to extract CSRF tokens")
	}

	// Step 2: Submit consent
	consentData := map[string]string{
		"agree":           "agree",
		"consentUUID":     "default",
		"sessionId":       sessionID,
		"csrfToken":       csrfToken,
		"originalDoneUrl": "https://finance.yahoo.com/",
		"namespace":       "yahoo",
	}

	collectURL := fmt.Sprintf("%s?sessionId=%s", endpoints.CollectConsentURL, sessionID)
	_, err = a.client.Post(collectURL, nil, consentData)
	if err != nil {
		return fmt.Errorf("failed to submit consent: %w", err)
	}

	// Step 3: Copy consent
	copyURL := fmt.Sprintf("%s?sessionId=%s", endpoints.CopyConsentURL, sessionID)
	_, err = a.client.Get(copyURL, nil)
	if err != nil {
		return fmt.Errorf("failed to copy consent: %w", err)
	}

	// Step 4: Get crumb
	resp, err = a.client.Get(endpoints.CrumbCSRFURL, nil)
	if err != nil {
		return fmt.Errorf("failed to get crumb: %w", err)
	}

	if resp.StatusCode == 429 || strings.Contains(resp.Body, "Too Many Requests") {
		return fmt.Errorf("rate limited")
	}

	if resp.Body == "" || strings.Contains(resp.Body, "<html>") {
		return fmt.Errorf("invalid crumb response")
	}

	a.crumb = strings.TrimSpace(resp.Body)
	a.expiry = time.Now().Add(1 * time.Hour)

	return nil
}

// extractCookies extracts and stores cookies from response headers.
func (a *AuthManager) extractCookies(headers map[string]string) {
	for key, value := range headers {
		if strings.ToLower(key) == "set-cookie" {
			// Extract just the cookie name=value part (before any attributes like Expires, Path, etc.)
			parts := strings.Split(value, ";")
			if len(parts) > 0 {
				a.cookie = strings.TrimSpace(parts[0])
				// Set cookie on the client for subsequent requests
				a.client.SetCookie(a.cookie)
			}
			break
		}
	}
}

// extractInputValue extracts value from HTML input element by name.
func extractInputValue(html, name string) string {
	// Pattern: <input name="NAME" ... value="VALUE" ...>
	// or: <input ... name="NAME" ... value="VALUE" ...>
	pattern := fmt.Sprintf(`<input[^>]*name=["']%s["'][^>]*value=["']([^"']*)["']`, regexp.QuoteMeta(name))
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1]
	}

	// Try alternate order: value before name
	pattern = fmt.Sprintf(`<input[^>]*value=["']([^"']*)["'][^>]*name=["']%s["']`, regexp.QuoteMeta(name))
	re = regexp.MustCompile(pattern)
	matches = re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// AddCrumbToParams adds the crumb parameter to URL values.
func (a *AuthManager) AddCrumbToParams(params url.Values) (url.Values, error) {
	crumb, err := a.GetCrumb()
	if err != nil {
		return params, err
	}

	if params == nil {
		params = url.Values{}
	}
	params.Set("crumb", crumb)
	return params, nil
}

// Reset clears the authentication state.
func (a *AuthManager) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.cookie = ""
	a.crumb = ""
	a.expiry = time.Time{}
	a.user = nil
}

// SwitchStrategy switches to the alternate authentication strategy.
func (a *AuthManager) SwitchStrategy() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.strategy == StrategyBasic {
		a.strategy = StrategyCSRF
	} else {
		a.strategy = StrategyBasic
	}

	// Clear existing auth
	a.cookie = ""
	a.crumb = ""
	a.expiry = time.Time{}
	a.user = nil
}

func parseLoginUser(html string) (map[string]interface{}, bool, error) {
	re := regexp.MustCompile(`(?s)<script[^>]*id=["']nimbus-benji-config["'][^>]*>(.*?)</script>`)
	matches := re.FindStringSubmatch(html)
	if len(matches) < 2 {
		return nil, false, nil
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(matches[1]), &payload); err != nil {
		return nil, false, err
	}

	i13n, ok := payload["i13n"].(map[string]interface{})
	if !ok {
		return nil, false, nil
	}
	user, ok := i13n["user"].(map[string]interface{})
	if !ok {
		return nil, false, nil
	}
	if guid, ok := user["guid"].(string); !ok || guid == "" {
		return nil, false, nil
	}

	return user, true, nil
}
