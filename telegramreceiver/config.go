package telegramreceiver

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Supported run modes
const (
	// Use long polling: periodically request updates from Telegram
	ModePolling = "polling"
	// Use webhook: Telegram pushes updates to an HTTP endpoint
	ModeWebhook = "webhook"
)

const (
	defaultMode           = ModePolling
	defaultPort           = 8443
	defaultTimeoutSeconds = 30
	defaultRetryMax       = 5
	defaultLogLevel       = "info"
	defaultLogFilePath    = "telegram_receiver.log"
)

// Config holds configuration settings for the Telegram receiver.
type Config struct {
	BotToken      string // BotToken is the Telegram API token for the bot (required for authentication).
	Mode          string // Mode of receiving updates, either "polling" or "webhook".
	WebhookURL    string // WebhookURL is the public URL for receiving updates via webhook (used in webhook mode).
	WebhookPort   int    // WebhookPort is the local port number for the webhook server (used in webhook mode).
	WebhookSecret string // WebhookSecret is an optional secret token to secure webhook requests (used in webhook mode).
	LogLevel      string // LogLevel sets the verbosity of logging (e.g., "debug", "info", "warn").
	LogFilePath   string // LogFilePath is the file path for logging output (if empty, logs output to the console).
	PollTimeout   int    // PollTimeout is the long polling timeout in seconds (used in polling mode).
	RetryMax      int    // RetryMax is the maximum number of retry attempts for failed operations.
}

// LoadConfig reads configuration from environment variables,
// applies default values for missing settings, and validates the config.
func LoadConfig() (*Config, error) {
	// Create a new Config instance to fill in.
	cfg := &Config{}

	// Get the bot token from environment variable.
	// The bot token is required to authenticate with the Telegram API.
	cfg.BotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	if cfg.BotToken == "" {
		// If the token is not provided, we cannot proceed, so return an error.
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is required and cannot be empty")
	}
	// Optionally, we could validate the token format (Telegram bot tokens have a specific pattern),
	// but for now we just ensure it is present.

	// Get the mode from environment (how the bot receives updates).
	modeEnv := os.Getenv("TELEGRAM_MODE")
	if modeEnv == "" {
		// If no mode is specified, default to "polling" since it's simpler to start with.
		modeEnv = "polling"
	}
	// Normalize the mode to lower-case to allow case-insensitive input (e.g., "Polling" or "WEBHOOK").
	cfg.Mode = strings.ToLower(modeEnv)
	// Validate the mode: it should be either "polling" or "webhook".
	if cfg.Mode != "polling" && cfg.Mode != "webhook" {
		return nil, fmt.Errorf("TELEGRAM_MODE must be either 'polling' or 'webhook', got '%s'", cfg.Mode)
	}

	// Get the webhook URL from environment.
	cfg.WebhookURL = os.Getenv("TELEGRAM_WEBHOOK_URL")
	if cfg.Mode == "webhook" {
		// In webhook mode, a public URL is necessary for Telegram to send updates.
		// If it's not provided, we cannot set up webhooks properly.
		if cfg.WebhookURL == "" {
			return nil, fmt.Errorf("TELEGRAM_WEBHOOK_URL must be set in webhook mode")
		}
		// (We assume the URL provided is reachable by Telegram; otherwise, webhook will not work.)
	}

	// Get the webhook listening port from environment.
	portStr := os.Getenv("TELEGRAM_WEBHOOK_PORT")
	if portStr == "" {
		// If no port is specified and we're in webhook mode, use a default port.
		// Telegram supports specific ports for webhooks: 443, 80, 88, or 8443.
		// We'll choose 8443 as a common default for webhook if not explicitly set.
		if cfg.Mode == "webhook" {
			cfg.WebhookPort = 8443
		} else {
			// In polling mode, the webhook port isn't used, so leaving it as 0 (or any value) is fine.
			cfg.WebhookPort = 0
		}
	} else {
		// If a port string is provided, try to convert it to an integer.
		port, err := strconv.Atoi(portStr)
		if err != nil {
			// If conversion fails (not a number), that's a configuration error.
			return nil, fmt.Errorf("TELEGRAM_WEBHOOK_PORT must be a number, got '%s'", portStr)
		}
		cfg.WebhookPort = port
	}
	// Note: We are not checking if the port is one of the allowed ports (443, 80, 88, 8443).
	// In a real application, you might want to ensure the port is valid for Telegram's requirements.

	// Get the optional webhook secret token from environment.
	cfg.WebhookSecret = os.Getenv("TELEGRAM_WEBHOOK_SECRET")
	// It's fine if the webhook secret is empty (Telegram bot API allows no secret),
	// but if provided, Telegram will include it in each update for verification.

	// Get the log level from environment.
	cfg.LogLevel = os.Getenv("TELEGRAM_LOG_LEVEL")
	if cfg.LogLevel == "" {
		// If not specified, default to "info" level for general usage.
		cfg.LogLevel = "info"
	}
	// We could also normalize log level to lower-case or restrict to known values,
	// but we'll trust the user for advanced values like "debug", "error", etc.

	// Get the log file path from environment.
	cfg.LogFilePath = os.Getenv("TELEGRAM_LOG_FILE_PATH")
	// If LogFilePath is empty, the application can default to logging to standard output.
	// If provided, the application should attempt to log to this file path.

	// Get the poll timeout (for long polling) from environment.
	pollTimeoutStr := os.Getenv("TELEGRAM_POLL_TIMEOUT")
	if pollTimeoutStr == "" {
		// Default poll timeout in seconds if not specified.
		// Long polling waits for updates; a typical default might be 30 seconds.
		cfg.PollTimeout = 30
	} else {
		// Parse the poll timeout string into an integer (seconds).
		pollTimeout, err := strconv.Atoi(pollTimeoutStr)
		if err != nil {
			return nil, fmt.Errorf("TELEGRAM_POLL_TIMEOUT must be a number (in seconds), got '%s'", pollTimeoutStr)
		}
		if pollTimeout < 0 {
			// Negative timeout doesn't make sense, so treat as an error.
			return nil, fmt.Errorf("TELEGRAM_POLL_TIMEOUT cannot be negative, got %d", pollTimeout)
		}
		cfg.PollTimeout = pollTimeout
	}

	// Get the max retry count from environment.
	retryMaxStr := os.Getenv("TELEGRAM_RETRY_MAX")
	if retryMaxStr == "" {
		// If not set, default to a reasonable number of retries, e.g., 3.
		cfg.RetryMax = 3
	} else {
		// Parse the retry count into an integer.
		retryMax, err := strconv.Atoi(retryMaxStr)
		if err != nil {
			return nil, fmt.Errorf("TELEGRAM_RETRY_MAX must be a number, got '%s'", retryMaxStr)
		}
		if retryMax < 0 {
			// Negative retries doesn't make sense; treat as error.
			return nil, fmt.Errorf("TELEGRAM_RETRY_MAX cannot be negative, got %d", retryMax)
		}
		cfg.RetryMax = retryMax
	}

	// At this point, all configuration fields are set (either from environment or defaults)
	// and basic validation has been done. Return the Config struct pointer.
	return cfg, nil
}
