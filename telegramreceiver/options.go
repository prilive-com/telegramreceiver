package telegramreceiver

import (
	"log/slog"
	"net/http"
	"time"
)

// Option configures a Client. Use With* functions to create options.
// This interface-based approach prevents misuse and enables type safety.
type Option interface {
	apply(*ClientConfig)
}

// optionFunc wraps a function to implement Option interface.
type optionFunc func(*ClientConfig)

func (f optionFunc) apply(c *ClientConfig) { f(c) }

// ClientConfig holds all configuration for a Client.
// Use DefaultClientConfig() to get sensible defaults.
type ClientConfig struct {
	// Required
	BotToken string

	// Receiver mode
	Mode ReceiverMode

	// Webhook settings
	WebhookPort   int
	WebhookSecret string
	TLSCertPath   string
	TLSKeyPath    string
	AllowedDomain string
	WebhookURL    string

	// Long polling settings
	PollingTimeout       int
	PollingLimit         int
	PollingMaxErrors     int
	PollingDeleteWebhook bool
	AllowedUpdates       []string

	// Retry settings (exponential backoff)
	RetryInitialDelay  time.Duration
	RetryMaxDelay      time.Duration
	RetryBackoffFactor float64

	// Rate limiting
	RateLimitRequests float64
	RateLimitBurst    int

	// Request settings
	MaxBodySize       int64
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration

	// Circuit breaker
	BreakerMaxRequests uint32
	BreakerInterval    time.Duration
	BreakerTimeout     time.Duration

	// Kubernetes-aware shutdown
	DrainDelay      time.Duration
	ShutdownTimeout time.Duration

	// Logging
	LogFilePath string
	Logger      *slog.Logger

	// Custom HTTP client (for testing)
	HTTPClient HTTPClient
}

// DefaultClientConfig returns a ClientConfig with sensible defaults.
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Mode:                  ModeWebhook,
		WebhookPort:           8443,
		PollingTimeout:        30,
		PollingLimit:          100,
		PollingMaxErrors:      10,
		RetryInitialDelay:     time.Second,
		RetryMaxDelay:         60 * time.Second,
		RetryBackoffFactor:    2.0,
		RateLimitRequests:     10,
		RateLimitBurst:        20,
		MaxBodySize:           1048576,
		ReadTimeout:           10 * time.Second,
		ReadHeaderTimeout:     2 * time.Second,
		WriteTimeout:          15 * time.Second,
		IdleTimeout:           120 * time.Second,
		BreakerMaxRequests:    5,
		BreakerInterval:       2 * time.Minute,
		BreakerTimeout:        60 * time.Second,
		DrainDelay:            5 * time.Second,
		ShutdownTimeout:       15 * time.Second,
		LogFilePath:           "logs/telegramreceiver.log",
		Logger:                slog.Default(),
	}
}

// WithMode sets the receiver mode (webhook or longpolling).
func WithMode(mode ReceiverMode) Option {
	return optionFunc(func(c *ClientConfig) { c.Mode = mode })
}

// WithWebhook configures webhook mode settings.
func WithWebhook(port int, secret string) Option {
	return optionFunc(func(c *ClientConfig) {
		c.Mode = ModeWebhook
		c.WebhookPort = port
		c.WebhookSecret = secret
	})
}

// WithWebhookTLS sets TLS certificate paths for webhook mode.
func WithWebhookTLS(certPath, keyPath string) Option {
	return optionFunc(func(c *ClientConfig) {
		c.TLSCertPath = certPath
		c.TLSKeyPath = keyPath
	})
}

// WithWebhookURL sets the public URL for webhook auto-registration.
func WithWebhookURL(url string) Option {
	return optionFunc(func(c *ClientConfig) { c.WebhookURL = url })
}

// WithAllowedDomain restricts webhook requests to a specific domain.
func WithAllowedDomain(domain string) Option {
	return optionFunc(func(c *ClientConfig) { c.AllowedDomain = domain })
}

// WithPolling configures long polling mode settings.
func WithPolling(timeout, limit int) Option {
	return optionFunc(func(c *ClientConfig) {
		c.Mode = ModeLongPolling
		c.PollingTimeout = timeout
		c.PollingLimit = limit
	})
}

// WithPollingMaxErrors sets maximum consecutive errors before stopping.
// Set to 0 for unlimited retries.
func WithPollingMaxErrors(max int) Option {
	return optionFunc(func(c *ClientConfig) { c.PollingMaxErrors = max })
}

// WithPollingDeleteWebhook enables automatic webhook deletion before polling starts.
func WithPollingDeleteWebhook(delete bool) Option {
	return optionFunc(func(c *ClientConfig) { c.PollingDeleteWebhook = delete })
}

// WithAllowedUpdateTypes filters which update types to receive.
func WithAllowedUpdateTypes(types []string) Option {
	return optionFunc(func(c *ClientConfig) { c.AllowedUpdates = types })
}

// WithRetry configures exponential backoff retry settings.
func WithRetry(initialDelay, maxDelay time.Duration, backoffFactor float64) Option {
	return optionFunc(func(c *ClientConfig) {
		c.RetryInitialDelay = initialDelay
		c.RetryMaxDelay = maxDelay
		c.RetryBackoffFactor = backoffFactor
	})
}

// WithRateLimit sets rate limiting parameters.
func WithRateLimit(requestsPerSecond float64, burst int) Option {
	return optionFunc(func(c *ClientConfig) {
		c.RateLimitRequests = requestsPerSecond
		c.RateLimitBurst = burst
	})
}

// WithBreakerConfig configures the circuit breaker.
func WithBreakerConfig(maxRequests uint32, interval, timeout time.Duration) Option {
	return optionFunc(func(c *ClientConfig) {
		c.BreakerMaxRequests = maxRequests
		c.BreakerInterval = interval
		c.BreakerTimeout = timeout
	})
}

// WithTimeouts sets HTTP server timeouts.
func WithTimeouts(read, readHeader, write, idle time.Duration) Option {
	return optionFunc(func(c *ClientConfig) {
		c.ReadTimeout = read
		c.ReadHeaderTimeout = readHeader
		c.WriteTimeout = write
		c.IdleTimeout = idle
	})
}

// WithMaxBodySize sets the maximum request body size.
func WithMaxBodySize(size int64) Option {
	return optionFunc(func(c *ClientConfig) { c.MaxBodySize = size })
}

// WithShutdown configures Kubernetes-aware graceful shutdown.
func WithShutdown(drainDelay, timeout time.Duration) Option {
	return optionFunc(func(c *ClientConfig) {
		c.DrainDelay = drainDelay
		c.ShutdownTimeout = timeout
	})
}

// WithLogger sets a custom slog.Logger.
func WithLogger(logger *slog.Logger) Option {
	return optionFunc(func(c *ClientConfig) { c.Logger = logger })
}

// WithLogFile sets the log file path.
func WithLogFile(path string) Option {
	return optionFunc(func(c *ClientConfig) { c.LogFilePath = path })
}

// WithHTTPClient sets a custom HTTP client (useful for testing).
func WithHTTPClientOption(client HTTPClient) Option {
	return optionFunc(func(c *ClientConfig) { c.HTTPClient = client })
}

// Presets for common configurations

// ProductionPreset returns options suitable for production environments.
func ProductionPreset() Option {
	return optionFunc(func(c *ClientConfig) {
		c.PollingMaxErrors = 10
		c.RetryInitialDelay = 2 * time.Second
		c.RetryMaxDelay = 60 * time.Second
		c.BreakerMaxRequests = 5
		c.DrainDelay = 10 * time.Second
		c.ShutdownTimeout = 30 * time.Second
	})
}

// DevelopmentPreset returns options suitable for development.
func DevelopmentPreset() Option {
	return optionFunc(func(c *ClientConfig) {
		c.PollingMaxErrors = 3
		c.RetryInitialDelay = 500 * time.Millisecond
		c.RetryMaxDelay = 5 * time.Second
		c.BreakerMaxRequests = 2
		c.DrainDelay = time.Second
		c.ShutdownTimeout = 5 * time.Second
	})
}

// Compile-time check that http.Client implements HTTPClient
var _ HTTPClient = (*http.Client)(nil)
