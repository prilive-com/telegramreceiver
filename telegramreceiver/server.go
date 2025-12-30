package telegramreceiver

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"
)

// ServerState tracks the shutdown state for health endpoints.
type ServerState struct {
	isShuttingDown atomic.Bool
}

// IsShuttingDown returns true if the server is in shutdown mode.
func (s *ServerState) IsShuttingDown() bool {
	return s.isShuttingDown.Load()
}

// StartWebhookServer starts the HTTPS webhook server with Kubernetes-aware
// graceful shutdown. It wraps the handler with health endpoints:
//   - /healthz - liveness probe (always 200 unless shutting down)
//   - /readyz  - readiness probe (503 during shutdown drain)
//
// If WebhookURL and BotToken are configured, it automatically registers
// the webhook with Telegram before starting the server.
func StartWebhookServer(ctx context.Context, cfg *Config, handler http.Handler, logger *slog.Logger) error {
	if err := validateConfig(cfg); err != nil {
		logger.Error("Configuration validation failed", "error", err)
		return err
	}

	if err := ensureLogPath(cfg.LogFilePath); err != nil {
		logger.Error("Failed to create log directory", "error", err)
		return err
	}

	// Auto-register webhook if URL and bot token are provided
	if cfg.WebhookURL != "" && cfg.BotToken.Value() != "" {
		logger.Info("Registering webhook with Telegram", "url", cfg.WebhookURL)
		if err := SetWebhook(ctx, cfg.BotToken, cfg.WebhookURL, cfg.WebhookSecret); err != nil {
			logger.Error("Failed to register webhook", "error", err)
			return fmt.Errorf("failed to register webhook: %w", err)
		}
		logger.Info("Webhook registered successfully")
	}

	state := &ServerState{}

	// Wrap handler with health endpoints
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if state.isShuttingDown.Load() {
			http.Error(w, "shutting down", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if state.isShuttingDown.Load() {
			http.Error(w, "shutting down", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.Handle("/", handler)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.WebhookPort),
		Handler:           mux,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    1 << 20, // 1 MB
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			CurvePreferences: []tls.CurveID{
				tls.X25519,    // Fast, secure, preferred
				tls.CurveP256, // Wide compatibility fallback
			},
		},
	}

	go func() {
		logger.Info("Webhook server starting", "port", cfg.WebhookPort)
		if err := server.ListenAndServeTLS(cfg.TLSCertPath, cfg.TLSKeyPath); !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Webhook server error", "error", err)
		}
	}()

	<-ctx.Done()

	// Kubernetes-aware shutdown sequence:
	// 1. Mark as shutting down (health endpoints return 503)
	// 2. Wait for drain delay (allows LB to stop routing new requests)
	// 3. Gracefully shutdown (drain existing connections)
	state.isShuttingDown.Store(true)
	logger.Info("Shutdown initiated, starting drain delay", "delay", cfg.DrainDelay)

	time.Sleep(cfg.DrainDelay)

	logger.Info("Drain delay complete, shutting down server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Graceful shutdown failed", "error", err)
		return err
	}
	logger.Info("Webhook server stopped gracefully")
	return nil
}

// StartLongPolling creates and starts a long polling client.
// It automatically deletes any existing webhook before starting.
// Returns the client so the caller can call Stop() when needed.
func StartLongPolling(ctx context.Context, cfg *Config, updates chan<- TelegramUpdate, logger *slog.Logger) (*LongPollingClient, error) {
	if err := validateConfig(cfg); err != nil {
		logger.Error("Configuration validation failed", "error", err)
		return nil, err
	}

	if err := ensureLogPath(cfg.LogFilePath); err != nil {
		logger.Error("Failed to create log directory", "error", err)
		return nil, err
	}

	// Build options based on config
	var opts []LongPollingOption
	if cfg.PollingMaxErrors != defaultMaxConsecutiveErrors {
		opts = append(opts, WithMaxErrors(cfg.PollingMaxErrors))
	}
	if len(cfg.AllowedUpdates) > 0 {
		opts = append(opts, WithAllowedUpdates(cfg.AllowedUpdates))
	}

	client := NewLongPollingClient(
		cfg.BotToken,
		updates,
		logger,
		cfg.PollingTimeout,
		cfg.PollingLimit,
		cfg.PollingRetryDelay,
		cfg.BreakerMaxRequests,
		cfg.BreakerInterval,
		cfg.BreakerTimeout,
		opts...,
	)

	if err := client.Start(ctx); err != nil {
		return nil, err
	}

	return client, nil
}
