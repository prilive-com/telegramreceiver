package telegramreceiver

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

func StartWebhookServer(ctx context.Context, cfg *Config, handler http.Handler, logger *slog.Logger) error {
	if err := validateConfig(cfg); err != nil {
		logger.Error("Configuration validation failed", "error", err)
		return err
	}

	if err := ensureLogPath(cfg.LogFilePath); err != nil {
		logger.Error("Failed to create log directory", "error", err)
		return err
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.WebhookPort),
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	go func() {
		logger.Info("Webhook server starting", "port", cfg.WebhookPort)
		if err := server.ListenAndServeTLS(cfg.TLSCertPath, cfg.TLSKeyPath); !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Webhook server error", "error", err)
		}
	}()

	<-ctx.Done()
	logger.Info("Shutting down webhook server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Graceful shutdown failed", "error", err)
		return err
	}
	logger.Info("Webhook server stopped gracefully")
	return nil
}
