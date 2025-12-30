package telegramreceiver

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// SecretToken is a string type that redacts itself in logs and string output.
// Use this for sensitive values like API tokens or secrets.
type SecretToken string

// LogValue implements slog.LogValuer to redact sensitive tokens in logs.
func (SecretToken) LogValue() slog.Value {
	return slog.StringValue("[REDACTED]")
}

// String returns "[REDACTED]" to prevent accidental exposure in fmt.Print, logs, etc.
func (SecretToken) String() string {
	return "[REDACTED]"
}

// Value returns the actual secret value. Use sparingly and never log the result.
func (t SecretToken) Value() string {
	return string(t)
}

// NewLogger creates a production-ready structured logger using Go's built-in log/slog.
// Logs are output in JSON format to stdout and optionally to a log file.
func NewLogger(logLevel slog.Level, logFilePath string) (*slog.Logger, error) {
	var logOutput io.Writer = os.Stdout

	if logFilePath != "" {
		safeDir := "./logs"
		cleanPath := filepath.Clean(filepath.Join(safeDir, filepath.Base(logFilePath)))

		logFile, err := os.OpenFile(cleanPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
		if err != nil {
			return nil, err
		}
		logOutput = io.MultiWriter(os.Stdout, logFile)
	}

	handler := slog.NewJSONHandler(logOutput, &slog.HandlerOptions{
		Level: logLevel,
	})

	logger := slog.New(handler)
	return logger, nil
}
