package telegramreceiver

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Clear any existing env vars
	envVars := []string{
		"WEBHOOK_PORT", "TLS_CERT_PATH", "TLS_KEY_PATH", "LOG_FILE_PATH",
		"WEBHOOK_SECRET", "ALLOWED_DOMAIN", "RATE_LIMIT_REQUESTS", "RATE_LIMIT_BURST",
		"MAX_BODY_SIZE", "READ_TIMEOUT", "READ_HEADER_TIMEOUT", "WRITE_TIMEOUT",
		"IDLE_TIMEOUT", "BREAKER_MAX_REQUESTS", "BREAKER_INTERVAL", "BREAKER_TIMEOUT",
		"DRAIN_DELAY", "SHUTDOWN_TIMEOUT",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Check defaults
	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"WebhookPort", cfg.WebhookPort, 8443},
		{"LogFilePath", cfg.LogFilePath, "logs/telegramreceiver.log"},
		{"RateLimitRequests", cfg.RateLimitRequests, 10.0},
		{"RateLimitBurst", cfg.RateLimitBurst, 20},
		{"MaxBodySize", cfg.MaxBodySize, int64(1048576)},
		{"ReadTimeout", cfg.ReadTimeout, 10 * time.Second},
		{"ReadHeaderTimeout", cfg.ReadHeaderTimeout, 2 * time.Second},
		{"WriteTimeout", cfg.WriteTimeout, 15 * time.Second},
		{"IdleTimeout", cfg.IdleTimeout, 120 * time.Second},
		{"BreakerMaxRequests", cfg.BreakerMaxRequests, uint32(5)},
		{"BreakerInterval", cfg.BreakerInterval, 2 * time.Minute},
		{"BreakerTimeout", cfg.BreakerTimeout, 60 * time.Second},
		{"DrainDelay", cfg.DrainDelay, 5 * time.Second},
		{"ShutdownTimeout", cfg.ShutdownTimeout, 15 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestLoadConfig_CustomValues(t *testing.T) {
	// Set custom values
	os.Setenv("WEBHOOK_PORT", "9443")
	os.Setenv("TLS_CERT_PATH", "/path/to/cert.pem")
	os.Setenv("TLS_KEY_PATH", "/path/to/key.pem")
	os.Setenv("WEBHOOK_SECRET", "my-secret")
	os.Setenv("RATE_LIMIT_REQUESTS", "50")
	os.Setenv("DRAIN_DELAY", "10s")
	defer func() {
		os.Unsetenv("WEBHOOK_PORT")
		os.Unsetenv("TLS_CERT_PATH")
		os.Unsetenv("TLS_KEY_PATH")
		os.Unsetenv("WEBHOOK_SECRET")
		os.Unsetenv("RATE_LIMIT_REQUESTS")
		os.Unsetenv("DRAIN_DELAY")
	}()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.WebhookPort != 9443 {
		t.Errorf("WebhookPort = %d, want 9443", cfg.WebhookPort)
	}
	if cfg.TLSCertPath != "/path/to/cert.pem" {
		t.Errorf("TLSCertPath = %s, want /path/to/cert.pem", cfg.TLSCertPath)
	}
	if cfg.WebhookSecret != "my-secret" {
		t.Errorf("WebhookSecret = %s, want my-secret", cfg.WebhookSecret)
	}
	if cfg.RateLimitRequests != 50.0 {
		t.Errorf("RateLimitRequests = %f, want 50.0", cfg.RateLimitRequests)
	}
	if cfg.DrainDelay != 10*time.Second {
		t.Errorf("DrainDelay = %v, want 10s", cfg.DrainDelay)
	}
}

func TestLoadConfig_InvalidValues(t *testing.T) {
	tests := []struct {
		name   string
		envVar string
		value  string
	}{
		{"invalid port", "WEBHOOK_PORT", "not-a-number"},
		{"invalid rate limit", "RATE_LIMIT_REQUESTS", "invalid"},
		{"invalid duration", "READ_TIMEOUT", "not-a-duration"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.envVar, tt.value)
			defer os.Unsetenv(tt.envVar)

			_, err := LoadConfig()
			if err == nil {
				t.Errorf("LoadConfig() expected error for %s=%s", tt.envVar, tt.value)
			}
		})
	}
}

func TestGetEnv(t *testing.T) {
	// Test with existing env var
	os.Setenv("TEST_VAR", "test-value")
	defer os.Unsetenv("TEST_VAR")

	if got := getEnv("TEST_VAR", "default"); got != "test-value" {
		t.Errorf("getEnv() = %s, want test-value", got)
	}

	// Test with non-existing env var
	if got := getEnv("NON_EXISTING_VAR", "default"); got != "default" {
		t.Errorf("getEnv() = %s, want default", got)
	}
}
