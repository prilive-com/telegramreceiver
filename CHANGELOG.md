# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
