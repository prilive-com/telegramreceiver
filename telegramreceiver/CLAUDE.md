# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands
- Build: `go build ./...`
- Run: `go run ./example/main.go`
- Test: `go test ./...`
- Test single file: `go test ./path/to/package -run TestFunctionName`
- Lint: `golint ./...`
- Format: `gofmt -w .`

## Code Style Guidelines
- Imports: Group standard library, third-party, and local imports with a blank line between groups
- Formatting: Use gofmt
- Error handling: Always check errors, log with context (`logger.Error("message", "error", err)`)
- Naming: Use camelCase for private and PascalCase for exported names
- Documentation: Document all exported functions, types, and constants
- Testing: Write table-driven tests when appropriate
- Logging: Use structured logging with slog.Logger
- Context: Pass context.Context as the first parameter for long-running operations
- Security: Validate all inputs, use secure defaults, and implement proper TLS