# perfect-cat-streaming

Cat-video HLS streaming demo for New Relic: Go backend (`net/http`+chi) on ECS
Fargate with `ffmpeg` transcoding, MySQL on RDS, S3 + CloudFront for media, React
+ Vite frontend, all alerting/infra in `infra/terraform`.

## New Relic observability guidance

These are the same Kiro "Power" steering files (just Markdown) — reused here so
NRQL/observability work is accurate. The project-specific facts are imported
always; read the generic guides on demand when writing NRQL.

@.kiro/steering/perfect-cat-newrelic.md

When writing NRQL or investigating telemetry, also consult:

- `.kiro/steering/nrql-guide.md` — NRQL syntax, event types, clauses, anti-patterns
- `.kiro/steering/query-patterns.md` — golden signals / logs / tracing / infra patterns
- `.kiro/steering/troubleshooting-workflows.md` — step-by-step investigations

## Live New Relic queries (MCP)

Kiro reaches New Relic via `.kiro/settings/mcp.json`. Claude Code uses a separate
MCP config — to query live data here, register the same server, e.g. project
`.mcp.json`:

```json
{
  "mcpServers": {
    "newrelic": {
      "url": "https://mcp.newrelic.com/mcp/",
      "headers": { "api-key": "${NEW_RELIC_USER_API_KEY}" }
    }
  }
}
```

Use an env var (don't inline the key); account id is `6729598` (US).
