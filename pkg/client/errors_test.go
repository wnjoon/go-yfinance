package client

import (
	"errors"
	"fmt"
	"testing"
)

func TestYFError(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	err := NewError(ErrCodeNetwork, "network failed", cause)

	if err.Code != ErrCodeNetwork {
		t.Errorf("Code should be ErrCodeNetwork")
	}
	if err.Message != "network failed" {
		t.Errorf("Message should be 'network failed'")
	}
	if err.Cause != cause {
		t.Errorf("Cause should be set")
	}

	expected := "network failed: underlying error"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestYFErrorWithoutCause(t *testing.T) {
	err := NewError(ErrCodeRateLimit, "rate limited", nil)

	if err.Error() != "rate limited" {
		t.Errorf("Error() should not include cause when nil")
	}
}

func TestYFErrorUnwrap(t *testing.T) {
	cause := fmt.Errorf("root cause")
	err := NewError(ErrCodeAuth, "auth failed", cause)

	unwrapped := errors.Unwrap(err)
	if unwrapped != cause {
		t.Errorf("Unwrap should return the cause")
	}
}

func TestYFErrorIs(t *testing.T) {
	err := WrapRateLimitError()

	if !errors.Is(err, ErrRateLimit) {
		t.Error("Should match ErrRateLimit")
	}
	if errors.Is(err, ErrAuth) {
		t.Error("Should not match ErrAuth")
	}
}

func TestErrorHelpers(t *testing.T) {
	tests := []struct {
		name     string
		err      *YFError
		checkFn  func(error) bool
		expected bool
	}{
		{"IsRateLimitError true", WrapRateLimitError(), IsRateLimitError, true},
		{"IsRateLimitError false", WrapAuthError(nil), IsRateLimitError, false},
		{"IsAuthError true", WrapAuthError(nil), IsAuthError, true},
		{"IsAuthError false", WrapRateLimitError(), IsAuthError, false},
		{"IsNotFoundError true", WrapNotFoundError("AAPL"), IsNotFoundError, true},
		{"IsInvalidSymbolError true", WrapInvalidSymbolError("???"), IsInvalidSymbolError, true},
		{"IsNoDataError true", WrapNoDataError("AAPL"), IsNoDataError, true},
		{"IsTimeoutError true", WrapTimeoutError(nil), IsTimeoutError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.checkFn(tt.err)
			if result != tt.expected {
				t.Errorf("%s: got %v, want %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestHTTPStatusToError(t *testing.T) {
	tests := []struct {
		status   int
		expected ErrorCode
	}{
		{401, ErrCodeAuth},
		{403, ErrCodeAuth},
		{404, ErrCodeNotFound},
		{429, ErrCodeRateLimit},
		{500, ErrCodeNetwork},
		{502, ErrCodeNetwork},
		{503, ErrCodeNetwork},
		{504, ErrCodeNetwork},
		{400, ErrCodeUnknown},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("HTTP_%d", tt.status), func(t *testing.T) {
			err := HTTPStatusToError(tt.status, "test body")
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			if err.Code != tt.expected {
				t.Errorf("HTTP %d: got code %v, want %v", tt.status, err.Code, tt.expected)
			}
		})
	}
}

func TestHTTPStatusToErrorSuccess(t *testing.T) {
	err := HTTPStatusToError(200, "")
	if err != nil {
		t.Errorf("HTTP 200 should not return error")
	}
}
