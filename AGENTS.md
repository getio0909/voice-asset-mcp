# Repository Guidelines

## Architecture

Keep MCP registration in `internal/mcpserver/`, REST access in
`internal/backend/`, and executable wiring in `cmd/voice-asset-mcp/`. Never
query the VoiceAsset database or copy Server domain logic into this repository.

## Commands

- `make verify`: vet, test with coverage, and build.
- `make test-race`: run the race detector (requires a working C toolchain).
- `go test ./...`: run the smallest local test suite.
- `go run ./cmd/voice-asset-mcp --transport=stdio`: run the stdio server.

Use `gofmt`, table-driven tests, lowercase package names, and typed MCP inputs
and outputs. Keep stdout protocol-clean in stdio mode. Use Conventional Commits,
for example `feat(tools): add asset search`. Update tests and the documented
Server contract version whenever a tool schema changes.
