/**
 * NR telemetry load generator — drives the backend through full lifecycle
 * cycles (upload → wait-for-transcode → "play" → tag → delete) so the
 * backend's New Relic APM has a steady stream of transactions, MySQL
 * queries, and (in S3 mode) S3 calls to render. Vitest uses Jest-compatible
 * APIs so this file works as a Jest spec too.
 *
 * Disabled by default — gated on LOAD=1 so plain `npm test` stays fast.
 *
 *   # From host (default API_BASE = http://localhost:8080)
 *   LOAD=1 ITERS=20 npm test --prefix frontend -- telemetry-loop
 *
 *   # From inside the frontend container
 *   docker compose exec -e LOAD=1 -e ITERS=20 \
 *     -e API_BASE=http://backend:8080 \
 *     frontend npx vitest run telemetry-loop
 *
 *   # Generate the MP4 fixture once (uses ffmpeg in the backend container)
 *   make telemetry-fixture
 */
import { describe, it, expect } from "vitest";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const RUN = process.env.LOAD === "1";
const API = (process.env.API_BASE ?? "http://localhost:8080").replace(/\/$/, "");
const ITERS = parseInt(process.env.ITERS ?? "5", 10);
const PAUSE_MS = parseInt(process.env.PAUSE_MS ?? "1500", 10);
const READY_TIMEOUT_MS = parseInt(process.env.READY_TIMEOUT_MS ?? "180000", 10);
const FIXTURE =
  process.env.SAMPLE_MP4 ?? path.resolve(__dirname, "fixtures/sample.mp4");

const CAT_PRESETS = [
  { catName: "Bincho", breed: "siamese" as const },
  { catName: "Kanpachi", breed: "bengal" as const },
  { catName: "Mochi", breed: "siamese" as const },
  { catName: "Leo", breed: "bengal" as const },
];

interface Video {
  id: string;
  title: string;
  status: "pending" | "processing" | "ready" | "error";
  durationSec: number;
  playlistUrl?: string;
  errorMsg?: string;
  tags?: string[];
}

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

// Resolve whatever the backend returned (relative "/media/..." or absolute
// "http://localhost:8080/...") against the test's API_BASE so vitest can fetch
// it from any host.
function rewriteToApiBase(maybeRelativeUrl: string): string {
  if (maybeRelativeUrl.startsWith("/")) {
    return API + maybeRelativeUrl;
  }
  try {
    const u = new URL(maybeRelativeUrl);
    const target = new URL(API);
    u.protocol = target.protocol;
    u.host = target.host;
    return u.toString();
  } catch {
    return maybeRelativeUrl;
  }
}

async function uploadVideo(opts: {
  filepath: string;
  title: string;
  description?: string;
  catName: string;
  breed: "siamese" | "bengal" | "other";
  tags?: string[];
}): Promise<Video> {
  const buf = fs.readFileSync(opts.filepath);
  const fd = new FormData();
  fd.append(
    "file",
    new Blob([buf], { type: "video/mp4" }),
    path.basename(opts.filepath),
  );
  fd.append("title", opts.title);
  if (opts.description) fd.append("description", opts.description);
  fd.append("catName", opts.catName);
  fd.append("breed", opts.breed);
  if (opts.tags?.length) fd.append("tags", opts.tags.join(","));
  const res = await fetch(`${API}/api/videos`, { method: "POST", body: fd });
  if (!res.ok) {
    throw new Error(`upload failed ${res.status}: ${await res.text()}`);
  }
  return (await res.json()) as Video;
}

async function getVideo(id: string): Promise<Video> {
  const res = await fetch(`${API}/api/videos/${id}`);
  if (!res.ok) throw new Error(`get failed ${res.status}`);
  return (await res.json()) as Video;
}

async function pollUntilReady(id: string, timeoutMs: number): Promise<Video> {
  const start = Date.now();
  let last: Video | null = null;
  while (Date.now() - start < timeoutMs) {
    last = await getVideo(id);
    if (last.status === "ready") return last;
    if (last.status === "error") {
      throw new Error(`transcode failed id=${id}: ${last.errorMsg ?? "?"}`);
    }
    await sleep(1000);
  }
  throw new Error(
    `poll timed out after ${timeoutMs}ms id=${id} last=${last?.status}`,
  );
}

async function playFully(playlistUrl: string) {
  const url = rewriteToApiBase(playlistUrl);
  const playRes = await fetch(url);
  if (!playRes.ok) throw new Error(`playlist ${playRes.status}`);
  const text = await playRes.text();
  const segs = text
    .split("\n")
    .map((l) => l.trim())
    .filter((l) => l && !l.startsWith("#"));
  const base = url.substring(0, url.lastIndexOf("/") + 1);
  for (const seg of segs) {
    const segRes = await fetch(base + seg);
    if (!segRes.ok) throw new Error(`segment ${seg} ${segRes.status}`);
    await segRes.arrayBuffer(); // drain
  }
  return segs.length;
}

async function updateTags(id: string, tags: string[]): Promise<Video> {
  const res = await fetch(`${API}/api/videos/${id}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ tags }),
  });
  if (!res.ok) throw new Error(`patch failed ${res.status}`);
  return (await res.json()) as Video;
}

async function deleteVideo(id: string): Promise<void> {
  const res = await fetch(`${API}/api/videos/${id}`, { method: "DELETE" });
  if (!res.ok && res.status !== 204) {
    throw new Error(`delete failed ${res.status}: ${await res.text()}`);
  }
}

async function listVideos(): Promise<Video[]> {
  const res = await fetch(`${API}/api/videos`);
  if (!res.ok) throw new Error(`list failed ${res.status}`);
  return (await res.json()) as Video[];
}

function logf(i: number, msg: string, extra?: Record<string, unknown>) {
  const tail = extra
    ? " " +
      Object.entries(extra)
        .map(([k, v]) => `${k}=${String(v)}`)
        .join(" ")
    : "";
  console.log(`[loop ${String(i).padStart(3)}] ${msg}${tail}`);
}

// One full cycle. Each step intentionally hits the API so the backend records
// a distinct transaction in NR.
async function runCycle(i: number, fixturePath: string) {
  const preset = CAT_PRESETS[i % CAT_PRESETS.length];
  const cycleStart = Date.now();

  await listVideos(); // GET /api/videos
  logf(i, "listed");

  const initial = await uploadVideo({
    filepath: fixturePath,
    title: `[loop-${i}] ${preset.catName} clip ${new Date().toISOString()}`,
    description: `auto-generated by telemetry-loop iter=${i}`,
    catName: preset.catName,
    breed: preset.breed,
    tags: ["telemetry-loop", `iter-${i}`, preset.breed],
  });
  logf(i, "uploaded", { id: initial.id });

  const ready = await pollUntilReady(initial.id, READY_TIMEOUT_MS);
  logf(i, "ready", { dur: ready.durationSec, took_s: ((Date.now() - cycleStart) / 1000).toFixed(1) });

  if (!ready.playlistUrl) throw new Error("ready video missing playlistUrl");
  const segCount = await playFully(ready.playlistUrl);
  logf(i, "played", { segments: segCount });

  // GET detail again — emulates a user inspecting after playback
  await getVideo(initial.id);

  await updateTags(initial.id, ["telemetry-loop", `done-${i}`, "auto-tagged", "cat"]);
  logf(i, "tagged");

  await deleteVideo(initial.id);
  logf(i, "deleted", { total_s: ((Date.now() - cycleStart) / 1000).toFixed(1) });
}

describe.skipIf(!RUN)("NR telemetry loop", () => {
  it(
    `upload → play → tag → delete x${ITERS} (API=${API})`,
    async () => {
      if (!fs.existsSync(FIXTURE)) {
        throw new Error(
          `Sample MP4 not found at ${FIXTURE}. Generate it once with: make telemetry-fixture`,
        );
      }
      const fixtureSize = fs.statSync(FIXTURE).size;
      console.log(
        `[loop] starting iters=${ITERS} pause=${PAUSE_MS}ms fixture=${FIXTURE} (${(fixtureSize / 1024).toFixed(1)} KB)`,
      );

      for (let i = 0; i < ITERS; i++) {
        try {
          await runCycle(i, FIXTURE);
        } catch (err) {
          // Don't stop on transient failures — telemetry collection is the goal.
          console.error(`[loop ${i}] cycle failed:`, err);
        }
        if (i < ITERS - 1) await sleep(PAUSE_MS);
      }
      expect(true).toBe(true);
    },
    ITERS * 240_000 + 60_000, // generous: each cycle can take a few minutes when transcoding
  );
});
