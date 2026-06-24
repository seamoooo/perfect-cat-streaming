# New Relic Observability Steering Guide

Comprehensive guidance for querying New Relic observability data using NRQL through the MCP server.

## Steering Files

| File | Description |
|------|-------------|
| [nrql-guide.md](nrql-guide.md) | Comprehensive NRQL guide — syntax, event types, clauses, advanced patterns, and anti-patterns |
| [query-patterns.md](query-patterns.md) | Query patterns — golden signals, logs, tracing, transactions, and infrastructure |
| [troubleshooting-workflows.md](troubleshooting-workflows.md) | Troubleshooting workflows — step-by-step investigations for errors, performance, dependencies, and alerts |

---

## When to Use the New Relic MCP Server

Use New Relic MCP tools when you need to:
- **Investigate production issues**: Errors, performance degradation, service outages
- **Analyze application performance**: Transaction latency, throughput, error rates
- **Monitor infrastructure**: Host metrics, process data, container performance
- **Track distributed traces**: Request flows across microservices
- **Query logs**: Application and infrastructure log analysis
- **Evaluate golden signals**: Latency, traffic, errors, and saturation
- **Analyze service dependencies**: Understand service relationships
- **Monitor alerts**: Track alert violations and incidents
- **Optimize database performance**: Query slow database operations
- **Custom event analysis**: Query your custom telemetry data
