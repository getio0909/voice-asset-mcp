# ADR 0001: Use the official MCP Go SDK

- Status: Accepted
- Date: 2026-07-15

## Decision

Use `github.com/modelcontextprotocol/go-sdk` v1.6.1 for typed tools, stdio, and
Streamable HTTP. Access VoiceAsset exclusively through the public REST API.

## Consequences

Protocol behavior follows the maintained SDK and supports cancellation through
Go contexts. SDK updates require protocol compatibility tests and changelog
entries. Direct database access and duplicated Server business rules are
architecturally prohibited.
