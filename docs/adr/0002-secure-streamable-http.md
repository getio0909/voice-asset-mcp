# ADR 0002: Fail closed at the Streamable HTTP boundary

- Status: Accepted
- Date: 2026-07-15

## Context

Streamable HTTP can expose bearer credentials and long-lived sessions to remote
networks and browsers. Default SDK options do not enable cross-origin checks or
idle session expiry.

## Decision

Limit MCP request bodies to 2 MiB, reject unsafe cross-origin browser requests,
expire idle sessions after 15 minutes, and retain the SDK's localhost rebinding
protection. Require an inbound bearer token and native TLS certificate/key when
binding to a non-loopback address. Require HTTPS for a non-loopback VoiceAsset
Server URL. Accept only Server API `v1` contract `0.22.0` and fail closed on
malformed capability declarations.

## Consequences

Local stdio and loopback HTTP remain simple. Remote deployments must provision
TLS material and rotate bearer tokens. Contract upgrades require an explicit,
tested compatibility change before new Server responses are accepted.
