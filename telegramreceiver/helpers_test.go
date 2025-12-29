package telegramreceiver

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			cfg: &Config{
				WebhookPort: 8443,
				TLSCertPath: "/path/to/cert.pem",
				TLSKeyPath:  "/path/to/key.pem",
				LogFilePath: "logs/test.log",
			},
			wantErr: false,
		},
		{
			name: "port too low",
			cfg: &Config{
				WebhookPort: 0,
				TLSCertPath: "/path/to/cert.pem",
				TLSKeyPath:  "/path/to/key.pem",
				LogFilePath: "logs/test.log",
			},
			wantErr: true,
			errMsg:  "WebhookPort must be 1-65535",
		},
		{
			name: "port too high",
			cfg: &Config{
				WebhookPort: 70000,
				TLSCertPath: "/path/to/cert.pem",
				TLSKeyPath:  "/path/to/key.pem",
				LogFilePath: "logs/test.log",
			},
			wantErr: true,
			errMsg:  "WebhookPort must be 1-65535",
		},
		{
			name: "missing TLS cert path",
			cfg: &Config{
				WebhookPort: 8443,
				TLSCertPath: "",
				TLSKeyPath:  "/path/to/key.pem",
				LogFilePath: "logs/test.log",
			},
			wantErr: true,
			errMsg:  "TLS_CERT_PATH and TLS_KEY_PATH must be set",
		},
		{
			name: "missing TLS key path",
			cfg: &Config{
				WebhookPort: 8443,
				TLSCertPath: "/path/to/cert.pem",
				TLSKeyPath:  "",
				LogFilePath: "logs/test.log",
			},
			wantErr: true,
			errMsg:  "TLS_CERT_PATH and TLS_KEY_PATH must be set",
		},
		{
			name: "missing log file path",
			cfg: &Config{
				WebhookPort: 8443,
				TLSCertPath: "/path/to/cert.pem",
				TLSKeyPath:  "/path/to/key.pem",
				LogFilePath: "",
			},
			wantErr: true,
			errMsg:  "LOG_FILE_PATH must be set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
