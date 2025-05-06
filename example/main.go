package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/prilive-com/telegramreceiver/telegramreceiver"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Setup context clearly for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Capture system signals (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Load configuration clearly from environment variables
	cfg, err := telegramreceiver.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Setup structured logging clearly
	logger, err := telegramreceiver.NewLogger(slog.LevelInfo, cfg.LogFilePath)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Channel for receiving Telegram updates
	updatesChan := make(chan telegramreceiver.TelegramUpdate, 100)

	// WebhookHandler initialization
	webhookHandler := telegramreceiver.NewWebhookHandler(
		logger,
		cfg.WebhookSecret,
		cfg.AllowedDomain,
		updatesChan,

		cfg.RateLimitRequests,
		cfg.RateLimitBurst,
		cfg.MaxBodySize,

		cfg.BreakerMaxRequests,
		cfg.BreakerInterval,
		cfg.BreakerTimeout,
	)

	// Start HTTPS server clearly in background
	go func() {
		if err := telegramreceiver.StartWebhookServer(ctx, cfg, webhookHandler, logger); err != nil {
			logger.Error("Webhook server exited with error", "error", err)
		}
	}()

	logger.Info("Example Telegram receiver running. Press Ctrl+C to stop.")

	// Consume and pretty-print updates
	for {
		select {
		case update := <-updatesChan:
			printPrettyTelegramUpdate(update, logger)

		case sig := <-sigChan:
			logger.Info("Received shutdown signal", "signal", sig)
			cancel()
			return
		}
	}
}

// printPrettyTelegramUpdate prints incoming Telegram messages clearly and prettily.
func printPrettyTelegramUpdate(update telegramreceiver.TelegramUpdate, logger *slog.Logger) {
	logger.Info("Received new Telegram Update", "update_id", update.UpdateID)

	var prettyJSON map[string]interface{}
	if err := json.Unmarshal(update.Message, &prettyJSON); err != nil {
		logger.Error("Failed to parse Telegram message", "error", err)
		return
	}

	prettyData, err := json.MarshalIndent(prettyJSON, "", "  ")
	if err != nil {
		logger.Error("Failed to format Telegram message", "error", err)
		return
	}

	fmt.Printf("\n✅ New Telegram Message Received (Update ID: %d):\n%s\n",
		update.UpdateID, string(prettyData))
}
