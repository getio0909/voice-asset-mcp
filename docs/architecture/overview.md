# Architecture

```text
Agent -> MCP transport -> typed tool -> REST client -> VoiceAsset Server API
```

The process supports stdio and Streamable HTTP. MCP handlers translate semantic
requests into versioned public REST calls and return compact structured output.
The Server remains authoritative for authentication, authorization, business
rules, storage, and auditing. The MCP process never opens database connections.
