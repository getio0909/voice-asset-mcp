# ADR 0004: Gate typed write tools explicitly

- Status: Accepted
- Date: 2026-07-16

## Context

MCP clients can invoke tools autonomously. Exposing mutations by default, or
returning large media inline, would make a read-oriented installation more
powerful than its operator intended.

## Decision

Keep the default server limited to twelve read tools. Register nine typed write
tools only when `VOICE_ASSET_MCP_ENABLE_WRITES=true` or `--enable-writes` is
provided. Every write calls a versioned public REST endpoint with the configured
Server API key; Server scopes, workspace rules, concurrency checks, and audits
remain authoritative. Tool annotations declare writes and their destructive or
idempotent hints.

Represent created clips and transcript exports as compact IDs, time ranges,
hashes, and one-hour authenticated download URLs. Never embed audio Base64 or
expose storage keys. Validate UUIDs, ranges, formats, and retry keys before a
request leaves MCP.

## Consequences

Read deployments do not gain mutations during an upgrade. Operators enabling
writes must provision only the required Server scopes and protect the inbound
MCP bearer. Agents receive portable artifacts without bypassing the Server's
authorization or audit boundary.
