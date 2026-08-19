package client

import (
	"encoding/json"
	"errors"
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
	client   authClient
	mu       sync.RWMutex
	cookie   string
	crumb    string
	strategy AuthStrategy
	expiry   time.Time
	user     map[string]interface{}
}

type authResponseGetter func(rawURL string, params url.Values) (*Response, error)

type authClient interface {
	Get(rawURL string, params url.Values) (*Response, error)
	Post(rawURL string, params url.Values, body map[string]string) (*Response, error)
	SetCookie(cookie string)
	SetCookies(cookies map[string]string)
}

var subscriptionTierNames = map[int]string{
	6: "gold",
	5: "silver",
	3: "bronze",
}

const htmlElementNames = `a|abbr|acronym|address|applet|area|article|aside|audio|b|base|basefont|bdi|bdo|big|blink|blockquote|body|br|button|canvas|caption|center|cite|code|col|colgroup|content|data|datalist|dd|del|details|dfn|dialog|dir|div|dl|dt|em|embed|fieldset|figcaption|figure|font|footer|form|frame|frameset|h[1-6]|head|header|hgroup|hr|html|i|iframe|image|img|input|ins|kbd|keygen|label|legend|li|link|main|map|mark|marquee|menu|menuitem|meta|meter|nav|nobr|noembed|noframes|noscript|object|ol|optgroup|option|output|p|param|picture|plaintext|pre|progress|q|rp|rt|ruby|s|samp|script|search|section|select|shadow|slot|small|source|span|strike|strong|style|sub|summary|sup|table|tbody|td|template|textarea|tfoot|th|thead|time|title|tr|track|tt|u|ul|var|video|wbr|xmp`

var htmlCrumbPattern = regexp.MustCompile(`(?i)<!doctype\s+html(?:\s|>)|<!--|</?(?:` + htmlElementNames + `)(?:\s|/?>)`)

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
	a.crumb = ""
	a.expiry = time.Time{}
	a.user = nil
}

// SetLoginCookiesAndCheck sets Yahoo Finance login cookies and verifies them.
func (a *AuthManager) SetLoginCookiesAndCheck(cookieT, cookieY string) (bool, error) {
	a.SetLoginCookies(cookieT, cookieY)
	return a.CheckLogin()
}

// CheckLogin checks whether the current Yahoo cookies represent a logged-in user.
func (a *AuthManager) CheckLogin() (bool, error) {
	return a.checkLoginWithGetter(a.client.Get)
}

func (a *AuthManager) checkLoginWithGetter(getter authResponseGetter) (bool, error) {
	entitlement, loggedIn, err := a.fetchEntitlementWithGetter(getter)
	if err != nil {
		return false, err
	}

	a.mu.Lock()
	if loggedIn {
		a.user = userFromEntitlement(entitlement)
	} else {
		a.user = nil
	}
	a.mu.Unlock()
	return loggedIn, nil
}

// SubscriptionTier returns the active Yahoo Finance subscription tier.
func (a *AuthManager) SubscriptionTier() (string, error) {
	return a.subscriptionTierWithGetter(a.client.Get)
}

func (a *AuthManager) subscriptionTierWithGetter(getter authResponseGetter) (string, error) {
	entitlement, loggedIn, err := a.fetchEntitlementWithGetter(getter)
	if err != nil || !loggedIn {
		return "", err
	}
	return subscriptionTier(entitlement), nil
}

func (a *AuthManager) fetchEntitlementWithGetter(getter authResponseGetter) (map[string]interface{}, bool, error) {
	resp, err := getter(endpoints.SubscriptionsURL, nil)
	if err != nil {
		return nil, false, sanitizedAuthTransportError("subscriptions entitlement")
	}
	if resp == nil {
		return nil, false, WrapInvalidResponseError(fmt.Errorf("subscriptions entitlement: nil response"))
	}
	if isCycleTLSTransportFailure(resp) {
		return nil, false, sanitizedAuthTransportError("subscriptions entitlement")
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, false, nil
	}
	if resp.StatusCode >= 400 {
		return nil, false, fmt.Errorf("subscriptions API error: status %d", resp.StatusCode)
	}

	result, err := parseSubscriptionResult(resp.Body)
	if err != nil {
		return nil, false, err
	}
	if result == nil {
		return nil, false, nil
	}
	if guid, ok := result["guid"].(string); !ok || guid == "" {
		return nil, false, nil
	}
	return result, true, nil
}

func parseSubscriptionResult(body string) (map[string]interface{}, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return nil, fmt.Errorf("failed to parse subscriptions response: %w", err)
	}
	result, ok := payload["result"].(map[string]interface{})
	if !ok {
		return nil, nil
	}
	return result, nil
}

func userFromEntitlement(entitlement map[string]interface{}) map[string]interface{} {
	if entitlement == nil {
		return nil
	}
	user := make(map[string]interface{}, 1)
	if guid, ok := entitlement["guid"].(string); ok && guid != "" {
		user["guid"] = guid
	}
	if len(user) == 0 {
		return nil
	}
	return user
}

func subscriptionTier(entitlement map[string]interface{}) string {
	active := activeSubscription(entitlement)
	if active == nil {
		return "free"
	}
	if name, ok := subscriptionTierNames[tierInt(active["tier"])]; ok {
		return name
	}
	return "premium"
}

func activeSubscription(entitlement map[string]interface{}) map[string]interface{} {
	views, ok := entitlement["subscriptionView"].([]interface{})
	if !ok {
		return nil
	}
	for _, view := range views {
		subscription, ok := view.(map[string]interface{})
		if !ok {
			continue
		}
		if action, ok := subscription["action"].(string); ok && action == "ACTIVE" {
			return subscription
		}
	}
	return nil
}

func tierInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		var tier int
		if _, err := fmt.Sscanf(v, "%d", &tier); err == nil {
			return tier
		}
	}
	return 0
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

	if a.strategy == StrategyBasic {
		if err := a.fetchBasic(); err != nil {
			a.strategy = StrategyCSRF
			if fallbackErr := a.fetchCSRF(); fallbackErr != nil {
				return "", combinedAuthError("basic", err, "csrf", fallbackErr)
			}
		}
	} else {
		if err := a.fetchCSRF(); err != nil {
			a.strategy = StrategyBasic
			if fallbackErr := a.fetchBasic(); fallbackErr != nil {
				return "", combinedAuthError("csrf", err, "basic", fallbackErr)
			}
		}
	}

	return a.crumb, nil
}

// fetchBasic implements the basic authentication strategy.
// 1. GET https://fc.yahoo.com -> captures cookies
// 2. GET https://query1.finance.yahoo.com/v1/test/getcrumb -> gets crumb
func (a *AuthManager) fetchBasic() error {
	// Step 1: Get cookie from fc.yahoo.com
	resp, err := a.client.Get(endpoints.CookieURL, nil)
	if err != nil {
		return fmt.Errorf("failed to get cookie: %w", sanitizedAuthTransportError("basic cookie"))
	}
	if err := validateBasicCookieResponse(resp); err != nil {
		return fmt.Errorf("failed to get cookie: %w", err)
	}

	// Extract cookies from response headers
	a.extractCookies(resp.Headers)

	// Step 2: Get crumb
	resp, err = a.client.Get(endpoints.CrumbURL, nil)
	if err != nil {
		return fmt.Errorf("failed to get crumb: %w", sanitizedAuthTransportError("basic crumb"))
	}

	if err := validateCrumbResponse("basic crumb", resp); err != nil {
		return fmt.Errorf("failed to get crumb: %w", err)
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
		return fmt.Errorf("failed to get consent page: %w", sanitizedAuthTransportError("csrf consent"))
	}
	if err := validateAuthHTTPResponse("csrf consent", resp); err != nil {
		return fmt.Errorf("failed to get consent page: %w", err)
	}

	// Extract CSRF token and session ID from HTML
	csrfToken := extractInputValue(resp.Body, "csrfToken")
	sessionID := extractInputValue(resp.Body, "sessionId")

	if csrfToken == "" || sessionID == "" {
		return WrapInvalidResponseError(fmt.Errorf("csrf consent: missing csrfToken or sessionId"))
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
	resp, err = a.client.Post(collectURL, nil, consentData)
	if err != nil {
		return fmt.Errorf("failed to submit consent: %w", sanitizedAuthTransportError("csrf collect consent"))
	}
	if err := validateAuthHTTPResponse("csrf collect consent", resp); err != nil {
		return fmt.Errorf("failed to submit consent: %w", err)
	}

	// Step 3: Copy consent
	copyURL := fmt.Sprintf("%s?sessionId=%s", endpoints.CopyConsentURL, sessionID)
	resp, err = a.client.Get(copyURL, nil)
	if err != nil {
		return fmt.Errorf("failed to copy consent: %w", sanitizedAuthTransportError("csrf copy consent"))
	}
	if err := validateAuthHTTPResponse("csrf copy consent", resp); err != nil {
		return fmt.Errorf("failed to copy consent: %w", err)
	}

	// Step 4: Get crumb
	resp, err = a.client.Get(endpoints.CrumbCSRFURL, nil)
	if err != nil {
		return fmt.Errorf("failed to get crumb: %w", sanitizedAuthTransportError("csrf crumb"))
	}

	if err := validateCrumbResponse("csrf crumb", resp); err != nil {
		return fmt.Errorf("failed to get crumb: %w", err)
	}

	a.crumb = strings.TrimSpace(resp.Body)
	a.expiry = time.Now().Add(1 * time.Hour)

	return nil
}

func combinedAuthError(firstStrategy string, firstErr error, secondStrategy string, secondErr error) error {
	return WrapAuthError(errors.Join(
		fmt.Errorf("%s strategy: %w", firstStrategy, firstErr),
		fmt.Errorf("%s strategy: %w", secondStrategy, secondErr),
	))
}

func validateCrumbResponse(stage string, resp *Response) error {
	if err := validateAuthHTTPResponse(stage, resp); err != nil {
		return err
	}

	body := strings.TrimSpace(resp.Body)
	if body == "" {
		return WrapInvalidResponseError(fmt.Errorf("%s: empty crumb response", stage))
	}
	if looksLikeHTML(body) {
		return WrapInvalidResponseError(fmt.Errorf("%s: HTML crumb response", stage))
	}
	return nil
}

func looksLikeHTML(body string) bool {
	return htmlCrumbPattern.MatchString(body)
}

func validateBasicCookieResponse(resp *Response) error {
	if resp == nil {
		return WrapInvalidResponseError(fmt.Errorf("basic cookie: nil response"))
	}
	if isCycleTLSTransportFailure(resp) {
		return sanitizedAuthTransportError("basic cookie")
	}
	return nil
}

func validateAuthHTTPResponse(stage string, resp *Response) error {
	if resp == nil {
		return WrapInvalidResponseError(fmt.Errorf("%s: nil response", stage))
	}
	if isCycleTLSTransportFailure(resp) {
		return sanitizedAuthTransportError(stage)
	}
	if resp.StatusCode == 429 || strings.Contains(resp.Body, "Too Many Requests") {
		return fmt.Errorf("%s: %w", stage, WrapRateLimitError())
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s: %w", stage, sanitizedHTTPStatusError(resp.StatusCode))
	}
	return nil
}

func isCycleTLSTransportFailure(resp *Response) bool {
	if resp.StatusCode <= 0 {
		return true
	}
	body := strings.TrimSpace(resp.Body)
	return strings.HasPrefix(body, "Request returned a Syscall Error:")
}

func sanitizedAuthTransportError(stage string) error {
	return fmt.Errorf("%s: %w", stage, WrapNetworkError(errors.New("transport failure")))
}

func sanitizedHTTPStatusError(statusCode int) error {
	switch statusCode {
	case 401, 403:
		return WrapAuthError(fmt.Errorf("HTTP %d", statusCode))
	case 404:
		return NewError(ErrCodeNotFound, fmt.Sprintf("resource not found: HTTP %d", statusCode), nil)
	case 500, 502, 503, 504:
		return NewError(ErrCodeNetwork, fmt.Sprintf("server error: HTTP %d", statusCode), nil)
	default:
		return NewError(ErrCodeUnknown, fmt.Sprintf("HTTP %d", statusCode), nil)
	}
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
