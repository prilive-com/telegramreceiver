# telegramreceiver v2

> **Note**: v2.x uses typed structs instead of `json.RawMessage`. See [Migration](#migration-from-v1) below.

**telegramreceiver** is a production-ready Go 1.25+ library for consuming Telegram bot updates via HTTPS webhook with resilience features.

---

## Features

| Capability | Details |
|------------|---------|
| **Message Types** | Text, photos, documents, contacts, locations, callback queries |
| **Typed Structs** | `Message`, `User`, `Chat`, `CallbackQuery` - no manual JSON parsing |
| **Typed Errors** | `WebhookError` with HTTP status codes for clean error handling |
| **Security** | HTTPS (TLS 1.2+), X25519/P256 curves, host validation, constant-time secret check |
| **Resilience** | Rate-limiter, circuit-breaker, Kubernetes-aware graceful shutdown |
| **Health Probes** | `/healthz` and `/readyz` endpoints for Kubernetes liveness/readiness |
| **Performance** | Request-body max-size guard, sync.Pool buffer reuse |
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

```go
package main

import (
    "context"
    "fmt"
    "log"
    "log/slog"

    "github.com/prilive-com/telegramreceiver/v2/telegramreceiver"
)

func main() {
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

    go telegramreceiver.StartWebhookServer(context.Background(), cfg, handler, logger)

    // Process updates with typed structs
    for update := range updates {
        if update.Message != nil {
            fmt.Printf("Message from %s: %s\n",
                update.Message.From.Username,
                update.Message.Text)
        }

        if update.CallbackQuery != nil {
            fmt.Printf("Button click: %s\n", update.CallbackQuery.Data)
        }
    }
}
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
type WebhookError struct {
    Code    int
    Message string
    Err     error
}

// Sentinel errors
var (
    ErrForbidden        = &WebhookError{Code: 403, Message: "forbidden"}
    ErrUnauthorized     = &WebhookError{Code: 401, Message: "unauthorized"}
    ErrMethodNotAllowed = &WebhookError{Code: 405, Message: "method not allowed"}
    ErrChannelBlocked   = &WebhookError{Code: 503, Message: "updates channel blocked"}
)
```

---

## Sensitive Data Redaction

Use `SecretToken` type for automatic log redaction:

```go
token := telegramreceiver.SecretToken("my-secret-api-key")
logger.Info("config loaded", "webhook_secret", token)
// Output: {"webhook_secret": "[REDACTED]"}
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

### Handle Photos

```go
if update.Message != nil && len(update.Message.Photo) > 0 {
    // Photos come in multiple sizes, largest is last
    largest := update.Message.Photo[len(update.Message.Photo)-1]
    fmt.Printf("Photo file_id: %s\n", largest.FileID)

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

---

## Architecture

```
┌────────┐   HTTPS Webhook   ┌─────────────────┐     Channel      ┌───────────────┐
│Telegram│ ────────────────▶ │ WebhookHandler  │ ───────────────▶ │Your App Logic │
└────────┘                   │  - rate-limit   │                  └───────────────┘
                             │  - circuit-break│
                             │  - health probes│
                             └─────────────────┘
```

- **WebhookHandler**: Validates host, secret token, HTTP method. Rate-limits & circuit-breaks.
- **StartWebhookServer**: HTTPS server with health endpoints, configurable timeouts, and Kubernetes-aware graceful shutdown.
- **Config**: Populated from environment variables with sensible defaults.

---

## Health Endpoints

The server automatically exposes Kubernetes-compatible health endpoints:

| Endpoint | Purpose | Behavior |
|----------|---------|----------|
| `/healthz` | Liveness probe | Returns 200 OK, or 503 during shutdown |
| `/readyz` | Readiness probe | Returns 200 OK, or 503 during shutdown |

### Kubernetes Configuration

```yaml
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

### Graceful Shutdown Sequence

1. Context cancelled (SIGTERM received)
2. Health endpoints start returning 503
3. Drain delay (`DRAIN_DELAY`) - allows load balancer to stop routing
4. Graceful shutdown (`SHUTDOWN_TIMEOUT`) - drains existing connections

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `WEBHOOK_PORT` | `8443` | HTTPS listen port |
| `TLS_CERT_PATH` | *(required)* | Path to TLS certificate |
| `TLS_KEY_PATH` | *(required)* | Path to TLS private key |
| `LOG_FILE_PATH` | `logs/telegramreceiver.log` | Log file path |
| `WEBHOOK_SECRET` | *(optional)* | Secret token for Telegram verification |
| `ALLOWED_DOMAIN` | *(optional)* | Required Host header value |
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
| `DRAIN_DELAY` | `5s` | Time to wait for LB to stop routing before shutdown |
| `SHUTDOWN_TIMEOUT` | `15s` | Max time for graceful shutdown |

---

## TLS Configuration

The server enforces modern TLS settings:

- **Minimum version**: TLS 1.2
- **Curve preferences**: X25519 (fast, secure), P256 (compatibility)
- **SHA-1**: Disabled in TLS 1.2 handshakes (Go 1.25 default)

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

## Webhook Setup

Register your webhook with Telegram:

```bash
curl -X POST "https://api.telegram.org/bot<BOT_TOKEN>/setWebhook" \
     -H "Content-Type: application/json" \
     -d '{
       "url": "https://your.domain.com:8443",
       "secret_token": "<WEBHOOK_SECRET>"
     }'
```

---

## Docker Compose

See `Dockerfile`, `docker-compose.yml`, and `.env` in the repo for production deployment.

```bash
docker compose build
docker compose up -d
docker compose logs -f
```

---

## License

MIT © 2025 Prilive Com
