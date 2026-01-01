package telegramreceiver

import (
	"errors"
	"fmt"
	"time"
)

// WebhookError represents an error with an associated HTTP status code.
type WebhookError struct {
	Code    int
	Message string
	Err     error
}

func (e *WebhookError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *WebhookError) Unwrap() error {
	return e.Err
}

// Is implements errors.Is for WebhookError.
// Two WebhookErrors are equal if they have the same Code.
func (e *WebhookError) Is(target error) bool {
	t, ok := target.(*WebhookError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// Sentinel errors for webhook handler.
var (
	ErrForbidden        = &WebhookError{Code: 403, Message: "forbidden"}
	ErrUnauthorized     = &WebhookError{Code: 401, Message: "unauthorized"}
	ErrMethodNotAllowed = &WebhookError{Code: 405, Message: "method not allowed"}
	ErrChannelBlocked   = &WebhookError{Code: 503, Message: "updates channel blocked"}
	ErrBodyReadFailed   = &WebhookError{Code: 500, Message: "failed to read request body"}
	ErrInvalidJSON      = &WebhookError{Code: 400, Message: "invalid JSON payload"}
)

// Sentinel errors for configuration.
var (
	ErrBotTokenRequired      = errors.New("TELEGRAM_BOT_TOKEN is required for long polling mode")
	ErrInvalidReceiverMode   = errors.New("RECEIVER_MODE must be 'webhook' or 'longpolling'")
	ErrInvalidPollingTimeout = errors.New("POLLING_TIMEOUT must be between 0 and 60")
	ErrInvalidPollingLimit   = errors.New("POLLING_LIMIT must be between 1 and 100")
	ErrInvalidWebhookURL     = errors.New("WEBHOOK_URL must be a valid HTTPS URL")
)

// Sentinel errors for long polling runtime.
var (
	ErrPollingAlreadyRunning = errors.New("long polling client is already running")
	ErrMaxRetriesExceeded    = errors.New("max consecutive retries exceeded")
	ErrUpdatesChannelFull    = errors.New("updates channel is full, dropping update")
)

// TelegramAPIError represents an error response from the Telegram Bot API.
type TelegramAPIError struct {
	Code        int
	Description string
	RetryAfter  time.Duration
	Err         error
}

func (e *TelegramAPIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("telegram API error [%d]: %s: %v", e.Code, e.Description, e.Err)
	}
	if e.RetryAfter > 0 {
		return fmt.Sprintf("telegram API error [%d]: %s (retry after %v)", e.Code, e.Description, e.RetryAfter)
	}
	if e.Code != 0 {
		return fmt.Sprintf("telegram API error [%d]: %s", e.Code, e.Description)
	}
	return fmt.Sprintf("telegram API error: %s", e.Description)
}

func (e *TelegramAPIError) Unwrap() error {
	return e.Err
}

// Is implements errors.Is for TelegramAPIError.
// Two TelegramAPIErrors are equal if they have the same Code.
func (e *TelegramAPIError) Is(target error) bool {
	t, ok := target.(*TelegramAPIError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// IsRetryable returns true if the error indicates a temporary condition
// that may succeed on retry.
func (e *TelegramAPIError) IsRetryable() bool {
	// 429 - Too Many Requests
	// 500, 502, 503, 504 - Server errors
	return e.Code == 429 || (e.Code >= 500 && e.Code <= 504)
}

// NewTelegramAPIError creates a new TelegramAPIError.
func NewTelegramAPIError(code int, description string) *TelegramAPIError {
	return &TelegramAPIError{Code: code, Description: description}
}

// NewTelegramAPIErrorWithRetry creates a new TelegramAPIError with retry information.
func NewTelegramAPIErrorWithRetry(code int, description string, retryAfter time.Duration) *TelegramAPIError {
	return &TelegramAPIError{Code: code, Description: description, RetryAfter: retryAfter}
}
