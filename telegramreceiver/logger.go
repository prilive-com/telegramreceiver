package telegramreceiver

import (
	"io"
	"log/slog"
	"os"
)

// NewLogger creates a production-ready structured logger using Go's built-in log/slog.
// Logs are output in JSON format to stdout and optionally to a log file.
func NewLogger(logLevel slog.Level, logFilePath string) (*slog.Logger, error) {
	// Open log file if logFilePath is specified
	var logOutput io.Writer = os.Stdout
	if logFilePath != "" {
		logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, err
		}
		logOutput = io.MultiWriter(os.Stdout, logFile)
	}

	// Configure the JSON handler with the specified log level and output destinations
	handler := slog.NewJSONHandler(logOutput, &slog.HandlerOptions{
		Level: logLevel,
	})

	logger := slog.New(handler)
	return logger, nil
}
