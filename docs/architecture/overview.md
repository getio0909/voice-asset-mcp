# Architecture

```text
Agent -> MCP transport -> typed tool -> REST client -> VoiceAsset Server API
```

The process supports stdio and Streamable HTTP. MCP handlers translate semantic
requests into versioned public REST calls and return compact structured output.
The Server remains authoritative for authentication, authorization, business
rules, storage, and auditing. The MCP process never opens database connections.

Asset searches use the Server's opaque cursor. Transcript tools request an
immutable revision through REST and return segment citations with asset,
revision, segment, and half-open millisecond boundaries. The outbound Server
token determines scope; MCP does not elevate it or hold provider credentials.
Collection, tag, annotation, and processing-status tools use the same public
REST boundary and preserve workspace scope, opaque pagination, and integer
millisecond annotation ranges.

Write tools are registered only when an operator explicitly enables them. They
still use the same scoped Server token and public REST contract; MCP does not
reimplement optimistic concurrency, workspace validation, or auditing. Audio
clips and transcript exports return expiring authenticated URLs and bounded
metadata instead of bytes. Resource templates reuse the read client, and
prompts wrap fetched transcript text in explicit untrusted-data boundaries.
