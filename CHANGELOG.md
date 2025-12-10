# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
