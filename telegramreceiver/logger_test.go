package telegramreceiver

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestSecretToken_LogValue(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	token := SecretToken("super-secret-api-key-12345")
	logger.Info("test message", "token", token)

	output := buf.String()

	if strings.Contains(output, "super-secret-api-key-12345") {
		t.Error("log output should not contain the actual token value")
	}

	if !strings.Contains(output, "[REDACTED]") {
		t.Error("log output should contain [REDACTED]")
	}
}

func TestSecretToken_MultipleTokens(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	token1 := SecretToken("secret-one")
	token2 := SecretToken("secret-two")

	logger.Info("multiple tokens",
		"api_key", token1,
		"webhook_secret", token2,
	)

	output := buf.String()

	if strings.Contains(output, "secret-one") || strings.Contains(output, "secret-two") {
		t.Error("log output should not contain any token values")
	}

	// Should have two [REDACTED] entries
	count := strings.Count(output, "[REDACTED]")
	if count != 2 {
		t.Errorf("expected 2 [REDACTED] in output, got %d", count)
	}
}

func TestNewLogger(t *testing.T) {
	// Test with empty log file path (stdout only)
	logger, err := NewLogger(slog.LevelInfo, "")
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	if logger == nil {
		t.Fatal("NewLogger() returned nil logger")
	}
}
