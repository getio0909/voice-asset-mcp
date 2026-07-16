# VoiceAsset MCP

The Agent-facing semantic layer for VoiceAsset. It uses the official MCP Go SDK
and calls only the public VoiceAsset Server REST API; it never reads PostgreSQL
or object storage directly.

## Run locally

Requirements: Go 1.26.5 or newer in the 1.26 release line. Copy `.env.example` values into your environment;
do not commit the resulting credentials.

```bash
go run ./cmd/voice-asset-mcp --transport=stdio
go run ./cmd/voice-asset-mcp --transport=http --listen=127.0.0.1:8090
```

The Streamable HTTP endpoint is `/mcp`; liveness is `/health/live`. A bearer
token plus native TLS certificate/key files are mandatory when binding HTTP to
a non-loopback address. The bearer token does not make plain HTTP safe.
Connections to a non-loopback VoiceAsset Server must also use HTTPS.

## Validate

```bash
make verify
```

The current foundation exposes `get_system_capabilities`. Additional tools,
resources, prompts, scopes, and audit behavior are tracked for later vertical
slices and must be backed by the public Server contract.

This revision supports REST API namespace `v1` and OpenAPI contract `0.1.0`,
recorded in `CONTRACT_VERSION`. Capability negotiation fails closed for other
versions or malformed feature declarations.

## License

AGPL-3.0-or-later. See `LICENSE`.
