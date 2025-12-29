package telegramreceiver

import (
	"errors"
	"testing"
)

func TestWebhookError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *WebhookError
		expected string
	}{
		{
			name:     "simple error",
			err:      &WebhookError{Code: 401, Message: "unauthorized"},
			expected: "unauthorized",
		},
		{
			name:     "error with wrapped error",
			err:      &WebhookError{Code: 500, Message: "failed", Err: errors.New("connection refused")},
			expected: "failed: connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestWebhookError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	err := &WebhookError{Code: 500, Message: "outer", Err: inner}

	if unwrapped := err.Unwrap(); unwrapped != inner {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, inner)
	}
}

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  *WebhookError
		code int
	}{
		{"ErrForbidden", ErrForbidden, 403},
		{"ErrUnauthorized", ErrUnauthorized, 401},
		{"ErrMethodNotAllowed", ErrMethodNotAllowed, 405},
		{"ErrChannelBlocked", ErrChannelBlocked, 503},
		{"ErrBodyReadFailed", ErrBodyReadFailed, 500},
		{"ErrInvalidJSON", ErrInvalidJSON, 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.code {
				t.Errorf("%s.Code = %d, want %d", tt.name, tt.err.Code, tt.code)
			}
		})
	}
}
