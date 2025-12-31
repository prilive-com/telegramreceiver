# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

**telegramreceiver v2** is a production-ready Go 1.25+ library for receiving Telegram bot updates via **webhook** or **long polling** with resilience features: rate limiting (golang.org/x/time/rate), circuit breaker (sony/gobreaker), Kubernetes-aware graceful shutdown, and typed errors.

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

### Webhook Mode
```
┌────────┐   HTTPS    ┌─────────────────┐   Channel   ┌──────────────┐
│Telegram│ ─────────▶ │  WebhookHandler │ ──────────▶ │ Your App     │
└────────┘            │  - rate limiter │             │ (typed msgs) │
                      │  - circuit break│             └──────────────┘
                      │  - health probes│
                      └─────────────────┘
```

### Long Polling Mode
```
┌────────┐  getUpdates  ┌──────────────────┐   Channel   ┌──────────────┐
│Telegram│ ◀──────────▶ │ LongPollingClient│ ──────────▶ │ Your App     │
└────────┘              │  - circuit break │             │ (typed msgs) │
                        │  - retry/backoff │             └──────────────┘
                        └──────────────────┘
```

**Key components:**
- `telegram_api.go` - WebhookHandler implementing http.Handler with rate limiting, circuit breaker, and constant-time secret validation
- `longpolling.go` - LongPollingClient with circuit breaker and automatic webhook deletion
- `webhook_api.go` - SetWebhook, DeleteWebhook, GetWebhookInfo API functions
- `types.go` - Typed Telegram structs (TelegramUpdate, Message, User, Chat, CallbackQuery)
- `errors.go` - Typed WebhookError and TelegramAPIError with status codes
- `config.go` - LoadConfig() reads all settings from environment variables
- `server.go` - StartWebhookServer() and StartLongPolling() with auto webhook management
- `logger.go` - NewLogger() with JSON output and SecretToken type for log redaction

## Public API

### Webhook Mode
```go
cfg, _ := telegramreceiver.LoadConfig()
logger, _ := telegramreceiver.NewLogger(slog.LevelInfo, cfg.LogFilePath)
updates := make(chan telegramreceiver.TelegramUpdate, 100)

handler := telegramreceiver.NewWebhookHandler(
    logger, cfg.WebhookSecret, cfg.AllowedDomain, updates,
    cfg.RateLimitRequests, cfg.RateLimitBurst, cfg.MaxBodySize,
    cfg.BreakerMaxRequests, cfg.BreakerInterval, cfg.BreakerTimeout,
)

// Auto-registers webhook if WEBHOOK_URL and TELEGRAM_BOT_TOKEN set
go telegramreceiver.StartWebhookServer(ctx, cfg, handler, logger)
```

### Long Polling Mode
```go
cfg, _ := telegramreceiver.LoadConfig()
logger, _ := telegramreceiver.NewLogger(slog.LevelInfo, cfg.LogFilePath)
updates := make(chan telegramreceiver.TelegramUpdate, 100)

// Auto-deletes webhook before starting
client, _ := telegramreceiver.StartLongPolling(ctx, cfg, updates, logger)
defer client.Stop()
```

### Webhook Management
```go
// Register webhook
telegramreceiver.SetWebhook(ctx, botToken, "https://example.com/webhook", "secret")

// Remove webhook
telegramreceiver.DeleteWebhook(ctx, botToken, false)

// Check webhook status
info, _ := telegramreceiver.GetWebhookInfo(ctx, botToken)
```

## Required Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `RECEIVER_MODE` | `webhook` | Mode: `webhook` or `longpolling` |
| `TELEGRAM_BOT_TOKEN` | *(required for polling)* | Bot token from @BotFather |
| `TLS_CERT_PATH` | *(required for webhook)* | Path to TLS certificate |
| `TLS_KEY_PATH` | *(required for webhook)* | Path to TLS private key |
| `WEBHOOK_PORT` | `8443` | HTTPS listen port |
| `WEBHOOK_URL` | - | Public URL for auto-registration |
| `POLLING_TIMEOUT` | `30` | Seconds to wait for updates (0-60) |
| `POLLING_LIMIT` | `100` | Max updates per request (1-100) |
| `POLLING_MAX_ERRORS` | `10` | Max consecutive errors before stopping (0 = unlimited) |
| `POLLING_DELETE_WEBHOOK` | `false` | Delete existing webhook before starting |
| `ALLOWED_UPDATES` | - | Comma-separated update types (empty = all) |

See README.md for full list including rate limiting and circuit breaker settings.

## Key Patterns

### Mode-based initialization
```go
switch cfg.ReceiverMode {
case telegramreceiver.ModeWebhook:
    // Webhook setup...
case telegramreceiver.ModeLongPolling:
    // Long polling setup...
}
```

### Typed errors
```go
// Webhook errors
var ErrForbidden = &WebhookError{Code: 403, Message: "forbidden"}

// API errors (for long polling)
var ErrBotTokenRequired = errors.New("TELEGRAM_BOT_TOKEN is required for long polling mode")

// Telegram API errors
type TelegramAPIError struct {
    Code        int
    Description string
    Err         error
}
```

### Sensitive token redaction in logs
```go
token := telegramreceiver.SecretToken("my-api-key")
logger.Info("config loaded", "token", token)
// Output: {"token": "[REDACTED]"}

// Access actual value when needed
actual := token.Value()
```

### Health monitoring (long polling)
```go
client, _ := telegramreceiver.StartLongPolling(ctx, cfg, updates, logger)

// Check client health for K8s probes
if client.IsHealthy() {
    // Client is running and within error threshold
}

// Get current state
errors := client.ConsecutiveErrors()  // int32
offset := client.Offset()             // int - last processed update ID
running := client.Running()           // bool
```

### Functional options
```go
client := telegramreceiver.NewLongPollingClient(
    token, updates, logger,
    timeout, limit, retryDelay,
    breakerMaxReq, breakerInterval, breakerTimeout,
    telegramreceiver.WithMaxErrors(5),              // Stop after 5 consecutive errors
    telegramreceiver.WithAllowedUpdates([]string{"message", "callback_query"}),
    telegramreceiver.WithHTTPClient(customClient),  // Custom HTTP client for testing
)
```

### Nil-safe message handling
```go
if update.Message != nil && update.Message.From != nil {
    userID := update.Message.From.ID  // int64
}
```

## Resilience Features

- **Rate limiter**: `golang.org/x/time/rate` - configurable requests/second and burst
- **Circuit breaker**: `sony/gobreaker/v2` - prevents cascading failures (both modes, with Go generics)
- **Body size limit**: `http.MaxBytesReader` - guards against large payloads
- **Buffer pool**: `sync.Pool` - reduces allocations in hot path
- **Health probes**: `/healthz` and `/readyz` for Kubernetes liveness/readiness (webhook mode)
- **Programmatic health**: `IsHealthy()`, `ConsecutiveErrors()` for K8s integration (polling mode)
- **TLS hardening**: X25519 + P256 curve preferences, TLS 1.2+ minimum
- **Auto webhook management**: Automatic setWebhook/deleteWebhook on mode switch
- **Connection draining**: Proper HTTP response body reading (`io.Copy(io.Discard, resp.Body)`)
- **Double-close protection**: `sync.Once` prevents panic on repeated Stop() calls

## Concurrency Patterns

```go
// Thread-safe state with atomic operations
type LongPollingClient struct {
    running           atomic.Bool
    consecutiveErrors atomic.Int32
    closeOnce         sync.Once  // Safe channel close
}

// Safe shutdown
func (c *LongPollingClient) Stop() {
    if !c.running.CompareAndSwap(true, false) {
        return  // Already stopped
    }
    c.closeOnce.Do(func() {
        close(c.stopCh)  // Only closes once
    })
    c.wg.Wait()
}
```

## Go 1.25 Features Used

- `sync.WaitGroup.Go` - cleaner goroutine spawning in tests
- Container-aware `GOMAXPROCS` - automatic tuning in Kubernetes
- SHA-1 disabled in TLS 1.2 handshakes by default
