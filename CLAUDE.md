# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

**telegramreceiver v2** is a production-ready Go library for receiving Telegram bot webhook updates with resilience features: rate limiting (golang.org/x/time/rate), circuit breaker (sony/gobreaker), and graceful shutdown.

## Build Commands

```bash
# Build the library and example
go build ./...

# Run tests
go test -v ./...

# Format code
go fmt ./...

# Lint (go vet)
go vet ./...

# Run the example (requires TLS certs and env vars)
go run example/main.go

# Docker
docker compose build
docker compose up -d
docker compose logs -f
```

## Architecture

```
┌────────┐   HTTPS    ┌─────────────────┐   Channel   ┌──────────────┐
│Telegram│ ─────────▶ │  WebhookHandler │ ──────────▶ │ Your App     │
└────────┘            │  - rate limiter │             │ (typed msgs) │
                      │  - circuit break│             └──────────────┘
                      │  - body limit   │
                      └─────────────────┘
```

**Key files:**
- `telegramreceiver/telegram_api.go` - WebhookHandler with resilience patterns
- `telegramreceiver/types.go` - Typed Telegram structs (Message, User, Chat, etc.)
- `telegramreceiver/config.go` - Environment variable configuration
- `telegramreceiver/server.go` - HTTPS server with graceful shutdown
- `telegramreceiver/logger.go` - JSON structured logging (log/slog)
- `telegramreceiver/helpers.go` - Utilities

## Public API

```go
// Initialize
cfg, _ := telegramreceiver.LoadConfig()
logger, _ := telegramreceiver.NewLogger(slog.LevelInfo, cfg.LogFilePath)
updates := make(chan telegramreceiver.TelegramUpdate, 100)

handler := telegramreceiver.NewWebhookHandler(
    logger,
    cfg.WebhookSecret,
    cfg.AllowedDomain,
    updates,
    cfg.RateLimitRequests,
    cfg.RateLimitBurst,
    cfg.MaxBodySize,
    cfg.BreakerMaxRequests,
    cfg.BreakerInterval,
    cfg.BreakerTimeout,
)

go telegramreceiver.StartWebhookServer(ctx, cfg, handler, logger)

// Process typed updates
for update := range updates {
    if update.Message != nil {
        userID := update.Message.From.ID      // int64
        chatID := update.Message.Chat.ID      // int64
        text := update.Message.Text           // string
    }
}
```

## Typed Structs (v2)

v2 provides fully typed structs instead of `json.RawMessage`:

- `TelegramUpdate` - UpdateID, Message, EditedMessage, CallbackQuery
- `Message` - MessageID, From, Chat, Date, Text, Photo, Document, Caption, etc.
- `User` - ID, IsBot, FirstName, LastName, Username, LanguageCode
- `Chat` - ID, Type, Title, Username
- `CallbackQuery` - ID, From, Message, Data
- `PhotoSize`, `Document`, `Contact`, `Location`, `MessageEntity`

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `TLS_CERT_PATH` | Yes | - | Path to TLS certificate |
| `TLS_KEY_PATH` | Yes | - | Path to TLS private key |
| `WEBHOOK_PORT` | No | `8443` | HTTPS listen port |
| `WEBHOOK_SECRET` | No | - | Telegram secret token |
| `LOG_FILE_PATH` | No | `logs/telegramreceiver.log` | Log file path |

See README.md for full list of configuration options.

## Update Types Handled

| Type | Access | Description |
|------|--------|-------------|
| Text message | `update.Message.Text` | Plain text messages |
| Photo | `update.Message.Photo` | Array of PhotoSize |
| Document | `update.Message.Document` | File attachments |
| Callback | `update.CallbackQuery.Data` | Inline button clicks |
| Contact | `update.Message.Contact` | Shared contacts |
| Location | `update.Message.Location` | Shared locations |

## Common Patterns

### Nil-safe message handling
```go
if update.Message != nil && update.Message.From != nil {
    // Safe to access From fields
}
```

### Command detection
```go
if update.Message != nil && strings.HasPrefix(update.Message.Text, "/") {
    // Handle command
}
```

### Photo handling
```go
if len(update.Message.Photo) > 0 {
    // Largest size is last in array
    largest := update.Message.Photo[len(update.Message.Photo)-1]
    fileID := largest.FileID
}
```
