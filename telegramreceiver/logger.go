package telegramreceiver

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

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
