package telegramreceiver

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// getTestBotToken returns the bot token from TEST_BOT_TOKEN environment variable.
// Skips the test if the variable is not set.
func getTestBotToken(t *testing.T) SecretToken {
	t.Helper()
	token := os.Getenv("TEST_BOT_TOKEN")
	if token == "" {
		t.Skip("TEST_BOT_TOKEN environment variable not set")
	}
	return SecretToken(token)
}

func TestValidateConfig(t *testing.T) {
	// Get bot token from environment for long polling tests
	botToken := SecretToken(os.Getenv("TEST_BOT_TOKEN"))

	tests := []struct {
		name         string
		cfg          *Config
		wantErr      bool
		errMsg       string
		needsBotToken bool // Skip if TEST_BOT_TOKEN not set
	}{
		{
			name: "valid webhook config",
			cfg: &Config{
				ReceiverMode: ModeWebhook,
				WebhookPort:  8443,
				TLSCertPath:  "/path/to/cert.pem",
				TLSKeyPath:   "/path/to/key.pem",
				LogFilePath:  "logs/test.log",
			},
			wantErr: false,
		},
		{
			name: "valid long polling config",
			cfg: &Config{
				ReceiverMode:   ModeLongPolling,
				BotToken:       botToken,
				LogFilePath:    "logs/test.log",
				PollingTimeout: 30,
				PollingLimit:   100,
			},
			wantErr:      false,
			needsBotToken: true,
		},
		{
			name: "invalid receiver mode",
			cfg: &Config{
				ReceiverMode: "invalid",
				LogFilePath:  "logs/test.log",
			},
			wantErr: true,
			errMsg:  "RECEIVER_MODE must be 'webhook' or 'longpolling'",
		},
		{
			name: "webhook port too low",
			cfg: &Config{
				ReceiverMode: ModeWebhook,
				WebhookPort:  0,
				TLSCertPath:  "/path/to/cert.pem",
				TLSKeyPath:   "/path/to/key.pem",
				LogFilePath:  "logs/test.log",
			},
			wantErr: true,
			errMsg:  "WebhookPort must be 1-65535",
		},
		{
			name: "webhook port too high",
			cfg: &Config{
				ReceiverMode: ModeWebhook,
				WebhookPort:  70000,
				TLSCertPath:  "/path/to/cert.pem",
				TLSKeyPath:   "/path/to/key.pem",
				LogFilePath:  "logs/test.log",
			},
			wantErr: true,
			errMsg:  "WebhookPort must be 1-65535",
		},
		{
			name: "missing TLS cert path",
			cfg: &Config{
				ReceiverMode: ModeWebhook,
				WebhookPort:  8443,
				TLSCertPath:  "",
				TLSKeyPath:   "/path/to/key.pem",
				LogFilePath:  "logs/test.log",
			},
			wantErr: true,
			errMsg:  "TLS_CERT_PATH and TLS_KEY_PATH must be set for webhook mode",
		},
		{
			name: "missing TLS key path",
			cfg: &Config{
				ReceiverMode: ModeWebhook,
				WebhookPort:  8443,
				TLSCertPath:  "/path/to/cert.pem",
				TLSKeyPath:   "",
				LogFilePath:  "logs/test.log",
			},
			wantErr: true,
			errMsg:  "TLS_CERT_PATH and TLS_KEY_PATH must be set for webhook mode",
		},
		{
			name: "missing log file path",
			cfg: &Config{
				ReceiverMode: ModeWebhook,
				WebhookPort:  8443,
				TLSCertPath:  "/path/to/cert.pem",
				TLSKeyPath:   "/path/to/key.pem",
				LogFilePath:  "",
			},
			wantErr: true,
			errMsg:  "LOG_FILE_PATH must be set",
		},
		{
			name: "long polling missing bot token",
			cfg: &Config{
				ReceiverMode:   ModeLongPolling,
				LogFilePath:    "logs/test.log",
				PollingTimeout: 30,
				PollingLimit:   100,
			},
			wantErr: true,
			errMsg:  "TELEGRAM_BOT_TOKEN is required for long polling mode",
		},
		{
			name: "long polling invalid timeout",
			cfg: &Config{
				ReceiverMode:   ModeLongPolling,
				BotToken:       botToken,
				LogFilePath:    "logs/test.log",
				PollingTimeout: 100,
				PollingLimit:   100,
			},
			wantErr:       true,
			errMsg:        "POLLING_TIMEOUT must be between 0 and 60",
			needsBotToken: true,
		},
		{
			name: "long polling invalid limit",
			cfg: &Config{
				ReceiverMode:   ModeLongPolling,
				BotToken:       botToken,
				LogFilePath:    "logs/test.log",
				PollingTimeout: 30,
				PollingLimit:   0,
			},
			wantErr:       true,
			errMsg:        "POLLING_LIMIT must be between 1 and 100",
			needsBotToken: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip tests that need bot token if not set
			if tt.needsBotToken && botToken.Value() == "" {
				t.Skip("TEST_BOT_TOKEN environment variable not set")
			}

			err := validateConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && err.Error() != tt.errMsg {
				t.Errorf("validateConfig() error = %q, want %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestEnsureLogPath(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "create nested directories",
			path:    filepath.Join(tmpDir, "a", "b", "c", "test.log"),
			wantErr: false,
		},
		{
			name:    "existing directory",
			path:    filepath.Join(tmpDir, "test.log"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ensureLogPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ensureLogPath() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				dir := filepath.Dir(tt.path)
				if _, err := os.Stat(dir); os.IsNotExist(err) {
					t.Errorf("directory %s was not created", dir)
				}
			}
		})
	}
}

func TestConfig_FullDefaults(t *testing.T) {
	cfg := &Config{
		ReceiverMode:       ModeWebhook,
		WebhookPort:        8443,
		TLSCertPath:        "/cert.pem",
		TLSKeyPath:         "/key.pem",
		LogFilePath:        "logs/test.log",
		RateLimitRequests:  10,
		RateLimitBurst:     20,
		MaxBodySize:        1 << 20,
		ReadTimeout:        10 * time.Second,
		ReadHeaderTimeout:  2 * time.Second,
		WriteTimeout:       15 * time.Second,
		IdleTimeout:        120 * time.Second,
		BreakerMaxRequests: 5,
		BreakerInterval:    2 * time.Minute,
		BreakerTimeout:     60 * time.Second,
		DrainDelay:         5 * time.Second,
		ShutdownTimeout:    15 * time.Second,
	}

	if err := validateConfig(cfg); err != nil {
		t.Errorf("validateConfig() with full defaults should not error: %v", err)
	}
}

func TestConfig_LongPollingDefaults(t *testing.T) {
	botToken := getTestBotToken(t)

	cfg := &Config{
		ReceiverMode:              ModeLongPolling,
		BotToken:                  botToken,
		LogFilePath:               "logs/test.log",
		PollingTimeout:            30,
		PollingLimit:              100,
		PollingRetryInitialDelay:  time.Second,
		PollingRetryMaxDelay:      60 * time.Second,
		PollingRetryBackoffFactor: 2.0,
		BreakerMaxRequests:        5,
		BreakerInterval:           2 * time.Minute,
		BreakerTimeout:            60 * time.Second,
	}

	if err := validateConfig(cfg); err != nil {
		t.Errorf("validateConfig() with long polling defaults should not error: %v", err)
	}
}
