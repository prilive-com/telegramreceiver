package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prilive-com/telegramreceiver/v2/telegramreceiver"
)

// This example demonstrates the new v3 API with simplified configuration.
// Run with: go run v3_example.go
//
// Required environment variable: TELEGRAM_BOT_TOKEN
// Optional: config.yaml file in the same directory

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Option 1: Simple programmatic configuration
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable required")
	}

	client, err := telegramreceiver.New(token,
		telegramreceiver.WithMode(telegramreceiver.ModeLongPolling),
		telegramreceiver.WithPolling(30, 100),
		telegramreceiver.WithPollingMaxErrors(5),
		telegramreceiver.WithRetry(time.Second, 60*time.Second, 2.0),
		telegramreceiver.DevelopmentPreset(),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Option 2: Load from config file + env vars + programmatic overrides
	// client, err := telegramreceiver.NewFromConfig("config.yaml",
	//     telegramreceiver.WithLogger(customLogger),
	// )

	if err := client.Start(ctx); err != nil {
		log.Fatalf("Failed to start: %v", err)
	}
	defer client.Stop()

	slog.Info("Telegram receiver running. Press Ctrl+C to stop.",
		"mode", client.Config().Mode,
	)

	// Consume updates
	for {
		select {
		case update := <-client.Updates():
			handleUpdate(update)

		case sig := <-sigChan:
			slog.Info("Received shutdown signal", "signal", sig)
			cancel()
			return
		}
	}
}

func handleUpdate(update telegramreceiver.TelegramUpdate) {
	fmt.Printf("\n--- Update ID: %d ---\n", update.UpdateID)

	if update.Message != nil {
		msg := update.Message
		if msg.From != nil {
			fmt.Printf("From: %s (@%s)\n", msg.From.FirstName, msg.From.Username)
		}
		if msg.Text != "" {
			fmt.Printf("Text: %s\n", msg.Text)
		}
	}

	if update.CallbackQuery != nil {
		cb := update.CallbackQuery
		fmt.Printf("Callback: %s from %s\n", cb.Data, cb.From.Username)
	}
}
