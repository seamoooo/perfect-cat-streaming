import { useEffect, useRef } from "react";
import type Hls from "hls.js";
import {
  attachVideoTracker,
  type VideoTrackerHandle,
  type VideoTrackerMeta,
} from "../lib/observability/video";

/**
 * Hook: attach a New Relic Html5 video tracker to the underlying <video> when
 * present, and dispose it on unmount / when the source changes. Returns a ref
 * to the handle so callers can later call `current?.attachIMAAdsTracker(...)`
 * once the Google IMA SDK is wired in.
 */
export function useNewRelicVideoTracker(
  videoEl: HTMLVideoElement | null,
  hls: Hls | null,
  meta: VideoTrackerMeta,
) {
  const handleRef = useRef<VideoTrackerHandle | null>(null);

  useEffect(() => {
    if (!videoEl) return;
    let cancelled = false;
    attachVideoTracker(videoEl, hls, meta).then((h) => {
      if (cancelled) {
        h.dispose();
        return;
      }
      handleRef.current = h;
    });
    return () => {
      cancelled = true;
      handleRef.current?.dispose();
      handleRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [videoEl, meta.videoId]);

  return handleRef;
}
