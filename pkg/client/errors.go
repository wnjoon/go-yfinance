package client

import (
	"errors"
	"fmt"
)

// ErrorCode represents the type of error.
type ErrorCode int

const (
	// ErrCodeUnknown is an unknown error.
	ErrCodeUnknown ErrorCode = iota
	// ErrCodeNetwork is a network-related error.
	ErrCodeNetwork
	// ErrCodeAuth is an authentication error.
	ErrCodeAuth
	// ErrCodeRateLimit is a rate limiting error (HTTP 429).
	ErrCodeRateLimit
	// ErrCodeNotFound is a not found error (HTTP 404).
	ErrCodeNotFound
	// ErrCodeInvalidSymbol is an invalid ticker symbol error.
	ErrCodeInvalidSymbol
	// ErrCodeInvalidResponse is an invalid response format error.
	ErrCodeInvalidResponse
	// ErrCodeNoData is a no data available error.
	ErrCodeNoData
	// ErrCodeTimeout is a request timeout error.
	ErrCodeTimeout
)

// YFError represents a Yahoo Finance API error.
type YFError struct {
	Code    ErrorCode
	Message string
	Cause   error
}

// Error implements the error interface.
func (e *YFError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error.
func (e *YFError) Unwrap() error {
	return e.Cause
}

// Is reports whether the error matches the target.
func (e *YFError) Is(target error) bool {
	t, ok := target.(*YFError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// NewError creates a new YFError.
func NewError(code ErrorCode, message string, cause error) *YFError {
	return &YFError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Predefined errors for easy comparison
var (
	ErrNetwork         = &YFError{Code: ErrCodeNetwork, Message: "network error"}
	ErrAuth            = &YFError{Code: ErrCodeAuth, Message: "authentication error"}
	ErrRateLimit       = &YFError{Code: ErrCodeRateLimit, Message: "rate limited"}
	ErrNotFound        = &YFError{Code: ErrCodeNotFound, Message: "not found"}
	ErrInvalidSymbol   = &YFError{Code: ErrCodeInvalidSymbol, Message: "invalid symbol"}
	ErrInvalidResponse = &YFError{Code: ErrCodeInvalidResponse, Message: "invalid response"}
	ErrNoData          = &YFError{Code: ErrCodeNoData, Message: "no data available"}
	ErrTimeout         = &YFError{Code: ErrCodeTimeout, Message: "request timeout"}
)

// WrapNetworkError wraps an error as a network error.
func WrapNetworkError(err error) *YFError {
	return NewError(ErrCodeNetwork, "network error", err)
}

// WrapAuthError wraps an error as an authentication error.
func WrapAuthError(err error) *YFError {
	return NewError(ErrCodeAuth, "authentication failed", err)
}

// WrapRateLimitError creates a rate limit error.
func WrapRateLimitError() *YFError {
	return NewError(ErrCodeRateLimit, "rate limited by Yahoo Finance", nil)
}

// WrapNotFoundError creates a not found error for a symbol.
func WrapNotFoundError(symbol string) *YFError {
	return NewError(ErrCodeNotFound, fmt.Sprintf("symbol not found: %s", symbol), nil)
}

// WrapInvalidSymbolError creates an invalid symbol error.
func WrapInvalidSymbolError(symbol string) *YFError {
	return NewError(ErrCodeInvalidSymbol, fmt.Sprintf("invalid symbol: %s", symbol), nil)
}

// WrapInvalidResponseError wraps an error as an invalid response error.
func WrapInvalidResponseError(err error) *YFError {
	return NewError(ErrCodeInvalidResponse, "invalid response format", err)
}

// WrapNoDataError creates a no data error for a symbol.
func WrapNoDataError(symbol string) *YFError {
	return NewError(ErrCodeNoData, fmt.Sprintf("no data available for: %s", symbol), nil)
}

// WrapTimeoutError wraps an error as a timeout error.
func WrapTimeoutError(err error) *YFError {
	return NewError(ErrCodeTimeout, "request timeout", err)
}

// IsRateLimitError checks if the error is a rate limit error.
func IsRateLimitError(err error) bool {
	return errors.Is(err, ErrRateLimit)
}

// IsAuthError checks if the error is an authentication error.
func IsAuthError(err error) bool {
	return errors.Is(err, ErrAuth)
}

// IsNotFoundError checks if the error is a not found error.
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsInvalidSymbolError checks if the error is an invalid symbol error.
func IsInvalidSymbolError(err error) bool {
	return errors.Is(err, ErrInvalidSymbol)
}

// IsNoDataError checks if the error is a no data error.
func IsNoDataError(err error) bool {
	return errors.Is(err, ErrNoData)
}

// IsTimeoutError checks if the error is a timeout error.
func IsTimeoutError(err error) bool {
	return errors.Is(err, ErrTimeout)
}

// HTTPStatusToError converts an HTTP status code to an appropriate error.
func HTTPStatusToError(statusCode int, body string) *YFError {
	switch statusCode {
	case 401, 403:
		return WrapAuthError(fmt.Errorf("HTTP %d", statusCode))
	case 404:
		return NewError(ErrCodeNotFound, "resource not found", nil)
	case 429:
		return WrapRateLimitError()
	case 500, 502, 503, 504:
		return NewError(ErrCodeNetwork, fmt.Sprintf("server error: HTTP %d", statusCode), nil)
	default:
		if statusCode >= 400 {
			return NewError(ErrCodeUnknown, fmt.Sprintf("HTTP %d: %s", statusCode, body), nil)
		}
		return nil
	}
}
