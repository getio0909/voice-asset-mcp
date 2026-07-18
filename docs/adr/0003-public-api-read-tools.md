# ADR 0003: Public-API Read Tools and Exact Citations

- Status: Accepted
- Date: 2026-07-16

## Context

Phase 4 requires useful Agent semantics without turning MCP into a database
adapter or duplicating Server authorization and transcript rules.

## Decision

Use official MCP Go SDK `v1.6.1`, the current stable release supporting the
2025-11-25 specification. Register typed, structured, read-only tools for asset
pagination/search, asset metadata, specified immutable revisions, parent
lineages, exact time-range segments, collections, tags, asset annotations,
bounded processing status, and capability negotiation.

Every tool calls Server contract `0.22.0` with the configured bearer token.
Server scopes and audit logs are authoritative. REST bodies are bounded to
1 MiB, non-2xx bodies are discarded, cancellation propagates through request
contexts, and remote MCP requests have per-IP fixed-window limits. Segment
ranges use `[start_ms, end_ms)` and return full citation identifiers plus exact
overlap boundaries. Asset search returns at most five chronological hits from
each matching asset's latest immutable Revisions with the same identity and
millisecond timecodes.

## Consequences

- The MCP binary contains no PostgreSQL driver or provider credential.
- Tool results are compact JSON rather than audio Base64.
- `list_transcript_revisions` follows the latest public parent lineage; a future
  paginated Server history endpoint may supersede traversal when branches exist.
- The remote service uses a durable, revocable API key with only the read
  scopes required by its tools. Write tools, Resources, Prompts, and audio
  clips remain explicit later slices.
