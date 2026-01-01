package telegramreceiver

import (
	"context"
	"net/http"
)

// Receiver defines the interface for receiving Telegram updates.
// This interface allows for easy mocking in tests.
type Receiver interface {
	// Start begins receiving updates from Telegram.
	Start(ctx context.Context) error
	// Stop gracefully stops receiving updates.
	Stop()
	// IsHealthy returns health status for Kubernetes probes.
	IsHealthy() bool
}

// Ensure LongPollingClient implements Receiver at compile time.
var _ Receiver = (*LongPollingClient)(nil)

// HTTPClient is an interface for HTTP client operations.
// This allows for mocking HTTP calls in tests.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Ensure http.Client implements HTTPClient.
var _ HTTPClient = (*http.Client)(nil)

// WebhookProcessor defines the interface for webhook HTTP handlers.
type WebhookProcessor interface {
	http.Handler
}

// Ensure WebhookHandler implements WebhookProcessor.
var _ WebhookProcessor = (*WebhookHandler)(nil)

// UpdateHandler processes incoming Telegram updates.
type UpdateHandler interface {
	HandleUpdate(ctx context.Context, update TelegramUpdate) error
}
