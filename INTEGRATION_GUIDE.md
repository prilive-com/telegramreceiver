# TelegramReceiver v2 Integration Guide

This guide helps developers integrate the telegramreceiver library into their applications for receiving Telegram bot webhook updates.

## Installation

```bash
go get github.com/prilive-com/telegramreceiver/v2@latest
```

Requires Go 1.24.3+

## Basic Setup

### 1. Environment Configuration

Create a `.env` file or set environment variables:

```bash
# Required
TLS_CERT_PATH=/path/to/cert.pem
TLS_KEY_PATH=/path/to/key.pem

# Optional (with defaults)
WEBHOOK_PORT=8443
WEBHOOK_SECRET=your_secret_token
ALLOWED_DOMAIN=your.domain.com
LOG_FILE_PATH=logs/telegramreceiver.log
RATE_LIMIT_REQUESTS=10
RATE_LIMIT_BURST=20
MAX_BODY_SIZE=1048576
```

### 2. Initialize the Library

```go
package main

import (
    "context"
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

    // Load configuration from environment
    cfg, err := telegramreceiver.LoadConfig()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Setup logger
    logger, err := telegramreceiver.NewLogger(slog.LevelInfo, cfg.LogFilePath)
    if err != nil {
        log.Fatalf("Failed to initialize logger: %v", err)
    }

    // Create updates channel
    updates := make(chan telegramreceiver.TelegramUpdate, 100)

    // Create webhook handler
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

    // Start HTTPS server
    go telegramreceiver.StartWebhookServer(ctx, cfg, handler, logger)

    // Handle shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    // Process updates
    for {
        select {
        case update := <-updates:
            handleUpdate(update)
        case <-sigChan:
            cancel()
            return
        }
    }
}
```

## Handling Updates

### Text Messages

```go
func handleUpdate(update telegramreceiver.TelegramUpdate) {
    if update.Message == nil {
        return
    }

    msg := update.Message

    // Access user info
    if msg.From != nil {
        userID := msg.From.ID           // int64
        username := msg.From.Username   // string
        firstName := msg.From.FirstName // string
    }

    // Access chat info
    chatID := msg.Chat.ID     // int64
    chatType := msg.Chat.Type // "private", "group", "supergroup", "channel"

    // Access message content
    text := msg.Text          // string
    messageID := msg.MessageID // int

    fmt.Printf("User %d in chat %d: %s\n", msg.From.ID, chatID, text)
}
```

### Command Handling

```go
import "strings"

func handleUpdate(update telegramreceiver.TelegramUpdate) {
    if update.Message == nil || update.Message.Text == "" {
        return
    }

    text := update.Message.Text

    // Check for commands
    if strings.HasPrefix(text, "/") {
        parts := strings.Fields(text)
        command := parts[0]
        args := parts[1:]

        switch command {
        case "/start":
            handleStart(update.Message)
        case "/help":
            handleHelp(update.Message)
        case "/echo":
            handleEcho(update.Message, args)
        default:
            handleUnknownCommand(update.Message, command)
        }
        return
    }

    // Handle regular text
    handleText(update.Message)
}
```

### Photos

```go
func handleUpdate(update telegramreceiver.TelegramUpdate) {
    if update.Message == nil {
        return
    }

    // Check for photos
    if len(update.Message.Photo) > 0 {
        // Photos come in multiple sizes, largest is last
        largest := update.Message.Photo[len(update.Message.Photo)-1]

        fileID := largest.FileID       // Use this to download the file
        width := largest.Width         // int
        height := largest.Height       // int
        fileSize := largest.FileSize   // int (optional)

        // Caption is optional
        caption := update.Message.Caption

        fmt.Printf("Received photo: %s (%dx%d)\n", fileID, width, height)
        if caption != "" {
            fmt.Printf("Caption: %s\n", caption)
        }
    }
}
```

### Documents

```go
func handleUpdate(update telegramreceiver.TelegramUpdate) {
    if update.Message == nil || update.Message.Document == nil {
        return
    }

    doc := update.Message.Document

    fileID := doc.FileID         // Use this to download
    fileName := doc.FileName     // Original filename
    mimeType := doc.MimeType     // e.g., "application/pdf"
    fileSize := doc.FileSize     // int64

    fmt.Printf("Received document: %s (%s, %d bytes)\n",
        fileName, mimeType, fileSize)
}
```

### Callback Queries (Inline Button Clicks)

```go
func handleUpdate(update telegramreceiver.TelegramUpdate) {
    if update.CallbackQuery != nil {
        cb := update.CallbackQuery

        callbackID := cb.ID       // Use this to answer the callback
        data := cb.Data           // Button data you set when creating the button
        userID := cb.From.ID      // User who clicked

        // Original message (if available)
        if cb.Message != nil {
            originalChatID := cb.Message.Chat.ID
            originalMessageID := cb.Message.MessageID
        }

        fmt.Printf("User %d clicked button with data: %s\n", userID, data)

        // You should answer the callback to remove loading state
        // Use telegramsender or direct API call:
        // POST /answerCallbackQuery with callback_query_id
    }
}
```

### Contacts

```go
func handleUpdate(update telegramreceiver.TelegramUpdate) {
    if update.Message == nil || update.Message.Contact == nil {
        return
    }

    contact := update.Message.Contact

    phoneNumber := contact.PhoneNumber // string
    firstName := contact.FirstName     // string
    lastName := contact.LastName       // string (optional)
    userID := contact.UserID           // int64 (optional, if contact is a Telegram user)

    fmt.Printf("Received contact: %s %s (%s)\n",
        firstName, lastName, phoneNumber)
}
```

### Locations

```go
func handleUpdate(update telegramreceiver.TelegramUpdate) {
    if update.Message == nil || update.Message.Location == nil {
        return
    }

    loc := update.Message.Location

    latitude := loc.Latitude   // float64
    longitude := loc.Longitude // float64

    fmt.Printf("Received location: %.6f, %.6f\n", latitude, longitude)
}
```

### Edited Messages

```go
func handleUpdate(update telegramreceiver.TelegramUpdate) {
    // Check for edited messages
    if update.EditedMessage != nil {
        msg := update.EditedMessage
        fmt.Printf("Message %d was edited: %s\n", msg.MessageID, msg.Text)
    }
}
```

### Reply Detection

```go
func handleUpdate(update telegramreceiver.TelegramUpdate) {
    if update.Message == nil {
        return
    }

    // Check if this is a reply to another message
    if update.Message.ReplyToMessage != nil {
        original := update.Message.ReplyToMessage
        fmt.Printf("This is a reply to message %d: %s\n",
            original.MessageID, original.Text)
    }
}
```

## Complete Handler Example

```go
func handleUpdate(update telegramreceiver.TelegramUpdate) {
    // Handle callback queries first
    if update.CallbackQuery != nil {
        handleCallback(update.CallbackQuery)
        return
    }

    // Handle edited messages
    if update.EditedMessage != nil {
        handleEditedMessage(update.EditedMessage)
        return
    }

    // Handle regular messages
    if update.Message == nil {
        return
    }

    msg := update.Message

    // Route by content type
    switch {
    case msg.Text != "" && strings.HasPrefix(msg.Text, "/"):
        handleCommand(msg)
    case msg.Text != "":
        handleTextMessage(msg)
    case len(msg.Photo) > 0:
        handlePhoto(msg)
    case msg.Document != nil:
        handleDocument(msg)
    case msg.Contact != nil:
        handleContact(msg)
    case msg.Location != nil:
        handleLocation(msg)
    default:
        log.Printf("Unhandled message type from user %d", msg.From.ID)
    }
}
```

## Integration with telegramsender

```go
import (
    "github.com/prilive-com/telegramreceiver/v2/telegramreceiver"
    "github.com/prilive-com/telegramsender/v2/telegramsender"
)

func main() {
    // Setup receiver
    receiverCfg, _ := telegramreceiver.LoadConfig()
    updates := make(chan telegramreceiver.TelegramUpdate, 100)
    // ... setup handler and server ...

    // Setup sender
    senderCfg, _ := telegramsender.LoadConfig()
    senderLogger, _ := telegramsender.NewLogger(slog.LevelInfo, senderCfg.LogFilePath)
    defer senderLogger.Close()
    api := telegramsender.NewTelegramAPI(senderLogger, senderCfg)

    // Echo bot example
    for update := range updates {
        if update.Message != nil && update.Message.Text != "" {
            api.SendMessage(ctx, telegramsender.MessageRequest{
                ChatID: update.Message.Chat.ID,
                Text:   "You said: " + update.Message.Text,
            })
        }
    }
}
```

## Webhook Setup with Telegram

After starting your server, register the webhook:

```bash
curl -X POST "https://api.telegram.org/bot<BOT_TOKEN>/setWebhook" \
     -H "Content-Type: application/json" \
     -d '{
       "url": "https://your.domain.com:8443",
       "secret_token": "<WEBHOOK_SECRET>",
       "allowed_updates": ["message", "edited_message", "callback_query"]
     }'
```

Verify webhook status:

```bash
curl "https://api.telegram.org/bot<BOT_TOKEN>/getWebhookInfo"
```

## Error Handling

```go
for update := range updates {
    func() {
        defer func() {
            if r := recover(); r != nil {
                log.Printf("Panic handling update %d: %v", update.UpdateID, r)
            }
        }()

        handleUpdate(update)
    }()
}
```

## Best Practices

1. **Always check for nil** before accessing pointer fields
2. **Use goroutines** for slow operations to not block the update loop
3. **Answer callback queries** to remove the loading indicator
4. **Log update IDs** for debugging
5. **Handle panics** to prevent crashes from malformed updates

## Troubleshooting

| Issue | Solution |
|-------|----------|
| No updates received | Check webhook is registered, TLS cert is valid |
| `nil pointer` panic | Add nil checks before accessing `.From`, `.Chat`, etc. |
| Rate limited | Increase `RATE_LIMIT_REQUESTS` and `RATE_LIMIT_BURST` |
| Circuit breaker open | Check for errors in your handler, reduce failure rate |

## Documentation

- [README.md](README.md) - Quick start and typed structs reference
- [CHANGELOG.md](CHANGELOG.md) - Version history
- [CLAUDE.md](CLAUDE.md) - Claude Code guidance
