# DIF Services

Planned services:

| Service | Phase | Purpose |
|---|---|---|
| ingestion | P0 | Parse P0 formats, emit deterministic nodes/edges/source anchors, promote versions atomically. |
| retriever | P0 | Retrieve source-anchored document passages. |
| mcp-server | P0 | Expose `search_docs`; later cross-graph tools. |
| agent-service | P2 | Produce claim-block responses with source refs. |

Service code may start only through the current execution queue in `action_plan.md` / `prompts.md`. P0 service work must keep MCP thin, preserve source anchors, enforce corpus admission, audit/meter tool calls, and avoid P1 federation features until the real RIF compatibility resolver passes service-level tests.
