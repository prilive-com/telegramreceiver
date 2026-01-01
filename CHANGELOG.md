# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.3.0] - 2026-01-01

### Added

- **v3 API** - Simplified configuration with modern Go library patterns
  - `New(token, opts...)` - Simple programmatic constructor
  - `NewFromConfig(path, opts...)` - Multi-source config (file + env + options)
  - `Client` type with `Start()`, `Stop()`, `Updates()`, `Config()`, `WebhookHandler()`
  - Interface-based `Option` pattern for type-safe configuration

- **Configuration Options** (v3)
  - `WithMode(mode)` - Set receiver mode (webhook/longpolling)
  - `WithWebhook(port, secret)` - Configure webhook settings
  - `WithWebhookTLS(certPath, keyPath)` - Set TLS certificate paths
  - `WithWebhookURL(url)` - Set webhook URL for auto-registration
  - `WithPolling(timeout, limit)` - Configure long polling
  - `WithPollingMaxErrors(n)` - Set max consecutive errors
  - `WithPollingDeleteWebhook(bool)` - Delete webhook before polling
  - `WithAllowedUpdateTypes(types)` - Filter update types
  - `WithRetry(initialDelay, maxDelay, factor)` - Exponential backoff settings
  - `WithRateLimit(rps, burst)` - Rate limiting settings
  - `WithBreakerConfig(maxReq, interval, timeout)` - Circuit breaker
  - `WithTimeouts(read, readHeader, write, idle)` - HTTP server timeouts
  - `WithShutdown(drainDelay, timeout)` - Kubernetes-aware shutdown
  - `WithLogger(logger)` - Custom slog.Logger
  - `WithLogFile(path)` - Log file path
  - `WithHTTPClientOption(client)` - Custom HTTP client for testing

- **Presets** (v3)
  - `ProductionPreset()` - Production-optimized settings
  - `DevelopmentPreset()` - Development-friendly settings

- **Multi-source Configuration**
  - koanf-based configuration loading
  - Precedence: defaults → config file → env vars (TELEGRAM_*) → programmatic options
  - YAML config file support

- **Validation**
  - go-playground/validator integration
  - Actionable error messages with remediation hints

- **New Files**
  - `client.go` - v3 Client type and constructors
  - `options.go` - Option interface and With* functions
  - `client_test.go` - Tests for v3 API
  - `interfaces.go` - Consumer-side interfaces (Receiver, HTTPClient, etc.)
  - `example/v3/main.go` - v3 API example
  - `example/v3/config.yaml` - Example config file

### Changed

- **Exponential Backoff** - Added cryptographic jitter using `crypto/rand`
  - Prevents thundering herd in distributed systems
  - New config: `RetryInitialDelay`, `RetryMaxDelay`, `RetryBackoffFactor`

### Deprecated

- `NewLongPollingClient()` - Use `New()` with `WithMode(ModeLongPolling)` instead
- `NewWebhookHandler()` - Use `New()` with `WithMode(ModeWebhook)` and `client.WebhookHandler()`
- `StartWebhookServer()` - Use `New()` with `client.Start()`
- `StartLongPolling()` - Use `New()` with `client.Start()`

All deprecated functions will be removed in v4.

## [2.2.0] - 2025-12-31

### Added

- **Exponential Backoff with Crypto Jitter** for retry logic
- **Interfaces** for testability (Receiver, HTTPClient, WebhookProcessor, UpdateHandler)

### Changed

- Retry configuration now uses `PollingRetryInitialDelay`, `PollingRetryMaxDelay`, `PollingRetryBackoffFactor`

## [2.1.1] - 2025-12-30

### Changed

- **Webhook deletion is now opt-in** - Long polling no longer automatically deletes webhooks
  - Set `POLLING_DELETE_WEBHOOK=true` to enable webhook deletion before starting
  - Default is `false` - assumes no webhook exists or user manages webhooks manually
  - Added `WithDeleteWebhook(bool)` functional option

### Fixed

- Unexpected API calls when starting long polling without a webhook configured

## [2.1.0] - 2025-12-30

### Added

- **Long Polling Mode** - Alternative to webhooks for development and NAT environments
  - `LongPollingClient` with circuit breaker and automatic retry
  - `StartLongPolling()` convenience function with auto webhook deletion
  - `RECEIVER_MODE` environment variable to switch between `webhook` and `longpolling`

- **Webhook Management API**
  - `SetWebhook()` - Register webhook with Telegram
  - `DeleteWebhook()` - Remove webhook (with optional pending updates drop)
  - `GetWebhookInfo()` - Query current webhook status
  - `WebhookInfo` struct with all Telegram webhook fields

- **Health Monitoring** (Long Polling)
  - `IsHealthy()` - Check if client is running and within error threshold
  - `ConsecutiveErrors()` - Get current consecutive error count
  - `Offset()` - Get current update offset
  - `Running()` - Check if polling is active

- **Functional Options**
  - `WithMaxErrors(n)` - Set max consecutive errors before stopping (0 = unlimited)
  - `WithAllowedUpdates(types)` - Filter update types
  - `WithHTTPClient(client)` - Custom HTTP client for testing
  - `WithCircuitBreaker(breaker)` - Custom circuit breaker

- **New Environment Variables**
  - `RECEIVER_MODE` - Select `webhook` or `longpolling`
  - `POLLING_TIMEOUT` - Long polling timeout (0-60 seconds)
  - `POLLING_LIMIT` - Max updates per request (1-100)
  - `POLLING_RETRY_DELAY` - Delay between retries
  - `POLLING_MAX_ERRORS` - Max errors before stopping
  - `ALLOWED_UPDATES` - Comma-separated update type filter

- **New Errors**
  - `ErrBotTokenRequired` - Bot token missing for long polling
  - `ErrPollingAlreadyRunning` - Attempted double start
  - `ErrMaxRetriesExceeded` - Consecutive error limit reached
  - `ErrInvalidPollingTimeout` - Timeout out of range
  - `ErrInvalidPollingLimit` - Limit out of range

- **Documentation**
  - Thread Safety section in README
  - Best Practices section with production checklist
  - Troubleshooting guide for common issues

### Changed

- **Circuit Breaker** - Migrated from `sony/gobreaker` v1 to v2 (Go generics)
- **HTTP Client** - Added proper transport timeouts and connection draining
- **Shutdown** - Added `sync.Once` to prevent double-close panic on Stop()
- **State Tracking** - Using `atomic.Int32` for thread-safe error counting

### Fixed

- Double-close panic when calling `Stop()` multiple times
- HTTP connection leaks from unread response bodies
- Race condition in concurrent Start/Stop calls

## [2.0.0] - 2025-12-10

### Breaking Changes

- **TelegramUpdate.Message** changed from `json.RawMessage` to `*Message` typed struct
- **TelegramUpdate.CallbackQuery** changed from `json.RawMessage` to `*CallbackQuery` typed struct
- Consumers must remove manual `json.Unmarshal()` calls and access fields directly

### Added

- **Typed Telegram structs** - No manual JSON parsing required
  - `Message` - MessageID, From, Chat, Date, Text, Photo, Document, Caption, ReplyToMessage, Entities, Contact, Location
  - `User` - ID, IsBot, FirstName, LastName, Username, LanguageCode
  - `Chat` - ID, Type, Title, Username, FirstName, LastName
  - `CallbackQuery` - ID, From, Message, InlineMessageID, ChatInstance, Data
  - `MessageEntity` - Type, Offset, Length, URL, User, Language
  - `PhotoSize` - FileID, FileUniqueID, Width, Height, FileSize
  - `Document` - FileID, FileUniqueID, Thumbnail, FileName, MimeType, FileSize
  - `Contact` - PhoneNumber, FirstName, LastName, UserID, VCard
  - `Location` - Longitude, Latitude

- **TelegramUpdate.EditedMessage** field for edited message updates

- **Documentation**
  - `CHANGELOG.md` - Version history
  - `CLAUDE.md` - Claude Code guidance
  - Updated `README.md` with v2 typed structs, usage examples, migration guide

### Changed

- `example/main.go` - Now uses typed structs instead of manual JSON parsing
- `README.md` - Complete rewrite for v2

### Removed

- `json.RawMessage` fields in `TelegramUpdate` (replaced with typed structs)

## [1.x.x] - Previous versions

See git history for changes prior to v2.0.0.

**Note**: v1.x used `json.RawMessage` requiring manual JSON parsing. Please upgrade to v2.0.0.
