package telegramreceiver

import (
	"errors"
	"os"
	"path/filepath"
)

// ensureLogPath creates all parent directories for the log file.
func ensureLogPath(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0o755)
}

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
	if cfg.PollingTimeout < 0 || cfg.PollingTimeout > 60 {
		return ErrInvalidPollingTimeout
	}
	if cfg.PollingLimit < 1 || cfg.PollingLimit > 100 {
		return ErrInvalidPollingLimit
	}
	return nil
}
