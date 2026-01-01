package telegramreceiver

import (
	"os"
	"testing"
	"time"
)

func TestNew_RequiresToken(t *testing.T) {
	_, err := New("")
	if err == nil {
		t.Error("expected error for empty token")
	}
}

func TestNew_ValidatesTokenFormat(t *testing.T) {
	_, err := New("invalid-token")
	if err == nil {
		t.Error("expected error for invalid token format")
	}
}

func TestNew_WithValidToken(t *testing.T) {
	token := os.Getenv("TEST_BOT_TOKEN")
	if token == "" {
		t.Skip("TEST_BOT_TOKEN not set")
	}

	client, err := New(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client.Config().BotToken != token {
		t.Error("token not set correctly")
	}
}

func TestNew_WithOptions(t *testing.T) {
	token := os.Getenv("TEST_BOT_TOKEN")
	if token == "" {
		t.Skip("TEST_BOT_TOKEN not set")
	}

	client, err := New(token,
		WithMode(ModeLongPolling),
		WithPolling(60, 50),
		WithPollingMaxErrors(5),
		WithRetry(2*time.Second, 30*time.Second, 1.5),
		WithRateLimit(20, 40),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg := client.Config()

	if cfg.Mode != ModeLongPolling {
		t.Errorf("expected ModeLongPolling, got %v", cfg.Mode)
	}
	if cfg.PollingTimeout != 60 {
		t.Errorf("expected timeout 60, got %d", cfg.PollingTimeout)
	}
	if cfg.PollingLimit != 50 {
		t.Errorf("expected limit 50, got %d", cfg.PollingLimit)
	}
	if cfg.PollingMaxErrors != 5 {
		t.Errorf("expected max errors 5, got %d", cfg.PollingMaxErrors)
	}
	if cfg.RetryInitialDelay != 2*time.Second {
		t.Errorf("expected initial delay 2s, got %v", cfg.RetryInitialDelay)
	}
	if cfg.RateLimitRequests != 20 {
		t.Errorf("expected rate limit 20, got %f", cfg.RateLimitRequests)
	}
}

func TestDefaultClientConfig(t *testing.T) {
	cfg := DefaultClientConfig()

	if cfg.Mode != ModeWebhook {
		t.Errorf("expected ModeWebhook default, got %v", cfg.Mode)
	}
	if cfg.WebhookPort != 8443 {
		t.Errorf("expected port 8443, got %d", cfg.WebhookPort)
	}
	if cfg.PollingTimeout != 30 {
		t.Errorf("expected timeout 30, got %d", cfg.PollingTimeout)
	}
	if cfg.RetryBackoffFactor != 2.0 {
		t.Errorf("expected backoff factor 2.0, got %f", cfg.RetryBackoffFactor)
	}
}

func TestPresets(t *testing.T) {
	t.Run("ProductionPreset", func(t *testing.T) {
		cfg := DefaultClientConfig()
		ProductionPreset().apply(&cfg)

		if cfg.PollingMaxErrors != 10 {
			t.Errorf("expected max errors 10, got %d", cfg.PollingMaxErrors)
		}
		if cfg.ShutdownTimeout != 30*time.Second {
			t.Errorf("expected shutdown timeout 30s, got %v", cfg.ShutdownTimeout)
		}
	})

	t.Run("DevelopmentPreset", func(t *testing.T) {
		cfg := DefaultClientConfig()
		DevelopmentPreset().apply(&cfg)

		if cfg.PollingMaxErrors != 3 {
			t.Errorf("expected max errors 3, got %d", cfg.PollingMaxErrors)
		}
		if cfg.ShutdownTimeout != 5*time.Second {
			t.Errorf("expected shutdown timeout 5s, got %v", cfg.ShutdownTimeout)
		}
	})
}

func TestUpdatesChannel(t *testing.T) {
	token := os.Getenv("TEST_BOT_TOKEN")
	if token == "" {
		t.Skip("TEST_BOT_TOKEN not set")
	}

	client, err := New(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updates := client.Updates()
	if updates == nil {
		t.Error("updates channel should not be nil")
	}
}
