---
inclusion: always
---

# perfect-cat-streaming — New Relic project facts

Project-specific observability context to make NRQL/troubleshooting accurate on
THIS repo. Pairs with the generic New Relic steering files in this folder.
Everything below was verified against the live New Relic account.

## Account & entities

- New Relic account ID: **6729598** (US data center).
- Backend APM app name: **`PerfectCatStreaming`** — set via `NEW_RELIC_APP_NAME`,
  and `APP_ENV=prod` so there is **no `(dev)` suffix**. Always filter
  `WHERE appName = 'PerfectCatStreaming'`. Historical data may exist under
  `PerfectCatStreaming (dev)` — ignore it (older deploys).
- The Browser agent and the Video agent report under their **own** entities
  (separate applicationIDs from the APM app).

## Architecture (so queries map to reality)

Go backend (`net/http` + chi) on **ECS Fargate**, `ffmpeg` → HLS, **MySQL on
RDS**, **S3 + CloudFront** for media. Uploads are **async**: `POST /api/videos`
returns `202` immediately, then a background worker queue transcodes.

## Backend transactions & custom attributes

- Web txn names: `WebTransaction/Go/<METHOD> <route>`,
  e.g. `WebTransaction/Go/GET /api/videos/{id}`.
- Background transcode job txn name: **`OtherTransaction/Go/transcoder.job`**.
- Custom attributes on `Transaction`: `cat.breed`, `cat.name`, `video.id`,
  `video.status`, `video.count`, `upload.size_bytes`, `upload.content_type`,
  `transcode.realtime_factor`, `transcode.wall_sec`,
  `transcode.video_duration_sec`, `transcode.segment_count`,
  `transcode.output_bytes`, `transcode.publish_target`, `chaos.injected`.
- Transcode failures = `NoticeError` on the job txn →
  `FROM TransactionError WHERE appName = 'PerfectCatStreaming'
   AND transactionName = 'OtherTransaction/Go/transcoder.job'`
  (e.g. message `ffmpeg failed: exit status 1`, common for phone HEVC/H.265).

## Frontend telemetry

- In-house QoE events (`PageAction`): `kanpachi.first_pounce` (TTFF),
  `kanpachi.hairball_start` / `kanpachi.hairball_end` (rebuffer),
  `kanpachi.hiss` (player error), `kanpachi.stretch` (bitrate change).
  Attributes: `videoId`, `catName`, `breed`, `sessionId`.
- New Relic Video agent emits `CONTENT_*` events incl. `CONTENT_ERROR`:
  `FROM PageAction WHERE actionName = 'CONTENT_ERROR'`.
- Core Web Vitals:
  `FROM PageViewTiming SELECT percentile(largestContentfulPaint, 75)`
  (LCP is in **seconds**; CWV "good" ≤ 2.5).

## ⚠️ Known gotcha — AWS/ECS metrics are NOT in New Relic

The CloudWatch Metric Stream is configured (AWS/RDS, AWS/ECS, ALB, S3, WAFV2)
but **no `aws.*` metrics are actually arriving** —
`FROM Metric SELECT uniques(metricName) WHERE metricName LIKE 'aws%'` is empty.
So **do NOT query `aws.ecs.CPUUtilization` / `aws.ecs.MemoryUtilization`** for the
backend; they don't exist here. Use APM proxies instead:

- Transcode job CPU/memory pressure → `average(\`transcode.realtime_factor\`)`
  (healthy ≈ 1.8; sustained > 5 = starved/degraded).
- CPU-contention symptom → `FROM Transaction WHERE appName='PerfectCatStreaming'
  AND transactionType='Web' AND duration > 10` (normal web p95 ≈ a few ms).

## Chaos / demo signals

- Description-keyword chaos sets `chaos.injected`: `sre_slow_transcode`,
  `backend_500`, `frontend_render_error`; the "player" keyword fires an hls.js
  error tagged `reason = 'chaos'`.
- Older env-gated chaos sets `chaos.injected` = `latency` | `error_500`.

## Alert policies (Terraform-managed in `infra/terraform`)

- **`PerfectCatStreaming alerts`** — transcode throughput, web throughput
  (baseline), Browser CWV LCP p75, Video `CONTENT_ERROR`, Synthetics uptime.
- **`PerfectCatStreaming - upload pipeline`** — transcode error, transcode
  degraded (`realtime_factor > 5`), backend latency spike. Routes to Email +
  Slack + AI agent (Slack/SRE-Agent attached in the NR UI; the workflow has
  `lifecycle.ignore_changes = [destination]`).

## Demo fragility toggles (gitignored `terraform.tfvars`)

`backend_cpu` / `backend_memory`, `transcode_timeout_sec`, `transcode_workers`.
- Normal: `2048 / 4096 / 1800 / 2`.
- Deliberately-broken demo (forces transcode failures → alerts): `256 / 512 / 5 / 4`.
The committed `variables.tf` defaults stay sane; only the gitignored tfvars flips them.
