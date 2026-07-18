# Changelog

All notable changes follow Keep a Changelog and Semantic Versioning.

## [Unreleased]

### Added

- Compatibility with additive Server contract `0.20.0`; the personal
  terminal-job event feed is Session-only and adds no MCP tool, resource,
  prompt, scope, subscription, or Agent authority. Test, vet, and build pass;
  the matching `.20260718.5` binary is deployed on isolated 10443.
- Compatibility with additive Server contract `0.19.0`; the deployment System
  Settings projection remains session-admin/Console-facing and adds no MCP
  tool, resource, prompt, scope, or Agent authority. The matching binary is
  deployed as `.20260718.4` on the isolated 10443 host.
- Compatibility with additive Server contract `0.18.0`; personal device pairing
  remains a session-only Console/Android workflow and adds no MCP tool, scope,
  or Agent authority. The matching MCP binary is deployed as `.20260718.3` on
  the isolated 10443 host.
- Compatibility with additive Server contract `0.17.0`; the job-retry action
  remains outside the MCP surface and grants no new Agent authority.
- Compatibility with additive Server contract `0.16.0`; the MCP surface remains
  intentionally unchanged and continues to consume only its existing public API
  operations while failing closed on a mismatched contract.
- Remote deployment tests now use the operating-system trust store by default;
  a custom CA file remains an explicit option for private test deployments.
- Compatibility with additive Server contract `0.15.0`; personal password
  changes remain session-only and do not expand MCP API-key scopes, tools,
  resources, or destructive authority.
- Compatibility with additive Server contract `0.14.0`; workspace profile and
  membership administration remain Owner/Console-facing and do not expand MCP
  scopes, tools, resources, or destructive authority.
- Compatibility with the additive Server contract `0.13.0`; the new
  administration inventories remain Console-facing and do not expand MCP
  scopes, tools, resources, or destructive authority.
- Deployed `0.1.0-dev+phase6.20260717.4` against Server contract `0.13.0`. The
  official Go SDK strict-CA smoke passes unauthenticated `401`, all 21 tools,
  capability discovery, and asset listing without side effects; the deployed
  binary SHA-256 is
  `2508bb3d416488a7d3d0b5fddea9c2805b6509f29b15954407ad8c54d6410a8a`.

- Compatibility with the additive Server contract `0.12.0`; Owner-only asset
  purge remains a Server/Console administrative workflow and does not expand
  MCP scopes or expose a destructive Agent tool.
- Compatibility with the additive Server contract `0.11.0`; MCP behavior and
  its public-API-only authorization boundary are unchanged by waveform delivery.
- Structured full-text asset-search results with ASR Provider and Speaker
  filters plus bounded latest-Revision Segment identifiers and timecodes. Unit
  tests and the official-SDK live workflow verify the public API mapping.
- Reproducible release scripts for six Linux/Windows/macOS AMD64/ARM64 archives,
  deterministic SHA-256 manifests, safe package/contract/target/version checks,
  and a required-CycloneDX mode for the Tag workflow. Two complete local builds
  produced identical archive hashes, and Tag builds now inject the release
  version reported by `--version` instead of retaining `dev`.
- A side-effect-free strict-CA remote smoke test for unauthenticated `401`,
  official-SDK connection, all 21 tools, capability discovery, and asset listing.
- A Tag-triggered draft release pipeline for Linux, Windows, and macOS AMD64/
  ARM64 archives with SHA-256 checksums and a CycloneDX SBOM.
- Twelve read tools, five resource templates, and six prompt workflows backed
  only by the public Server API, with transcript prompt-injection isolation.
- An explicit fail-closed write gate exposing nine typed tools for jobs,
  metadata, tags, annotations, approval, bounded clips, and transcript exports.
- Compact one-hour clip/export download metadata without embedded audio or
  large Base64 responses.
- Initial stdio and Streamable HTTP server foundation.
- Public API-backed `get_system_capabilities` tool.
- Public API-backed asset list/search, asset metadata, specified transcript,
  transcript lineage, and exact time-range segment tools.
- Public API-backed collection, tag, annotation, and bounded processing-status
  tools with stable cursors and integer-millisecond citations.
- Safe bounded REST responses, cancellation propagation, opaque pagination,
  scope-denial tests, read-only tool annotations, and per-IP HTTP rate limits.
- Real stdio and Streamable HTTP integration tests using the official Go SDK.
- Opt-in strict-CA remote Streamable HTTP coverage for unauthenticated rejection,
  discovery of all 21 tools, asset/revision/range reads, real clip and WebVTT
  creation, authenticated download metadata, SHA-256, and byte-range verification.
- Fail-closed contract validation, Origin/body/session protections, bearer
  authentication, and mandatory TLS for non-loopback HTTP exposure.
- Vulnerability, license, secret, and SBOM supply-chain checks.

### Changed

- Upgraded the existing indirect `golang.org/x/sys` dependency from `0.41.0`
  to `0.44.0`, which contains the fix for `GO-2026-5024` on Windows. The
  CI-pinned `govulncheck v1.6.0` now reports no reachable or required-module
  vulnerabilities in the current MCP tree.
- Deployed `0.1.0-dev+phase6.20260717.2` against Server contract `0.11.0`.
  The official SDK again passes strict-CA unauthenticated denial, 21-tool
  discovery, capability reads, and asset listing; waveform delivery remains a
  Server/Console concern and does not broaden MCP scopes.
- Deployed `0.1.0-dev+phase6.20260717.1` against Server contract `0.10.0`.
  The official SDK passes strict-CA Streamable HTTP discovery/read checks and a
  direct public-API workflow that searches latest-Revision transcript text and
  verifies the returned Segment identity and exact timecode.
- Advanced fail-closed Server compatibility to OpenAPI contract `0.10.0`; the
  21-tool surface is unchanged, while `search_assets` now exposes transcript
  hits suitable for direct Agent citation.
- Advanced fail-closed Server compatibility to OpenAPI contract `0.9.0` and
  deployed `0.1.0-dev+phase5.20260717.2`. The additive asset filters and
  lifecycle endpoints do not change the 21-tool surface; strict-CA official-SDK
  discovery, unauthenticated `401`, capability reads, and Agent-attributed asset
  listing pass against the isolated deployment.
- Deployed `0.1.0-dev+phase5.20260717.1` against Server contract `0.8.0`; the
  strict-CA read smoke passed and persisted an Agent `asset.listed` audit without
  creating artifacts. The public Caddy and independent gateway PIDs were unchanged.
- Advanced fail-closed Server compatibility to OpenAPI contract `0.8.0`; the
  additive browser refresh and personal-session APIs do not alter MCP tools.
- Deployed the write-enabled service behind the isolated `10443` gateway with
  a six-scope Agent key; read-only denial, audit attribution, key rotation, and
  post-revocation `401` behavior are verified.
- Advanced fail-closed Server compatibility to OpenAPI contract `0.7.0` for
  additive organization read models.
- Advanced fail-closed Server compatibility to OpenAPI contract `0.6.0` so the
  remote service can authenticate with a durable, scoped Agent credential.
- Advanced fail-closed Server compatibility to OpenAPI contract `0.5.0` and
  added a test that keeps the compiled constant aligned with the repository pin.
