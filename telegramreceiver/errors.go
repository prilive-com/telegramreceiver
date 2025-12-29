package telegramreceiver

import "fmt"

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

// Sentinel errors for webhook handler.
var (
	ErrForbidden        = &WebhookError{Code: 403, Message: "forbidden"}
	ErrUnauthorized     = &WebhookError{Code: 401, Message: "unauthorized"}
	ErrMethodNotAllowed = &WebhookError{Code: 405, Message: "method not allowed"}
	ErrChannelBlocked   = &WebhookError{Code: 503, Message: "updates channel blocked"}
	ErrBodyReadFailed   = &WebhookError{Code: 500, Message: "failed to read request body"}
	ErrInvalidJSON      = &WebhookError{Code: 400, Message: "invalid JSON payload"}
)
