package telegramreceiver

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/big"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sony/gobreaker/v2"
)

// LongPollingClient polls Telegram's getUpdates API for new updates.
// If POLLING_DELETE_WEBHOOK is set to true, it calls deleteWebhook before starting
// to ensure the bot is not in webhook mode.
type LongPollingClient struct {
	botToken SecretToken
	updates  chan<- TelegramUpdate
	logger   *slog.Logger

	// Polling configuration
	timeout              int
	limit                int
	maxErrors            int      // Max consecutive errors before stopping (0 = unlimited)
	allowedUpdates       []string // Optional: filter update types
	deleteWebhookOnStart bool     // Delete existing webhook before starting

	// Retry configuration with exponential backoff
	retryInitialDelay  time.Duration // Initial delay before first retry
	retryMaxDelay      time.Duration // Maximum delay cap
	retryBackoffFactor float64       // Multiplier for each retry (e.g., 2.0 for doubling)

	// HTTP client
	client httpClient

	// Circuit breaker for resilience
	breaker *gobreaker.CircuitBreaker[[]byte]

	// State management
	running           atomic.Bool
	offset            int
	consecutiveErrors atomic.Int32 // Exposed for health checks
	stopCh            chan struct{}
	closeOnce         sync.Once // Prevents double-close panic
	wg                sync.WaitGroup
}

const defaultMaxConsecutiveErrors = 10

// Default retry configuration for exponential backoff
const (
	defaultRetryInitialDelay  = 1 * time.Second
	defaultRetryMaxDelay      = 60 * time.Second
	defaultRetryBackoffFactor = 2.0
)

// LongPollingOption configures the LongPollingClient.
type LongPollingOption func(*LongPollingClient)

// WithHTTPClient sets a custom HTTP client for the polling client.
func WithHTTPClient(client *http.Client) LongPollingOption {
	return func(c *LongPollingClient) {
		c.client = client
	}
}

// WithCircuitBreaker sets a custom circuit breaker for the polling client.
func WithCircuitBreaker(breaker *gobreaker.CircuitBreaker[[]byte]) LongPollingOption {
	return func(c *LongPollingClient) {
		c.breaker = breaker
	}
}

// WithMaxErrors sets the maximum consecutive errors before stopping.
// Set to 0 for unlimited retries.
func WithMaxErrors(max int) LongPollingOption {
	return func(c *LongPollingClient) {
		c.maxErrors = max
	}
}

// WithAllowedUpdates sets the update types to receive.
// See https://core.telegram.org/bots/api#update
func WithAllowedUpdates(types []string) LongPollingOption {
	return func(c *LongPollingClient) {
		c.allowedUpdates = types
	}
}

// WithDeleteWebhook configures the client to delete any existing webhook before starting.
// Default is false - the client assumes no webhook exists or the user manages webhooks manually.
func WithDeleteWebhook(delete bool) LongPollingOption {
	return func(c *LongPollingClient) {
		c.deleteWebhookOnStart = delete
	}
}

// WithRetryConfig sets exponential backoff parameters for retry logic.
// initialDelay: delay before first retry (default: 1s)
// maxDelay: maximum delay cap (default: 60s)
// backoffFactor: multiplier for each retry (default: 2.0)
func WithRetryConfig(initialDelay, maxDelay time.Duration, backoffFactor float64) LongPollingOption {
	return func(c *LongPollingClient) {
		if initialDelay > 0 {
			c.retryInitialDelay = initialDelay
		}
		if maxDelay > 0 {
			c.retryMaxDelay = maxDelay
		}
		if backoffFactor > 1.0 {
			c.retryBackoffFactor = backoffFactor
		}
	}
}

// NewLongPollingClient creates a new long polling client.
// The updates channel must be provided (dependency injection pattern).
//
// Deprecated: Use New() or NewFromConfig() instead for a simpler API.
// This function will be removed in v4.
//
//	client, err := telegramreceiver.New(token,
//	    telegramreceiver.WithMode(telegramreceiver.ModeLongPolling),
//	    telegramreceiver.WithPolling(30, 100),
//	)
func NewLongPollingClient(
	botToken SecretToken,
	updates chan<- TelegramUpdate,
	logger *slog.Logger,
	timeout int,
	limit int,
	breakerMaxRequests uint32,
	breakerInterval time.Duration,
	breakerTimeout time.Duration,
	opts ...LongPollingOption,
) *LongPollingClient {
	client := &LongPollingClient{
		botToken:           botToken,
		updates:            updates,
		logger:             logger,
		timeout:            timeout,
		limit:              limit,
		maxErrors:          defaultMaxConsecutiveErrors,
		retryInitialDelay:  defaultRetryInitialDelay,
		retryMaxDelay:      defaultRetryMaxDelay,
		retryBackoffFactor: defaultRetryBackoffFactor,
		client:             defaultPollingHTTPClient(timeout),
		stopCh:             make(chan struct{}),
	}

	// Create default circuit breaker
	client.breaker = gobreaker.NewCircuitBreaker[[]byte](gobreaker.Settings{
		Name:        "telegram-polling",
		MaxRequests: breakerMaxRequests,
		Interval:    breakerInterval,
		Timeout:     breakerTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Info("circuit breaker state changed",
				"name", name,
				"from", from.String(),
				"to", to.String(),
			)
		},
	})

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	return client
}

// defaultPollingHTTPClient creates an HTTP client optimized for long polling.
func defaultPollingHTTPClient(timeoutSeconds int) *http.Client {
	// Add extra time for network overhead beyond the Telegram timeout
	httpTimeout := time.Duration(timeoutSeconds+10) * time.Second

	return &http.Client{
		Timeout: httpTimeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			MaxIdleConns:          10,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			ResponseHeaderTimeout: time.Duration(timeoutSeconds+5) * time.Second,
			ForceAttemptHTTP2:     true,
		},
	}
}

// calculateBackoff computes the next retry delay using exponential backoff with cryptographic jitter.
// Uses crypto/rand for jitter to avoid thundering herd in distributed systems.
// Formula: min(maxDelay, initialDelay * (backoffFactor ^ attempt)) + random_jitter
func (c *LongPollingClient) calculateBackoff(attempt int32) time.Duration {
	// Calculate base delay with exponential backoff
	baseDelay := float64(c.retryInitialDelay) * math.Pow(c.retryBackoffFactor, float64(attempt-1))

	// Cap at maxDelay
	if baseDelay > float64(c.retryMaxDelay) {
		baseDelay = float64(c.retryMaxDelay)
	}

	// Add cryptographic jitter (0-25% of base delay)
	jitterRange := int64(baseDelay * 0.25)
	if jitterRange > 0 {
		jitterBig, err := rand.Int(rand.Reader, big.NewInt(jitterRange))
		if err == nil {
			baseDelay += float64(jitterBig.Int64())
		}
		// If crypto/rand fails, proceed without jitter (fail-safe)
	}

	return time.Duration(baseDelay)
}

// Start begins polling for updates from Telegram.
// If deleteWebhookOnStart is enabled, it deletes any existing webhook before starting.
// Returns ErrPollingAlreadyRunning if the client is already running.
func (c *LongPollingClient) Start(ctx context.Context) error {
	if !c.running.CompareAndSwap(false, true) {
		return ErrPollingAlreadyRunning
	}

	// Only delete webhook if explicitly configured
	if c.deleteWebhookOnStart {
		c.logger.Info("deleting existing webhook before starting long polling")
		if err := DeleteWebhookWithClient(ctx, c.client, c.botToken, false); err != nil {
			c.running.Store(false)
			return fmt.Errorf("failed to delete webhook: %w", err)
		}
	}

	c.wg.Add(1)
	go c.pollLoop(ctx)

	c.logger.Info("long polling started",
		"timeout", c.timeout,
		"limit", c.limit,
		"max_errors", c.maxErrors,
	)

	return nil
}

// Stop gracefully stops the polling client.
// It blocks until the polling goroutine has finished.
// Safe to call multiple times.
func (c *LongPollingClient) Stop() {
	if !c.running.CompareAndSwap(true, false) {
		return
	}

	// Use sync.Once to prevent double-close panic
	c.closeOnce.Do(func() {
		close(c.stopCh)
	})
	c.wg.Wait()
	c.logger.Info("long polling stopped")
}

// pollLoop is the main polling loop.
func (c *LongPollingClient) pollLoop(ctx context.Context) {
	defer c.wg.Done()
	defer c.running.Store(false)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("polling stopped due to context cancellation")
			return
		case <-c.stopCh:
			c.logger.Info("polling stopped due to stop signal")
			return
		default:
		}

		updates, err := c.fetchUpdates(ctx)
		if err != nil {
			errCount := c.consecutiveErrors.Add(1)
			backoff := c.calculateBackoff(errCount)
			c.logger.Error("failed to fetch updates",
				"error", err,
				"consecutive_errors", errCount,
				"retry_delay", backoff,
			)

			// Check max errors (0 = unlimited)
			if c.maxErrors > 0 && int(errCount) >= c.maxErrors {
				c.logger.Error("max consecutive errors exceeded, stopping polling",
					"max_errors", c.maxErrors,
				)
				return
			}

			select {
			case <-ctx.Done():
				return
			case <-c.stopCh:
				return
			case <-time.After(backoff):
				continue
			}
		}

		c.consecutiveErrors.Store(0)

		for _, update := range updates {
			// Update offset to acknowledge this update
			if update.UpdateID >= c.offset {
				c.offset = update.UpdateID + 1
			}

			select {
			case c.updates <- update:
				c.logger.Debug("update sent to channel",
					"update_id", update.UpdateID,
				)
			default:
				c.logger.Warn("updates channel full, dropping update",
					"update_id", update.UpdateID,
				)
			}
		}
	}
}

// getUpdatesResponse is the response from Telegram's getUpdates API.
type getUpdatesResponse struct {
	OK          bool             `json:"ok"`
	Result      []TelegramUpdate `json:"result,omitempty"`
	ErrorCode   int              `json:"error_code,omitempty"`
	Description string           `json:"description,omitempty"`
}

// fetchUpdates calls the Telegram getUpdates API.
func (c *LongPollingClient) fetchUpdates(ctx context.Context) ([]TelegramUpdate, error) {
	url := fmt.Sprintf("%s%s/getUpdates?timeout=%d&limit=%d&offset=%d",
		telegramAPIBaseURL,
		c.botToken.Value(),
		c.timeout,
		c.limit,
		c.offset,
	)

	// Add allowed_updates if configured
	if len(c.allowedUpdates) > 0 {
		encoded, err := json.Marshal(c.allowedUpdates)
		if err == nil {
			url += "&allowed_updates=" + string(encoded)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, &TelegramAPIError{Description: "failed to create request", Err: err}
	}

	// Use circuit breaker for the HTTP call
	respBody, err := c.breaker.Execute(func() ([]byte, error) {
		resp, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}
		defer func() {
			// Always drain remaining body for connection reuse
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		// Check HTTP status code
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		return body, nil
	})

	if err != nil {
		return nil, &TelegramAPIError{Description: "request failed", Err: err}
	}

	var response getUpdatesResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, &TelegramAPIError{Description: "failed to parse response", Err: err}
	}

	if !response.OK {
		return nil, &TelegramAPIError{
			Code:        response.ErrorCode,
			Description: response.Description,
		}
	}

	return response.Result, nil
}

// Running returns true if the polling client is currently running.
func (c *LongPollingClient) Running() bool {
	return c.running.Load()
}

// IsHealthy returns health status for K8s probes.
// Returns false if not running or too many consecutive errors.
func (c *LongPollingClient) IsHealthy() bool {
	if c.maxErrors == 0 {
		// Unlimited errors mode - just check if running
		return c.running.Load()
	}
	return c.running.Load() && int(c.consecutiveErrors.Load()) < c.maxErrors
}

// ConsecutiveErrors returns the current consecutive error count.
func (c *LongPollingClient) ConsecutiveErrors() int32 {
	return c.consecutiveErrors.Load()
}

// Offset returns the current update offset.
func (c *LongPollingClient) Offset() int {
	return c.offset
}
