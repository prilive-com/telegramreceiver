package telegramreceiver

import (
	"fmt"
	"os"
	"strconv"
)

// Default configuration values
const (
	defaultWebhookPort = 8443
	defaultLogLevel    = "info"
	defaultLogFilePath = "telegram_receiver.log"
)

// Config holds webhook-specific configuration settings for the Telegram receiver.
type Config struct {
	BotToken      string // BotToken is the Telegram API token for the bot (required).
	WebhookURL    string // WebhookURL is the public URL for webhook.
	WebhookPort   int    // WebhookPort is the local port number for webhook server.
	WebhookSecret string // WebhookSecret is an optional secret token for webhook.
	LogLevel      string // LogLevel sets logging verbosity (e.g., "debug", "info").
	LogFilePath   string // LogFilePath is the path for logging output file.
}

// LoadConfig reads configuration from environment variables and applies defaults.
func LoadConfig() (*Config, error) {
	cfg := &Config{
		BotToken:      os.Getenv("TELEGRAM_BOT_TOKEN"),
		WebhookURL:    os.Getenv("TELEGRAM_WEBHOOK_URL"),
		WebhookSecret: os.Getenv("TELEGRAM_WEBHOOK_SECRET"),
		LogLevel:      os.Getenv("LOG_LEVEL"),
		LogFilePath:   os.Getenv("LOG_FILE_PATH"),
	}

	if cfg.BotToken == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}
	if cfg.WebhookURL == "" {
		return nil, fmt.Errorf("TELEGRAM_WEBHOOK_URL is required")
	}

	if cfg.LogLevel == "" {
		cfg.LogLevel = defaultLogLevel
	}

	if cfg.LogFilePath == "" {
		cfg.LogFilePath = defaultLogFilePath
	}

	portStr := os.Getenv("TELEGRAM_WEBHOOK_PORT")
	if portStr == "" {
		cfg.WebhookPort = defaultWebhookPort
	} else {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid TELEGRAM_WEBHOOK_PORT: %v", err)
		}
		cfg.WebhookPort = port
	}

	return cfg, nil
}
