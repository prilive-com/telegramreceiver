# telegramreceiver v2

> **Note**: v2.x uses typed structs instead of `json.RawMessage`. See [Migration](#migration-from-v1) below.
>
> **New in v2.3**: Simplified configuration API with `New()` and `NewFromConfig()`. See [v3 API](#v3-api-simplified-configuration) below.

**telegramreceiver** is a production-ready Go 1.25+ library for consuming Telegram bot updates with resilience features. Supports both **webhook** and **long polling** modes.

---

## Features

| Capability | Details |
|------------|---------|
| **Dual Mode** | Webhook (HTTPS server) or Long Polling (API client) |
| **Message Types** | Text, photos, documents, contacts, locations, callback queries |
| **Typed Structs** | `Message`, `User`, `Chat`, `CallbackQuery` - no manual JSON parsing |
| **Typed Errors** | `WebhookError`, `TelegramAPIError` with status codes |
| **Auto Webhook** | Automatically registers/deletes webhooks on mode switch |
| **Health Checks** | `IsHealthy()`, `ConsecutiveErrors()` for K8s probes |
| **Security** | HTTPS (TLS 1.2+), X25519/P256 curves, host validation, constant-time secret check |
| **Resilience** | Rate-limiter, circuit-breaker, configurable retry limits |
| **Health Probes** | `/healthz` and `/readyz` endpoints for Kubernetes |
| **Performance** | Connection pooling, HTTP/2, body draining for connection reuse |
| **Observability** | Go 1.25 `log/slog` JSON logging with `SecretToken` redaction |
| **Configuration** | Environment variables with sensible defaults |

---

## Installation

```bash
go get github.com/prilive-com/telegramreceiver/v2@latest
```

Requires Go 1.25+

---

## Quick Start

### Webhook Mode

```go
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
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Handle shutdown signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    cfg, err := telegramreceiver.LoadConfig()
    if err != nil {
        log.Fatal(err)
    }

    logger, err := telegramreceiver.NewLogger(slog.LevelInfo, cfg.LogFilePath)
    if err != nil {
        log.Fatal(err)
    }

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

    // Auto-registers webhook if WEBHOOK_URL is set
    go telegramreceiver.StartWebhookServer(ctx, cfg, handler, logger)

    // Process updates
    for {
        select {
        case update := <-updates:
            if update.Message != nil {
                fmt.Printf("Message from %s: %s\n",
                    update.Message.From.Username,
                    update.Message.Text)
            }
        case <-sigChan:
            cancel()
            return
        }
    }
}
```

### Long Polling Mode

```go
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
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    cfg, err := telegramreceiver.LoadConfig()
    if err != nil {
        log.Fatal(err)
    }

    logger, err := telegramreceiver.NewLogger(slog.LevelInfo, cfg.LogFilePath)
    if err != nil {
        log.Fatal(err)
    }

    updates := make(chan telegramreceiver.TelegramUpdate, 100)

    // Automatically deletes webhook before starting
    client, err := telegramreceiver.StartLongPolling(ctx, cfg, updates, logger)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Stop()

    // Process updates
    for {
        select {
        case update := <-updates:
            if update.Message != nil {
                fmt.Printf("Message from %s: %s\n",
                    update.Message.From.Username,
                    update.Message.Text)
            }
        case <-sigChan:
            client.Stop()
            return
        }
    }
}
```

---

## v3 API (Simplified Configuration)

The new v3 API provides a cleaner interface with support for programmatic options, environment variables, and config files with proper precedence.

### Simple Usage

```go
package main

import (
    "context"
    "log"
    "os"
    "time"

    "github.com/prilive-com/telegramreceiver/v2/telegramreceiver"
)

func main() {
    ctx := context.Background()
    token := os.Getenv("TELEGRAM_BOT_TOKEN")

    // Create client with options
    client, err := telegramreceiver.New(token,
        telegramreceiver.WithMode(telegramreceiver.ModeLongPolling),
        telegramreceiver.WithPolling(30, 100),
        telegramreceiver.WithPollingMaxErrors(5),
        telegramreceiver.WithRetry(time.Second, 60*time.Second, 2.0),
    )
    if err != nil {
        log.Fatal(err)
    }

    if err := client.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Stop()

    // Process updates
    for update := range client.Updates() {
        if update.Message != nil {
            log.Printf("Message: %s", update.Message.Text)
        }
    }
}
```

### From Config File + Env Vars

```go
// Configuration precedence (highest to lowest):
// 1. Programmatic options (opts...)
// 2. Environment variables (TELEGRAM_*)
// 3. Config file
// 4. Default values

client, err := telegramreceiver.NewFromConfig("config.yaml",
    telegramreceiver.WithLogger(customLogger),  // Override from config
)
```

### Available Options

```go
// Receiver mode
telegramreceiver.WithMode(telegramreceiver.ModeLongPolling)
telegramreceiver.WithMode(telegramreceiver.ModeWebhook)

// Webhook settings
telegramreceiver.WithWebhook(8443, "secret-token")
telegramreceiver.WithWebhookTLS("/path/to/cert.pem", "/path/to/key.pem")
telegramreceiver.WithWebhookURL("https://example.com/webhook")
telegramreceiver.WithAllowedDomain("example.com")

// Long polling settings
telegramreceiver.WithPolling(30, 100)  // timeout, limit
telegramreceiver.WithPollingMaxErrors(5)
telegramreceiver.WithPollingDeleteWebhook(true)
telegramreceiver.WithAllowedUpdateTypes([]string{"message", "callback_query"})

// Retry settings (exponential backoff)
telegramreceiver.WithRetry(time.Second, 60*time.Second, 2.0)

// Rate limiting
telegramreceiver.WithRateLimit(10.0, 20)  // requests/sec, burst

// Circuit breaker
telegramreceiver.WithBreakerConfig(5, 2*time.Minute, 60*time.Second)

// Timeouts
telegramreceiver.WithTimeouts(10*time.Second, 2*time.Second, 15*time.Second, 120*time.Second)
telegramreceiver.WithMaxBodySize(1048576)

// Kubernetes-aware shutdown
telegramreceiver.WithShutdown(5*time.Second, 15*time.Second)

// Logging
telegramreceiver.WithLogger(slogLogger)
telegramreceiver.WithLogFile("logs/bot.log")

// Testing
telegramreceiver.WithHTTPClientOption(mockClient)

// Presets
telegramreceiver.ProductionPreset()
telegramreceiver.DevelopmentPreset()
```

### Config File Format

```yaml
# config.yaml
bot_token: "123456789:ABCdefGHI..."  # Or use TELEGRAM_BOT_TOKEN env var
mode: longpolling

# Long polling
polling_timeout: 30
polling_limit: 100
polling_max_errors: 10

# Retry
retry_initial_delay: 1s
retry_max_delay: 60s
retry_backoff_factor: 2.0

# Rate limiting
rate_limit_requests: 10.0
rate_limit_burst: 20

# Webhook (if mode: webhook)
webhook_port: 8443
webhook_secret: "secret"
```

---

## Receiver Modes

### Webhook Mode

Telegram pushes updates to your HTTPS server. Best for:
- Production environments with public IP/domain
- Multiple bot instances behind a load balancer
- Lower latency (instant delivery)

**Behavior:**
- If `WEBHOOK_URL` and `TELEGRAM_BOT_TOKEN` are set, the library automatically calls `setWebhook` on startup
- Requires TLS certificate and key
- Exposes `/healthz` and `/readyz` endpoints for Kubernetes

### Long Polling Mode

Your server pulls updates from Telegram API. Best for:
- Development/testing (no public IP needed)
- Single instance deployments
- NAT/firewall environments

**Behavior:**
- If `POLLING_DELETE_WEBHOOK=true`, calls `deleteWebhook` before starting
- Uses circuit breaker for resilience
- Configurable retry delay and max errors
- Provides `IsHealthy()` method for health checks

---

## Health Monitoring

### Webhook Mode (HTTP Endpoints)

```yaml
# Kubernetes probe configuration
livenessProbe:
  httpGet:
    path: /healthz
    port: 8443
    scheme: HTTPS
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /readyz
    port: 8443
    scheme: HTTPS
  initialDelaySeconds: 5
  periodSeconds: 5
```

### Long Polling Mode (Programmatic)

```go
client, _ := telegramreceiver.StartLongPolling(ctx, cfg, updates, logger)

// Check health status
if client.IsHealthy() {
    // Client is running and error count is below threshold
}

// Get current error count
errCount := client.ConsecutiveErrors()

// Get current update offset
offset := client.Offset()

// Check if running
if client.Running() {
    // Client is actively polling
}
```

---

## Webhook API Functions

The library provides functions for managing webhooks programmatically:

```go
// Register a webhook
err := telegramreceiver.SetWebhook(ctx, botToken, "https://example.com/webhook", "secret")

// Remove webhook (required before long polling)
err := telegramreceiver.DeleteWebhook(ctx, botToken, false) // false = keep pending updates

// Get current webhook info
info, err := telegramreceiver.GetWebhookInfo(ctx, botToken)
fmt.Printf("URL: %s, Pending: %d\n", info.URL, info.PendingUpdateCount)
```

### WebhookInfo Fields

```go
type WebhookInfo struct {
    URL                          string   // Current webhook URL
    HasCustomCertificate         bool     // Using self-signed cert
    PendingUpdateCount           int      // Updates waiting to be delivered
    IPAddress                    string   // Current webhook IP
    LastErrorDate                int64    // Unix timestamp of last error
    LastErrorMessage             string   // Last error description
    LastSynchronizationErrorDate int64    // Last sync error timestamp
    MaxConnections               int      // Max simultaneous connections
    AllowedUpdates               []string // Update types being received
}
```

---

## Functional Options

### Long Polling Client Options

```go
// Custom HTTP client
client := telegramreceiver.NewLongPollingClient(
    botToken, updates, logger,
    timeout, limit, retryDelay,
    breakerMaxReq, breakerInterval, breakerTimeout,
    telegramreceiver.WithHTTPClient(customHTTPClient),
)

// Set max consecutive errors (0 = unlimited)
client := telegramreceiver.NewLongPollingClient(
    // ... required params ...
    telegramreceiver.WithMaxErrors(5),
)

// Filter update types
client := telegramreceiver.NewLongPollingClient(
    // ... required params ...
    telegramreceiver.WithAllowedUpdates([]string{"message", "callback_query"}),
)

// Delete webhook before starting (default: false)
client := telegramreceiver.NewLongPollingClient(
    // ... required params ...
    telegramreceiver.WithDeleteWebhook(true),
)

// Custom circuit breaker
client := telegramreceiver.NewLongPollingClient(
    // ... required params ...
    telegramreceiver.WithCircuitBreaker(customBreaker),
)
```

---

## Typed Structs (v2)

v2 provides fully typed structs - no manual JSON parsing required:

### TelegramUpdate

```go
type TelegramUpdate struct {
    UpdateID      int            `json:"update_id"`
    Message       *Message       `json:"message,omitempty"`
    EditedMessage *Message       `json:"edited_message,omitempty"`
    CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}
```

### Message

```go
type Message struct {
    MessageID      int             `json:"message_id"`
    From           *User           `json:"from,omitempty"`
    Chat           *Chat           `json:"chat"`
    Date           int             `json:"date"`
    Text           string          `json:"text,omitempty"`
    ReplyToMessage *Message        `json:"reply_to_message,omitempty"`
    Photo          []PhotoSize     `json:"photo,omitempty"`
    Document       *Document       `json:"document,omitempty"`
    Caption        string          `json:"caption,omitempty"`
    Contact        *Contact        `json:"contact,omitempty"`
    Location       *Location       `json:"location,omitempty"`
}
```

### User

```go
type User struct {
    ID           int64  `json:"id"`
    IsBot        bool   `json:"is_bot"`
    FirstName    string `json:"first_name"`
    LastName     string `json:"last_name,omitempty"`
    Username     string `json:"username,omitempty"`
    LanguageCode string `json:"language_code,omitempty"`
}
```

### Chat

```go
type Chat struct {
    ID        int64  `json:"id"`
    Type      string `json:"type"`        // "private", "group", "supergroup", "channel"
    Title     string `json:"title,omitempty"`
    Username  string `json:"username,omitempty"`
    FirstName string `json:"first_name,omitempty"`
    LastName  string `json:"last_name,omitempty"`
}
```

### CallbackQuery

```go
type CallbackQuery struct {
    ID              string   `json:"id"`
    From            *User    `json:"from"`
    Message         *Message `json:"message,omitempty"`
    InlineMessageID string   `json:"inline_message_id,omitempty"`
    Data            string   `json:"data,omitempty"`
}
```

---

## Typed Errors

The library provides typed errors with HTTP status codes:

```go
// Webhook errors (HTTP handler)
type WebhookError struct {
    Code    int
    Message string
    Err     error
}

// Telegram API errors (long polling, webhook management)
type TelegramAPIError struct {
    Code        int
    Description string
    Err         error
}

// Sentinel errors
var (
    // Webhook handler errors
    ErrForbidden        = &WebhookError{Code: 403, Message: "forbidden"}
    ErrUnauthorized     = &WebhookError{Code: 401, Message: "unauthorized"}
    ErrMethodNotAllowed = &WebhookError{Code: 405, Message: "method not allowed"}
    ErrChannelBlocked   = &WebhookError{Code: 503, Message: "updates channel blocked"}

    // Configuration errors
    ErrBotTokenRequired      = errors.New("TELEGRAM_BOT_TOKEN is required for long polling mode")
    ErrInvalidReceiverMode   = errors.New("RECEIVER_MODE must be 'webhook' or 'longpolling'")
    ErrInvalidPollingTimeout = errors.New("POLLING_TIMEOUT must be between 0 and 60")
    ErrInvalidPollingLimit   = errors.New("POLLING_LIMIT must be between 1 and 100")
    ErrInvalidWebhookURL     = errors.New("WEBHOOK_URL must be a valid HTTPS URL")

    // Runtime errors
    ErrPollingAlreadyRunning = errors.New("long polling client is already running")
    ErrMaxRetriesExceeded    = errors.New("max consecutive retries exceeded")
    ErrUpdatesChannelFull    = errors.New("updates channel is full, dropping update")
)
```

---

## Sensitive Data Redaction

Use `SecretToken` type for automatic log redaction:

```go
token := telegramreceiver.SecretToken("my-secret-api-key")

// Automatically redacted in logs
logger.Info("config loaded", "webhook_secret", token)
// Output: {"webhook_secret": "[REDACTED]"}

// Automatically redacted in fmt.Print
fmt.Println(token)
// Output: [REDACTED]

// Access actual value when needed (use sparingly)
actualValue := token.Value()
```

---

## Usage Examples

### Handle Text Messages

```go
for update := range updates {
    if update.Message != nil && update.Message.Text != "" {
        userID := update.Message.From.ID
        chatID := update.Message.Chat.ID
        text := update.Message.Text

        fmt.Printf("User %d in chat %d: %s\n", userID, chatID, text)
    }
}
```

### Handle Commands

```go
import "strings"

for update := range updates {
    if update.Message != nil && strings.HasPrefix(update.Message.Text, "/") {
        parts := strings.Fields(update.Message.Text)
        command := parts[0]
        args := parts[1:]

        switch command {
        case "/start":
            // Handle /start command
        case "/help":
            // Handle /help command
        default:
            // Unknown command
        }
    }
}
```

### Handle Photos

```go
if update.Message != nil && len(update.Message.Photo) > 0 {
    // Photos come in multiple sizes, largest is last
    largest := update.Message.Photo[len(update.Message.Photo)-1]
    fmt.Printf("Photo file_id: %s (%dx%d)\n",
        largest.FileID,
        largest.Width,
        largest.Height)

    if update.Message.Caption != "" {
        fmt.Printf("Caption: %s\n", update.Message.Caption)
    }
}
```

### Handle Callback Queries (Inline Buttons)

```go
if update.CallbackQuery != nil {
    cb := update.CallbackQuery
    fmt.Printf("User %s clicked button with data: %s\n",
        cb.From.Username,
        cb.Data)

    // Answer the callback to remove loading state
    // (use telegramsender or direct API call)
}
```

### Handle Documents

```go
if update.Message != nil && update.Message.Document != nil {
    doc := update.Message.Document
    fmt.Printf("Document: %s (%s, %d bytes)\n",
        doc.FileName,
        doc.MimeType,
        doc.FileSize)
}
```

### Mode Selection at Runtime

```go
cfg, _ := telegramreceiver.LoadConfig()
updates := make(chan telegramreceiver.TelegramUpdate, 100)

switch cfg.ReceiverMode {
case telegramreceiver.ModeWebhook:
    handler := telegramreceiver.NewWebhookHandler(/* ... */)
    go telegramreceiver.StartWebhookServer(ctx, cfg, handler, logger)

case telegramreceiver.ModeLongPolling:
    client, _ := telegramreceiver.StartLongPolling(ctx, cfg, updates, logger)
    defer client.Stop()
}

// Process updates (same code for both modes)
for update := range updates {
    // Handle update
}
```

---

## Architecture

### Webhook Mode

```
┌────────┐   HTTPS Webhook   ┌─────────────────┐     Channel      ┌───────────────┐
│Telegram│ ────────────────▶ │ WebhookHandler  │ ───────────────▶ │Your App Logic │
└────────┘                   │  - rate-limit   │                  └───────────────┘
                             │  - circuit-break│
                             │  - health probes│
                             └─────────────────┘
```

### Long Polling Mode

```
┌────────┐   getUpdates API  ┌──────────────────┐     Channel      ┌───────────────┐
│Telegram│ ◀────────────────▶│LongPollingClient │───────────────▶ │Your App Logic │
└────────┘                   │  - circuit-break │                  └───────────────┘
                             │  - retry/backoff │
                             │  - health check  │
                             └──────────────────┘
```

---

## Environment Variables

### Mode Selection

| Variable | Default | Description |
|----------|---------|-------------|
| `RECEIVER_MODE` | `webhook` | Receiver mode: `webhook` or `longpolling` |
| `TELEGRAM_BOT_TOKEN` | *(required for polling)* | Bot token from @BotFather |

### Webhook Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `WEBHOOK_PORT` | `8443` | HTTPS listen port |
| `TLS_CERT_PATH` | *(required)* | Path to TLS certificate |
| `TLS_KEY_PATH` | *(required)* | Path to TLS private key |
| `WEBHOOK_SECRET` | *(optional)* | Secret token for Telegram verification |
| `ALLOWED_DOMAIN` | *(optional)* | Required Host header value |
| `WEBHOOK_URL` | *(optional)* | Public URL for auto-registration |

### Long Polling Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `POLLING_TIMEOUT` | `30` | Seconds to wait for updates (0-60) |
| `POLLING_LIMIT` | `100` | Max updates per request (1-100) |
| `POLLING_RETRY_DELAY` | `5s` | Delay between retries on error |
| `POLLING_MAX_ERRORS` | `10` | Max consecutive errors before stopping (0 = unlimited) |
| `POLLING_DELETE_WEBHOOK` | `false` | Delete existing webhook before starting |
| `ALLOWED_UPDATES` | *(empty)* | Comma-separated update types filter |

### Common Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_FILE_PATH` | `logs/telegramreceiver.log` | Log file path |
| `RATE_LIMIT_REQUESTS` | `10` | Requests per second |
| `RATE_LIMIT_BURST` | `20` | Burst tokens |
| `MAX_BODY_SIZE` | `1048576` | Max request body (1 MiB) |
| `READ_TIMEOUT` | `10s` | HTTP read timeout |
| `READ_HEADER_TIMEOUT` | `2s` | HTTP read header timeout |
| `WRITE_TIMEOUT` | `15s` | HTTP write timeout |
| `IDLE_TIMEOUT` | `120s` | HTTP idle timeout |
| `BREAKER_MAX_REQUESTS` | `5` | Circuit breaker half-open requests |
| `BREAKER_INTERVAL` | `2m` | Circuit breaker reset interval |
| `BREAKER_TIMEOUT` | `60s` | Circuit breaker open duration |
| `DRAIN_DELAY` | `5s` | Time to wait for LB to stop routing |
| `SHUTDOWN_TIMEOUT` | `15s` | Max time for graceful shutdown |

### Allowed Update Types

Valid values for `ALLOWED_UPDATES` (comma-separated):
- `message` - New incoming message
- `edited_message` - Message was edited
- `channel_post` - New channel post
- `edited_channel_post` - Channel post was edited
- `callback_query` - Inline button callback
- `inline_query` - Inline query
- `chosen_inline_result` - Inline result chosen
- `shipping_query` - Shipping query
- `pre_checkout_query` - Pre-checkout query
- `poll` - Poll state changed
- `poll_answer` - User changed poll answer

Example: `ALLOWED_UPDATES=message,callback_query`

---

## TLS Configuration

The server enforces modern TLS settings:

- **Minimum version**: TLS 1.2
- **Curve preferences**: X25519 (fast, secure), P256 (compatibility)
- **SHA-1**: Disabled in TLS 1.2 handshakes (Go 1.25 default)

---

## Graceful Shutdown

### Webhook Mode

1. Context cancelled (SIGTERM received)
2. Health endpoints start returning 503
3. Drain delay (`DRAIN_DELAY`) - allows load balancer to stop routing
4. Graceful shutdown (`SHUTDOWN_TIMEOUT`) - drains existing connections

### Long Polling Mode

```go
client, _ := telegramreceiver.StartLongPolling(ctx, cfg, updates, logger)

// On shutdown signal
client.Stop()  // Blocks until polling goroutine exits
// Safe to call Stop() multiple times
```

---

## Migration from v1

### Breaking Change: Typed Structs

v1 used `json.RawMessage` requiring manual parsing:

```go
// v1 - Manual parsing required
var msg map[string]interface{}
json.Unmarshal(update.Message, &msg)
text := msg["text"].(string) // Unsafe type assertion
```

v2 uses typed structs:

```go
// v2 - Direct access
text := update.Message.Text           // string
userID := update.Message.From.ID      // int64
chatID := update.Message.Chat.ID      // int64
```

### Migration Steps

1. Update import to include `/v2`: `github.com/prilive-com/telegramreceiver/v2/telegramreceiver`
2. Remove all `json.Unmarshal(update.Message, ...)` calls
3. Access fields directly: `update.Message.Text`, `update.Message.From.ID`
4. Handle nil checks: `if update.Message != nil { ... }`

---

## Integration Guide for External Developers

### Step 1: Add Dependency

```bash
go get github.com/prilive-com/telegramreceiver/v2@latest
```

### Step 2: Choose Your Mode

**Webhook** - If you have:
- Public domain with HTTPS
- TLS certificate
- Want instant message delivery

**Long Polling** - If you have:
- Development environment
- No public IP
- Firewall/NAT restrictions

### Step 3: Set Environment Variables

```bash
# Minimal for long polling
export RECEIVER_MODE=longpolling
export TELEGRAM_BOT_TOKEN=your_bot_token
export LOG_FILE_PATH=./logs/bot.log

# Minimal for webhook
export RECEIVER_MODE=webhook
export TLS_CERT_PATH=/path/to/cert.pem
export TLS_KEY_PATH=/path/to/key.pem
export LOG_FILE_PATH=./logs/bot.log
```

### Step 4: Implement Update Handler

```go
func handleUpdate(update telegramreceiver.TelegramUpdate) {
    // Always check for nil
    if update.Message == nil {
        return
    }

    // Handle different message types
    switch {
    case update.Message.Text != "":
        handleTextMessage(update.Message)
    case len(update.Message.Photo) > 0:
        handlePhoto(update.Message)
    case update.Message.Document != nil:
        handleDocument(update.Message)
    }
}
```

### Step 5: Run Your Bot

```go
func main() {
    cfg, _ := telegramreceiver.LoadConfig()
    logger, _ := telegramreceiver.NewLogger(slog.LevelInfo, cfg.LogFilePath)
    updates := make(chan telegramreceiver.TelegramUpdate, 100)

    // Start receiver based on mode
    if cfg.ReceiverMode == telegramreceiver.ModeLongPolling {
        client, _ := telegramreceiver.StartLongPolling(ctx, cfg, updates, logger)
        defer client.Stop()
    } else {
        handler := telegramreceiver.NewWebhookHandler(/* ... */)
        go telegramreceiver.StartWebhookServer(ctx, cfg, handler, logger)
    }

    // Process updates
    for update := range updates {
        handleUpdate(update)
    }
}
```

---

## Thread Safety

The library is designed for concurrent use:

| Component | Thread-Safe | Notes |
|-----------|-------------|-------|
| `WebhookHandler` | Yes | Can handle concurrent HTTP requests |
| `LongPollingClient` | Yes | Uses atomic operations for state |
| `Start()` / `Stop()` | Yes | Protected by atomic.Bool and sync.Once |
| Updates channel | Yes | Standard Go channel semantics |

**Safe patterns:**
```go
// OK: Multiple goroutines reading from updates
for i := 0; i < workers; i++ {
    go func() {
        for update := range updates {
            handleUpdate(update)
        }
    }()
}

// OK: Call Stop() multiple times (idempotent)
client.Stop()
client.Stop()  // No panic, no-op

// OK: Check health from multiple goroutines
go func() {
    for {
        if !client.IsHealthy() {
            alertAdmin()
        }
        time.Sleep(time.Second)
    }
}()
```

---

## Best Practices

### Production Checklist

- [ ] Set `POLLING_MAX_ERRORS` to a reasonable value (e.g., 10) to prevent infinite retry loops
- [ ] Use buffered updates channel (e.g., `make(chan TelegramUpdate, 100)`)
- [ ] Implement graceful shutdown with context cancellation
- [ ] Monitor `ConsecutiveErrors()` for alerting
- [ ] Set appropriate `RATE_LIMIT_*` values to prevent abuse
- [ ] Use `ALLOWED_UPDATES` to filter only needed update types

### Error Handling

```go
client, err := telegramreceiver.StartLongPolling(ctx, cfg, updates, logger)
if err != nil {
    if errors.Is(err, telegramreceiver.ErrBotTokenRequired) {
        log.Fatal("Bot token not configured")
    }
    if errors.Is(err, telegramreceiver.ErrPollingAlreadyRunning) {
        log.Warn("Polling already started")
        return
    }
    log.Fatal(err)
}
```

### Kubernetes Deployment

```yaml
# For long polling mode, use exec probe with health check
livenessProbe:
  exec:
    command:
      - /app/healthcheck  # Your binary that calls client.IsHealthy()
  initialDelaySeconds: 10
  periodSeconds: 30
```

---

## Docker Compose

See `Dockerfile`, `docker-compose.yml`, and `env.example` in the repo for production deployment.

```bash
docker compose build
docker compose up -d
docker compose logs -f
```

---

## Troubleshooting

### Long Polling Issues

| Problem | Cause | Solution |
|---------|-------|----------|
| `ErrBotTokenRequired` | Missing `TELEGRAM_BOT_TOKEN` | Set the environment variable |
| `ErrPollingAlreadyRunning` | Called `Start()` twice | Check `Running()` before starting |
| `409 Conflict` error | Another instance using same token | Stop other bots or use webhook |
| Updates not arriving | Webhook still registered | Set `POLLING_DELETE_WEBHOOK=true` or call `DeleteWebhook()` manually |
| `IsHealthy()` returns false | Too many consecutive errors | Check network, bot token, or Telegram API status |

### Webhook Issues

| Problem | Cause | Solution |
|---------|-------|----------|
| `401 Unauthorized` | Wrong `WEBHOOK_SECRET` | Match the secret used in `setWebhook` |
| `403 Forbidden` | Wrong `ALLOWED_DOMAIN` | Set to your public domain or remove |
| TLS handshake fails | Bad certificate | Use valid cert for your domain |
| No updates received | Webhook not registered | Set `WEBHOOK_URL` for auto-registration |
| Health probe fails | Server shutting down | Check `/healthz` response |

### Circuit Breaker

```go
// Check if circuit breaker is open
// The breaker opens after consecutive failures to protect the system

// Adjust settings if needed:
// BREAKER_MAX_REQUESTS=5   - requests in half-open state
// BREAKER_INTERVAL=2m      - time to reset error counts
// BREAKER_TIMEOUT=60s      - time in open state before half-open
```

---

## License

MIT
