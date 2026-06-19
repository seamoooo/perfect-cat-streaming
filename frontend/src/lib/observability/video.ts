// New Relic Video Agent wrapper for our HLS player.
//
// - attaches an Html5Tracker to the underlying <video> element so the standard
//   New Relic video events (CONTENT_REQUEST / CONTENT_START / CONTENT_END /
//   CONTENT_BUFFER_START|END / CONTENT_RENDITION_CHANGE / CONTENT_ERROR …)
//   are sent to the dedicated NR Video & Streaming entity
// - exposes attachIMAAdsTracker(adsManagerProvider) so that when Google IMA
//   SDK is wired up later the Ads tracker can be plugged in without touching
//   the player component
//
// @newrelic/video-core v4 has its OWN harvester (independent of the Browser
// Agent). All we need is the routing block — beacon + Video applicationID +
// licenseKey — from the "Streaming Video & Ads" install flow. If those env
// vars are missing the tracker is a no-op. The dynamic import keeps the
// video-agent code lazy.

import type Hls from "hls.js";

export interface VideoTrackerHandle {
  /** Tear down the tracker; called by the player on unmount or src change. */
  dispose: () => void;
  /** When the IMA SDK is integrated later, pass its AdsManager (or the
   *  `player.ima` object) here to start ad event tracking. */
  attachIMAAdsTracker: (imaAttachment: unknown) => void;
}

export interface VideoTrackerMeta {
  videoId: string;
  catName: string;
  breed: string;
  title: string;
  tags?: string[];
}

const NOOP_HANDLE: VideoTrackerHandle = {
  dispose: () => {},
  attachIMAAdsTracker: () => {},
};

/**
 * Walk a module object trying each dotted path until we hit a callable —
 * deals with `mod.default`, `mod.default.default`, named exports, etc. so we
 * survive Vite/Rollup CJS interop quirks.
 */
function resolveCtor(mod: Record<string, unknown>, ...paths: string[]): unknown {
  for (const path of paths) {
    let cur: unknown = mod;
    for (const seg of path.split(".")) {
      if (cur && typeof cur === "object" && seg in (cur as object)) {
        cur = (cur as Record<string, unknown>)[seg];
      } else {
        cur = undefined;
        break;
      }
    }
    if (typeof cur === "function") return cur;
  }
  return undefined;
}

/**
 * Start a New Relic Html5 video tracker against the given <video> element.
 * Best-effort: returns a no-op handle if the agent can't load.
 */
// Routing fields the Video Agent needs to send data to the right NR entity.
// They are intentionally separate from the Browser Agent values — NR creates a
// distinct entity for video data.
function videoRoutingInfo() {
  const e = import.meta.env;
  const videoAppID = (e.VITE_NEW_RELIC_VIDEO_APP_ID as string | undefined) ?? "";
  const licenseKey = (e.VITE_NEW_RELIC_LICENSE_KEY as string | undefined) ?? "";
  if (!videoAppID || !licenseKey) return null;
  return {
    beacon: "bam.nr-data.net",
    applicationID: videoAppID,
    licenseKey,
  };
}

export async function attachVideoTracker(
  videoEl: HTMLVideoElement | null,
  _hls: Hls | null,
  meta: VideoTrackerMeta,
): Promise<VideoTrackerHandle> {
  if (!videoEl) return NOOP_HANDLE;
  const routing = videoRoutingInfo();
  if (!routing) {
    console.info(
      "[newrelic-video] disabled (set VITE_NEW_RELIC_VIDEO_APP_ID and VITE_NEW_RELIC_LICENSE_KEY to enable)",
    );
    return NOOP_HANDLE;
  }

  try {
    console.info(
      `[newrelic-video] attaching tracker videoId=${meta.videoId} applicationID=${routing.applicationID} beacon=${routing.beacon}`,
    );
    const html5Mod = (await import("@newrelic/video-html5")) as Record<string, unknown>;
    // Vite's CJS interop sometimes double-wraps the default export:
    //   { default: { default: class Html5Tracker } }
    // Resolve through both shapes so we always end up with the constructor.
    const Html5Tracker = resolveCtor(
      html5Mod,
      "default.default",
      "default",
      "Html5Tracker",
    ) as new (player: HTMLVideoElement, options?: unknown) => unknown;

    if (typeof Html5Tracker !== "function") {
      console.error(
        "[newrelic-video] could not resolve Html5Tracker from @newrelic/video-html5 module:",
        html5Mod,
      );
      return NOOP_HANDLE;
    }

    const trackerOptions = {
      tag: videoEl,
      info: {
        // Routing — required so video data flows to the NR video entity
        beacon: routing.beacon,
        applicationID: routing.applicationID,
        licenseKey: routing.licenseKey,
        // Per-clip custom attributes — surfaced in NRQL alongside CONTENT_* events
        videoId: meta.videoId,
        contentTitle: meta.title,
        contentCustomAttributes: {
          catName: meta.catName,
          breed: meta.breed,
          tags: (meta.tags ?? []).join(","),
        },
      },
    };
    const tracker = new Html5Tracker(videoEl, trackerOptions) as {
      dispose?: () => void;
      setAdsTracker?: (t: unknown) => void;
      on?: (event: string, fn: (e: unknown) => void) => void;
    };

    // Echo every CONTENT_*/AD_* event the tracker emits so we can see at a
    // glance in DevTools whether the agent is firing. (NR's own logger is
    // verbose at debug level; this is a high-signal summary line.)
    tracker.on?.("*", (e: unknown) => {
      const evt = e as { type?: string; actionName?: string; data?: Record<string, unknown> };
      const name = evt?.actionName ?? evt?.type ?? "?";
      console.info(`[newrelic-video] event ${name}`, evt?.data ?? evt);
    });

    let imaAdsTracker: { dispose?: () => void } | null = null;

    return {
      dispose: () => {
        try {
          imaAdsTracker?.dispose?.();
        } catch (e) {
          console.warn("[newrelic-video] ads dispose failed", e);
        }
        try {
          tracker.dispose?.();
        } catch (e) {
          console.warn("[newrelic-video] tracker dispose failed", e);
        }
      },
      attachIMAAdsTracker: async (imaAttachment) => {
        try {
          // The IMA ads tracker lives at @newrelic/video-html5/src/ads/ima.
          // We import via the published `src` folder; the package ships it.
          // The argument shape is the `player.ima` object from the IMA SDK
          // bridge (or compatible). The user wires this up when integrating
          // ads in BinchoPlayer.tsx (or wherever IMA gets initialised).
          const imaMod = (await import("@newrelic/video-html5/src/ads/ima.js")) as Record<string, unknown>;
          const Html5ImaAdsTracker = resolveCtor(
            imaMod,
            "default.default",
            "default",
            "Html5ImaAdsTracker",
          ) as new (p: unknown, o?: unknown) => unknown;
          if (typeof Html5ImaAdsTracker !== "function") {
            console.warn("[newrelic-video] could not resolve Html5ImaAdsTracker", imaMod);
            return;
          }
          imaAdsTracker = new Html5ImaAdsTracker(imaAttachment, {
            tag: videoEl,
            info: {
              videoId: meta.videoId,
              catName: meta.catName,
            },
          }) as { dispose?: () => void };
          tracker.setAdsTracker?.(imaAdsTracker);
          console.info("[newrelic-video] IMA ads tracker attached");
        } catch (e) {
          console.warn("[newrelic-video] failed to attach IMA ads tracker", e);
        }
      },
    };
  } catch (e) {
    console.warn("[newrelic-video] tracker load failed; running without video APM", e);
    return NOOP_HANDLE;
  }
}
