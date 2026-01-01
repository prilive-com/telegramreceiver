package telegramreceiver

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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

// Logger wraps slog.Logger and manages the underlying file handle for proper cleanup.
type Logger struct {
	*slog.Logger
	file *os.File
}

// Close releases the log file handle. Safe to call multiple times or on nil file.
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// NewLogger creates a production-ready structured logger using Go's built-in log/slog.
// Logs are output in JSON format to stdout and optionally to a log file.
// The caller MUST call Logger.Close() when done to release the file handle.
func NewLogger(logLevel slog.Level, logFilePath string) (*Logger, error) {
	var logOutput io.Writer = os.Stdout
	var logFile *os.File

	if logFilePath != "" {
		// Validate log path for security
		if err := validateLogPath(logFilePath); err != nil {
			return nil, err
		}

		// Ensure parent directory exists
		if err := ensureLogPath(logFilePath); err != nil {
			return nil, err
		}

		var err error
		logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return nil, err
		}
		logOutput = io.MultiWriter(os.Stdout, logFile)
	}

	handler := slog.NewJSONHandler(logOutput, &slog.HandlerOptions{
		Level: logLevel,
	})

	return &Logger{
		Logger: slog.New(handler),
		file:   logFile,
	}, nil
}

// validateLogPath ensures the log path is safe and doesn't allow path traversal.
func validateLogPath(path string) error {
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("log path: path traversal not allowed")
	}

	// Ensure the path doesn't point to sensitive system directories
	sensitiveRoots := []string{"/etc", "/bin", "/sbin", "/usr", "/var/log", "/root", "/home"}
	for _, root := range sensitiveRoots {
		if strings.HasPrefix(cleanPath, root+"/") || cleanPath == root {
			return fmt.Errorf("log path: cannot write to system directory %s", root)
		}
	}

	return nil
}

// ensureLogPath creates all parent directories for the log file with secure permissions.
func ensureLogPath(path string) error {
	dir := filepath.Dir(path)
	// Use 0700 for directories - owner only access
	return os.MkdirAll(dir, 0700)
}
