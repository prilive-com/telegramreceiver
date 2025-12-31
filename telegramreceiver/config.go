package telegramreceiver

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// ReceiverMode defines how the bot receives updates from Telegram.
type ReceiverMode string

const (
	// ModeWebhook receives updates via HTTPS webhook (Telegram pushes to your server).
	ModeWebhook ReceiverMode = "webhook"
	// ModeLongPolling receives updates by polling Telegram API (your server pulls).
	ModeLongPolling ReceiverMode = "longpolling"
)

type Config struct {
	// Receiver mode selection
	ReceiverMode ReceiverMode

	// Bot token (required for long polling, optional for webhook auto-registration)
	BotToken SecretToken

	// Webhook configuration
	WebhookPort   int
	TLSCertPath   string
	TLSKeyPath    string
	WebhookSecret string
	AllowedDomain string
	WebhookURL    string // Public URL for auto-registration (optional)

	// Long polling configuration
	PollingTimeout       int           // Seconds to wait for updates (0-60)
	PollingLimit         int           // Max updates per request (1-100)
	PollingRetryDelay    time.Duration // Delay between retries on error
	PollingMaxErrors     int           // Max consecutive errors before stopping (0 = unlimited)
	PollingDeleteWebhook bool          // Delete existing webhook before starting (default: false)
	AllowedUpdates       []string      // Filter update types (empty = all)

	// Common configuration
	LogFilePath        string
	RateLimitRequests  float64
	RateLimitBurst     int
	MaxBodySize        int64
	ReadTimeout        time.Duration
	ReadHeaderTimeout  time.Duration
	WriteTimeout       time.Duration
	IdleTimeout        time.Duration
	BreakerMaxRequests uint32
	BreakerInterval    time.Duration
	BreakerTimeout     time.Duration

	// Kubernetes-aware shutdown settings
	DrainDelay      time.Duration // Time to wait for LB to stop routing before shutdown
	ShutdownTimeout time.Duration // Max time for graceful shutdown
}

func LoadConfig() (*Config, error) {
	// Parse receiver mode
	receiverModeStr := getEnv("RECEIVER_MODE", "webhook")
	var receiverMode ReceiverMode
	switch strings.ToLower(receiverModeStr) {
	case "webhook":
		receiverMode = ModeWebhook
	case "longpolling":
		receiverMode = ModeLongPolling
	default:
		return nil, ErrInvalidReceiverMode
	}

	// Parse webhook port
	webhookPort, err := strconv.Atoi(getEnv("WEBHOOK_PORT", "8443"))
	if err != nil {
		return nil, err
	}

	// Parse polling timeout (0-60 seconds)
	pollingTimeout, err := strconv.Atoi(getEnv("POLLING_TIMEOUT", "30"))
	if err != nil {
		return nil, err
	}
	if pollingTimeout < 0 || pollingTimeout > 60 {
		return nil, ErrInvalidPollingTimeout
	}

	// Parse polling limit (1-100)
	pollingLimit, err := strconv.Atoi(getEnv("POLLING_LIMIT", "100"))
	if err != nil {
		return nil, err
	}
	if pollingLimit < 1 || pollingLimit > 100 {
		return nil, ErrInvalidPollingLimit
	}

	// Parse polling retry delay
	pollingRetryDelay, err := time.ParseDuration(getEnv("POLLING_RETRY_DELAY", "5s"))
	if err != nil {
		return nil, err
	}

	// Parse polling max errors (0 = unlimited)
	pollingMaxErrors, err := strconv.Atoi(getEnv("POLLING_MAX_ERRORS", "10"))
	if err != nil {
		return nil, err
	}

	// Parse polling delete webhook (default: false)
	pollingDeleteWebhook := strings.ToLower(getEnv("POLLING_DELETE_WEBHOOK", "false")) == "true"

	// Parse allowed updates (comma-separated list)
	var allowedUpdates []string
	allowedUpdatesStr := getEnv("ALLOWED_UPDATES", "")
	if allowedUpdatesStr != "" {
		for _, u := range strings.Split(allowedUpdatesStr, ",") {
			trimmed := strings.TrimSpace(u)
			if trimmed != "" {
				allowedUpdates = append(allowedUpdates, trimmed)
			}
		}
	}

	// Parse webhook URL (validate if provided)
	webhookURL := getEnv("WEBHOOK_URL", "")
	if webhookURL != "" && !strings.HasPrefix(webhookURL, "https://") {
		return nil, ErrInvalidWebhookURL
	}

	rateLimitRequests, err := strconv.ParseFloat(getEnv("RATE_LIMIT_REQUESTS", "10"), 64)
	if err != nil {
		return nil, err
	}

	rateLimitBurst, err := strconv.Atoi(getEnv("RATE_LIMIT_BURST", "20"))
	if err != nil {
		return nil, err
	}

	maxBodySize, err := strconv.ParseInt(getEnv("MAX_BODY_SIZE", "1048576"), 10, 64)
	if err != nil {
		return nil, err
	}

	readTimeout, err := time.ParseDuration(getEnv("READ_TIMEOUT", "10s"))
	if err != nil {
		return nil, err
	}

	readHeaderTimeout, err := time.ParseDuration(getEnv("READ_HEADER_TIMEOUT", "2s"))
	if err != nil {
		return nil, err
	}

	writeTimeout, err := time.ParseDuration(getEnv("WRITE_TIMEOUT", "15s"))
	if err != nil {
		return nil, err
	}

	idleTimeout, err := time.ParseDuration(getEnv("IDLE_TIMEOUT", "120s"))
	if err != nil {
		return nil, err
	}

	breakerMaxRequests, err := strconv.ParseUint(getEnv("BREAKER_MAX_REQUESTS", "5"), 10, 32)
	if err != nil {
		return nil, err
	}

	breakerInterval, err := time.ParseDuration(getEnv("BREAKER_INTERVAL", "2m"))
	if err != nil {
		return nil, err
	}

	breakerTimeout, err := time.ParseDuration(getEnv("BREAKER_TIMEOUT", "60s"))
	if err != nil {
		return nil, err
	}

	drainDelay, err := time.ParseDuration(getEnv("DRAIN_DELAY", "5s"))
	if err != nil {
		return nil, err
	}

	shutdownTimeout, err := time.ParseDuration(getEnv("SHUTDOWN_TIMEOUT", "15s"))
	if err != nil {
		return nil, err
	}

	return &Config{
		ReceiverMode:       receiverMode,
		BotToken:           SecretToken(getEnv("TELEGRAM_BOT_TOKEN", "")),
		WebhookPort:        webhookPort,
		TLSCertPath:        getEnv("TLS_CERT_PATH", ""),
		TLSKeyPath:         getEnv("TLS_KEY_PATH", ""),
		WebhookSecret:      getEnv("WEBHOOK_SECRET", ""),
		AllowedDomain:      getEnv("ALLOWED_DOMAIN", ""),
		WebhookURL:         webhookURL,
		PollingTimeout:       pollingTimeout,
		PollingLimit:         pollingLimit,
		PollingRetryDelay:    pollingRetryDelay,
		PollingMaxErrors:     pollingMaxErrors,
		PollingDeleteWebhook: pollingDeleteWebhook,
		AllowedUpdates:       allowedUpdates,
		LogFilePath:        getEnv("LOG_FILE_PATH", "logs/telegramreceiver.log"),
		RateLimitRequests:  rateLimitRequests,
		RateLimitBurst:     rateLimitBurst,
		MaxBodySize:        maxBodySize,
		ReadTimeout:        readTimeout,
		ReadHeaderTimeout:  readHeaderTimeout,
		WriteTimeout:       writeTimeout,
		IdleTimeout:        idleTimeout,
		BreakerMaxRequests: uint32(breakerMaxRequests),
		BreakerInterval:    breakerInterval,
		BreakerTimeout:     breakerTimeout,
		DrainDelay:         drainDelay,
		ShutdownTimeout:    shutdownTimeout,
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
