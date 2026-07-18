# VoiceAsset MCP

The Agent-facing semantic layer for VoiceAsset. It uses the official MCP Go SDK
and calls only the public VoiceAsset Server REST API; it never reads PostgreSQL
or object storage directly.

## Run locally

Requirements: Go 1.26.5 or newer in the 1.26 release line. Copy `.env.example`
values into your environment; do not commit the resulting credentials.

```bash
go run ./cmd/voice-asset-mcp --transport=stdio
go run ./cmd/voice-asset-mcp --transport=http --listen=127.0.0.1:8090
# Explicitly expose state-changing tools only with a suitably scoped Server key:
go run ./cmd/voice-asset-mcp --transport=stdio --enable-writes
```

The Streamable HTTP endpoint is `/mcp`; liveness is `/health/live`. A bearer
token plus native TLS certificate/key files are mandatory when binding HTTP to
a non-loopback address. The bearer token does not make plain HTTP safe.
Connections to a non-loopback VoiceAsset Server must also use HTTPS.

## Validate

```bash
make verify
```

Build and verify the same deterministic six-platform archives used by the Tag
workflow. The output directory must be new or empty:

```bash
version=v1.0.0-rc.1
mkdir dist
bash scripts/build-release.sh "$version" dist
bash scripts/write-checksums.sh dist
bash scripts/verify-release.sh "$version" dist
```

The verifier checks safe package layout, contract pins, target metadata,
embedded and host runtime versions, and complete SHA-256 coverage. CI adds the
CycloneDX JSON SBOM and repeats verification with `--require-sbom` before draft
release upload. `voice-asset-mcp --version` is safe to run without configuration.

The side-effect-free strict-TLS deployment smoke uses
`VOICE_ASSET_MCP_REMOTE_READ_E2E=1`, `VOICE_ASSET_MCP_REMOTE_URL`,
and `VOICE_ASSET_MCP_HTTP_TOKEN`; it uses the operating-system trust store by
default. `VOICE_ASSET_MCP_CA_FILE` is optional for a private test CA. The fuller artifact
workflow uses `VOICE_ASSET_MCP_REMOTE_E2E=1` plus
`VOICE_ASSET_SERVER_REMOTE_TOKEN`. Supply tokens only through the test process;
do not write them into the repository.

The default read surface exposes:

- `list_assets` and PostgreSQL-backed `search_assets` with title/latest-Transcript
  terms, Collection/Tag/status/date/Provider/Speaker filters, opaque cursors,
  and bounded immutable Segment hits with timecodes
- `get_asset` and `get_asset_metadata`
- `list_collections`, `list_tags`, `get_annotations`, and
  `get_processing_status` with bounded organization read models
- `get_transcript`, `list_transcript_revisions`, and
  `get_transcript_segments` with exact half-open millisecond ranges
- `get_system_capabilities`

Five resource templates expose assets, latest/specified transcripts,
collections, and jobs. Six prompts cover summaries, action items, technical
terms, revision comparison, meeting minutes, and ASR-quality review; transcript
text is always marked as untrusted input.

Set `VOICE_ASSET_MCP_ENABLE_WRITES=true` or pass `--enable-writes` to add:

- transcription and LLM-correction job creation
- optimistic asset metadata updates and tag/annotation mutations
- transcript approval
- bounded audio-clip creation and JSON/Markdown/SRT/WebVTT export

Clip and export tools return compact metadata plus one-hour authenticated
download URLs, never embedded audio or large Base64 payloads. Write exposure is
fail-closed by default and still depends on the outbound Server API key scopes.

Every tool calls the public Server API with `VOICE_ASSET_SERVER_TOKEN`; Server
scope checks and immutable read-audit records remain authoritative. Remote MCP
requests are bounded, bearer protected, cross-origin protected, and rate
limited.

This revision supports REST API namespace `v1` and OpenAPI contract `0.22.0`,
recorded in `CONTRACT_VERSION`. Capability negotiation fails closed for other
versions or malformed feature declarations.

## License

AGPL-3.0-or-later. See `LICENSE`.
