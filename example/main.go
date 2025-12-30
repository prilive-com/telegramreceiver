package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/prilive-com/telegramreceiver/v2/telegramreceiver"
)

func main() {
	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Capture system signals (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Load configuration from environment variables
	cfg, err := telegramreceiver.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Setup structured logging
	logger, err := telegramreceiver.NewLogger(slog.LevelInfo, cfg.LogFilePath)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Channel for receiving Telegram updates
	updatesChan := make(chan telegramreceiver.TelegramUpdate, 100)

	// Start the appropriate receiver based on mode
	var pollingClient *telegramreceiver.LongPollingClient

	switch cfg.ReceiverMode {
	case telegramreceiver.ModeWebhook:
		logger.Info("Starting in webhook mode")
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

		go func() {
			if err := telegramreceiver.StartWebhookServer(ctx, cfg, webhookHandler, logger); err != nil {
				logger.Error("Webhook server exited with error", "error", err)
			}
		}()

	case telegramreceiver.ModeLongPolling:
		logger.Info("Starting in long polling mode")
		pollingClient, err = telegramreceiver.StartLongPolling(ctx, cfg, updatesChan, logger)
		if err != nil {
			log.Fatalf("Failed to start long polling: %v", err)
		}
		defer pollingClient.Stop()

	default:
		log.Fatalf("Unknown receiver mode: %s", cfg.ReceiverMode)
	}

	logger.Info("Telegram receiver running. Press Ctrl+C to stop.",
		"mode", cfg.ReceiverMode,
	)

	// Consume and print updates
	for {
		select {
		case update := <-updatesChan:
			printTelegramUpdate(update, logger)

		case sig := <-sigChan:
			logger.Info("Received shutdown signal", "signal", sig)
			cancel()
			if pollingClient != nil {
				pollingClient.Stop()
			}
			return
		}
	}
}

// printTelegramUpdate prints incoming Telegram messages using typed structs.
func printTelegramUpdate(update telegramreceiver.TelegramUpdate, logger *slog.Logger) {
	logger.Info("Received new Telegram Update", "update_id", update.UpdateID)

	// Handle regular messages
	if update.Message != nil {
		msg := update.Message
		fmt.Printf("\n--- New Message (Update ID: %d) ---\n", update.UpdateID)
		fmt.Printf("Message ID: %d\n", msg.MessageID)

		if msg.From != nil {
			fmt.Printf("From: %s %s (@%s, ID: %d)\n",
				msg.From.FirstName,
				msg.From.LastName,
				msg.From.Username,
				msg.From.ID)
		}

		if msg.Chat != nil {
			fmt.Printf("Chat: %s (ID: %d, Type: %s)\n",
				msg.Chat.Title,
				msg.Chat.ID,
				msg.Chat.Type)
		}

		if msg.Text != "" {
			fmt.Printf("Text: %s\n", msg.Text)
		}

		if len(msg.Photo) > 0 {
			fmt.Printf("Photo: %d size(s), largest: %s\n",
				len(msg.Photo),
				msg.Photo[len(msg.Photo)-1].FileID)
		}

		if msg.Document != nil {
			fmt.Printf("Document: %s (%s)\n",
				msg.Document.FileName,
				msg.Document.MimeType)
		}

		fmt.Println("-----------------------------------")
	}

	// Handle callback queries (inline button clicks)
	if update.CallbackQuery != nil {
		cb := update.CallbackQuery
		fmt.Printf("\n--- Callback Query (Update ID: %d) ---\n", update.UpdateID)
		fmt.Printf("Callback ID: %s\n", cb.ID)
		fmt.Printf("Data: %s\n", cb.Data)

		if cb.From != nil {
			fmt.Printf("From: %s (@%s, ID: %d)\n",
				cb.From.FirstName,
				cb.From.Username,
				cb.From.ID)
		}

		fmt.Println("--------------------------------------")
	}
}
