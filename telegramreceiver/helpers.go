package telegramreceiver

import (
	"errors"
	"strconv"
	"strings"
)

// validateConfig performs pre-flight sanity checks based on receiver mode.
func validateConfig(cfg *Config) error {
	// Common validation
	if cfg.LogFilePath == "" {
		return errors.New("LOG_FILE_PATH must be set")
	}

	// Mode-specific validation
	switch cfg.ReceiverMode {
	case ModeWebhook:
		return validateWebhookConfig(cfg)
	case ModeLongPolling:
		return validatePollingConfig(cfg)
	default:
		return ErrInvalidReceiverMode
	}
}

// validateWebhookConfig validates webhook-specific configuration.
func validateWebhookConfig(cfg *Config) error {
	if cfg.WebhookPort < 1 || cfg.WebhookPort > 65535 {
		return errors.New("WebhookPort must be 1-65535")
	}
	if cfg.TLSCertPath == "" || cfg.TLSKeyPath == "" {
		return errors.New("TLS_CERT_PATH and TLS_KEY_PATH must be set for webhook mode")
	}
	return nil
}

// validatePollingConfig validates long polling-specific configuration.
func validatePollingConfig(cfg *Config) error {
	if cfg.BotToken.Value() == "" {
		return ErrBotTokenRequired
	}
	if err := ValidateBotToken(cfg.BotToken); err != nil {
		return err
	}
	if cfg.PollingTimeout < 0 || cfg.PollingTimeout > 60 {
		return ErrInvalidPollingTimeout
	}
	if cfg.PollingLimit < 1 || cfg.PollingLimit > 100 {
		return ErrInvalidPollingLimit
	}
	return nil
}

// ValidateBotToken validates that the bot token has the correct format.
// Telegram bot tokens follow the pattern: 123456789:ABCDefGhIJKlmNoPQRsTUVwxyZ
func ValidateBotToken(token SecretToken) error {
	t := token.Value()

	if len(t) < 10 {
		return errors.New("bot token: too short")
	}

	if !strings.Contains(t, ":") {
		return errors.New("bot token: must contain colon separator")
	}

	parts := strings.Split(t, ":")
	if len(parts) != 2 {
		return errors.New("bot token: must contain exactly one colon")
	}

	// First part must be a number (bot ID)
	if _, err := strconv.ParseInt(parts[0], 10, 64); err != nil {
		return errors.New("bot token: bot ID must be numeric")
	}

	// Second part should be at least 30 chars (the actual token hash)
	if len(parts[1]) < 30 {
		return errors.New("bot token: token hash too short")
	}

	return nil
}
