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

// validateConfig performs pre-flight sanity checks.
func validateConfig(cfg *Config) error {
	switch {
	case cfg.WebhookPort < 1 || cfg.WebhookPort > 65535:
		return errors.New("WebhookPort must be 1-65535")
	case cfg.TLSCertPath == "" || cfg.TLSKeyPath == "":
		return errors.New("TLS_CERT_PATH and TLS_KEY_PATH must be set")
	case cfg.LogFilePath == "":
		return errors.New("LOG_FILE_PATH must be set")
	default:
		return nil
	}
}
