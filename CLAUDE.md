# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

**telegramreceiver v2** is a production-ready Go 1.25+ library for receiving Telegram bot webhook updates with resilience features: rate limiting (golang.org/x/time/rate), circuit breaker (sony/gobreaker), Kubernetes-aware graceful shutdown, and typed errors.

**Import path**: `github.com/prilive-com/telegramreceiver/v2/telegramreceiver`

## Build Commands

```bash
# Build
go build ./...

# Run tests
go test -v ./...

# Run single test
go test -v ./telegramreceiver -run TestFunctionName

# Format and lint
go fmt ./... && go vet ./...

# Run example (requires TLS certs and env vars)
go run example/main.go
```

## Architecture

```
┌────────┐   HTTPS    ┌─────────────────┐   Channel   ┌──────────────┐
│Telegram│ ─────────▶ │  WebhookHandler │ ──────────▶ │ Your App     │
└────────┘            │  - rate limiter │             │ (typed msgs) │
                      │  - circuit break│             └──────────────┘
                      │  - body limit   │
                      │  - health probes│
                      └─────────────────┘
```

**Key components:**
- `telegram_api.go` - WebhookHandler implementing http.Handler with rate limiting, circuit breaker, and constant-time secret validation
- `types.go` - Typed Telegram structs (TelegramUpdate, Message, User, Chat, CallbackQuery)
- `errors.go` - Typed WebhookError with HTTP status codes (ErrForbidden, ErrUnauthorized, etc.)
- `config.go` - LoadConfig() reads all settings from environment variables
- `server.go` - StartWebhookServer() with TLS 1.2+, health endpoints (/healthz, /readyz), and Kubernetes-aware graceful shutdown
- `logger.go` - NewLogger() with JSON output and SecretToken type for log redaction

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

## Required Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `TLS_CERT_PATH` | *(required)* | Path to TLS certificate |
| `TLS_KEY_PATH` | *(required)* | Path to TLS private key |
| `WEBHOOK_PORT` | `8443` | HTTPS listen port |
| `WEBHOOK_SECRET` | - | Telegram secret token (validated via constant-time compare) |
| `DRAIN_DELAY` | `5s` | Time to wait for LB to stop routing before shutdown |
| `SHUTDOWN_TIMEOUT` | `15s` | Max time for graceful shutdown |

See README.md for full list including rate limiting and circuit breaker settings.

## Key Patterns

### Kubernetes-aware graceful shutdown
```go
// Server exposes /healthz and /readyz endpoints
// Shutdown sequence:
// 1. Context cancelled → health endpoints return 503
// 2. Drain delay (DRAIN_DELAY) → allows LB to stop routing
// 3. Graceful shutdown (SHUTDOWN_TIMEOUT) → drains connections
```

### Typed errors
```go
// errors.go provides typed errors with HTTP status codes
var (
    ErrForbidden        = &WebhookError{Code: 403, Message: "forbidden"}
    ErrUnauthorized     = &WebhookError{Code: 401, Message: "unauthorized"}
    ErrMethodNotAllowed = &WebhookError{Code: 405, Message: "method not allowed"}
    ErrChannelBlocked   = &WebhookError{Code: 503, Message: "updates channel blocked"}
)
```

### Sensitive token redaction in logs
```go
// SecretToken automatically redacts itself in slog output
token := telegramreceiver.SecretToken("my-api-key")
logger.Info("config loaded", "token", token)
// Output: {"token": "[REDACTED]"}
```

### Nil-safe message handling
```go
if update.Message != nil && update.Message.From != nil {
    userID := update.Message.From.ID  // int64
}
```

## Resilience Features

- **Rate limiter**: `golang.org/x/time/rate` - configurable requests/second and burst
- **Circuit breaker**: `sony/gobreaker` - prevents cascading failures
- **Body size limit**: `http.MaxBytesReader` - guards against large payloads
- **Buffer pool**: `sync.Pool` - reduces allocations in hot path
- **Health probes**: `/healthz` and `/readyz` for Kubernetes liveness/readiness
- **TLS hardening**: X25519 + P256 curve preferences, TLS 1.2+ minimum

## Go 1.25 Features Used

- `sync.WaitGroup.Go` - cleaner goroutine spawning in tests
- Container-aware `GOMAXPROCS` - automatic tuning in Kubernetes
- SHA-1 disabled in TLS 1.2 handshakes by default
