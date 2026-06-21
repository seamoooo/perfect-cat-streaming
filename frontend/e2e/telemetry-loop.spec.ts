/**
 * Browser-driven NR telemetry load generator.
 *
 * Runs Chromium against the React app to exercise:
 *   - Browser Agent  → PageView, AJAX, SPA route change, JS errors
 *   - Video Agent    → CONTENT_REQUEST / CONTENT_START / CONTENT_BUFFER_*  /
 *                      CONTENT_RENDITION_CHANGE / CONTENT_END
 *   - Backend Go agent → all of the above HTTP transactions, MySQL queries,
 *                        transcoder.job background span
 *
 * Each iteration: upload via UI → wait until status=ready → open detail →
 * actually play `<video>` for PLAYBACK_MS → exercise the player chrome
 * (10s skip, rate change) → back to gallery → DELETE via API.
 *
 * Run from the host:  make telemetry-browser-loop
 * Or inline:          docker compose run --rm playwright
 */
import { test, expect, type Page } from "@playwright/test";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const ITERS = parseInt(process.env.ITERS ?? "5", 10);
const PLAYBACK_MS = parseInt(process.env.PLAYBACK_MS ?? "8000", 10);
const PAUSE_MS = parseInt(process.env.PAUSE_MS ?? "1000", 10);
const API_BASE = (process.env.API_BASE ?? "http://backend:8080").replace(
  /\/$/,
  "",
);
const FIXTURE =
  process.env.SAMPLE_MP4 ??
  path.resolve(__dirname, "../src/__tests__/fixtures/sample.mp4");

const CATS = [
  { catName: "Bincho", breed: "siamese" },
  { catName: "Kanpachi", breed: "bengal" },
  { catName: "Mochi", breed: "siamese" },
  { catName: "Leo", breed: "bengal" },
] as const;

const sleep = (ms: number) => new Promise<void>((r) => setTimeout(r, ms));

async function uploadInUI(
  page: Page,
  opts: { title: string; catName: string; breed: string },
): Promise<string> {
  await page.goto("/upload");

  // Upload form (dedicated upload page). There are two file inputs now (video
  // + optional thumbnail); target the video one specifically.
  await page.setInputFiles('input[accept*="video"]', FIXTURE);
  // The title input is the only blank text input on the page initially —
  // be specific via placeholder to avoid the cat-name/tag inputs.
  await page.locator('input[placeholder*="タイトル"]').fill(opts.title);
  await page.locator('input[placeholder*="猫の名前"]').fill(opts.catName);
  await page.locator("select").selectOption(opts.breed);
  await page.locator('input[placeholder*="タグ"]').fill(`playwright,e2e`);

  const postPromise = page.waitForResponse(
    (r) => r.url().includes("/api/videos") && r.request().method() === "POST",
    { timeout: 60_000 },
  );
  await page.locator('button[type="submit"]:has-text("アップロード")').click();
  const resp = await postPromise;
  const body = (await resp.json()) as { id: string };
  // UploadPage redirects to the list once the POST succeeds.
  await page.waitForURL((u) => u.pathname === "/videos", { timeout: 15_000 });
  return body.id;
}

async function waitForReadyOnCard(page: Page, title: string) {
  // The gallery hook polls /api/videos every 5s, so the status text updates
  // automatically. We find the link element by its inner title text and
  // assert "再生可" appears inside.
  const card = page.locator(`a:has(h3:text-is("${title}"))`).first();
  await expect(card).toBeVisible({ timeout: 30_000 });
  await expect(card.locator("text=再生可")).toBeVisible({ timeout: 120_000 });
  return card;
}

async function playForMs(page: Page, ms: number) {
  // Wait for <video> to attach
  const video = page.locator("video");
  await video.waitFor({ state: "attached", timeout: 30_000 });

  // Force play (autoplay flag in launchOptions usually makes this unnecessary
  // but doesn't hurt).
  await page.evaluate(async () => {
    const v = document.querySelector<HTMLVideoElement>("video");
    if (v) {
      try {
        await v.play();
      } catch {
        /* ignore — autoplay race */
      }
    }
  });

  // Wait until enough wall-clock playback has elapsed (currentTime advances
  // even if user can't see the frames). For very short fixture loops the
  // video may end before reaching the threshold — clamp via "ended" event.
  // Also short-circuit if the simulated player-chaos overlay appears: the hiss
  // error telemetry already fired, so we let the iteration continue instead of
  // hanging until timeout.
  await page.waitForFunction(
    (ms: number) => {
      if (document.querySelector('[role="alert"]')) return true; // chaos overlay
      const v = document.querySelector<HTMLVideoElement>("video");
      if (!v) return false;
      return v.ended || v.currentTime * 1000 >= ms;
    },
    ms,
    { timeout: ms * 4 + 15_000 },
  );
}

test.describe.configure({ mode: "serial" });

test(`NR browser+video telemetry loop x${ITERS}`, async ({ page, request }) => {
  test.setTimeout(ITERS * 6 * 60_000 + 60_000);

  // Surface page console errors so a broken build is obvious.
  page.on("pageerror", (err) => console.error("[pageerror]", err.message));
  page.on("console", (msg) => {
    const t = msg.type();
    if (t === "error" || t === "warning") {
      console.log(`[console.${t}] ${msg.text()}`);
    }
  });

  for (let i = 0; i < ITERS; i++) {
    const cat = CATS[i % CATS.length];
    const title = `[pw-${i}] ${cat.catName} ${new Date().toISOString().slice(11, 19)}`;
    const cycleStart = Date.now();

    console.log(`[iter ${i}] ▶ uploading "${title}"`);
    const videoId = await uploadInUI(page, {
      title,
      catName: cat.catName,
      breed: cat.breed,
    });
    console.log(`[iter ${i}]   id=${videoId}`);

    await waitForReadyOnCard(page, title);
    const readyAt = Date.now();
    console.log(
      `[iter ${i}]   ready in ${((readyAt - cycleStart) / 1000).toFixed(1)}s`,
    );

    // Click into detail → SPA route change → KanpachiPlayer mounts →
    // Video Agent attaches Html5Tracker to <video> → CONTENT_REQUEST fires
    const card = page.locator(`a:has(h3:text-is("${title}"))`).first();
    await card.click();
    await page.waitForURL(/\/clips\/[\w-]+/, { timeout: 15_000 });
    console.log(`[iter ${i}]   navigated to detail`);

    await playForMs(page, PLAYBACK_MS);
    console.log(`[iter ${i}]   played ${PLAYBACK_MS / 1000}s`);

    // Exercise the player chrome → more events:
    //   - 10s skip → seek → CONTENT_SEEK_START / CONTENT_SEEK_END
    //   - rate change → CONTENT_RATE_CHANGE
    if (i % 2 === 0) {
      await page
        .locator('button[title="10秒進む"]')
        .click()
        .catch(() => {});
      await page
        .locator('button:has-text("1.5x")')
        .click()
        .catch(() => {});
      await sleep(1500);
    }

    // Back to the list
    await page
      .locator('a:has-text("一覧へ戻る")')
      .click()
      .catch(() => {});
    await page.waitForURL((u) => u.pathname === "/videos", { timeout: 10_000 });

    // Cleanup via direct API (UI has no delete; keeps the gallery tidy across
    // iterations).
    const del = await request.delete(`${API_BASE}/api/videos/${videoId}`);
    if (!del.ok() && del.status() !== 204) {
      console.warn(`[iter ${i}]   delete failed status=${del.status()}`);
    } else {
      console.log(
        `[iter ${i}] ✔ deleted total=${((Date.now() - cycleStart) / 1000).toFixed(1)}s`,
      );
    }

    if (i < ITERS - 1) await sleep(PAUSE_MS);
  }
});
