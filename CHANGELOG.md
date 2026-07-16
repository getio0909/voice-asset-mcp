# Changelog

All notable changes follow Keep a Changelog and Semantic Versioning.

## [Unreleased]

### Added

- Initial stdio and Streamable HTTP server foundation.
- Public API-backed `get_system_capabilities` tool.
- Real stdio and Streamable HTTP integration tests using the official Go SDK.
- Fail-closed contract validation, Origin/body/session protections, bearer
  authentication, and mandatory TLS for non-loopback HTTP exposure.
- Vulnerability, license, secret, and SBOM supply-chain checks.
