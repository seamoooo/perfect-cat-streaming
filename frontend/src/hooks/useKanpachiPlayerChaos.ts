import { useEffect, useRef, useState } from "react";
import type Hls from "hls.js";

// Fraction of playbacks (0..1) that get a simulated fatal player error a few
// seconds into playback. Default ~1 in 5. Set VITE_PLAYER_CHAOS_ERROR_RATE=0
// to disable. Read once at module load.
const ENV_RATE = Number(import.meta.env.VITE_PLAYER_CHAOS_ERROR_RATE);
export const PLAYER_CHAOS_ERROR_RATE = Number.isFinite(ENV_RATE)
  ? Math.min(1, Math.max(0, ENV_RATE))
  : 0.2;

const MIN_DELAY_MS = 2000;
const MAX_DELAY_MS = 6000;

interface Args {
  videoEl: HTMLVideoElement | null;
  hls: Hls | null;
  /** changes per clip / reload so the dice are re-rolled */
  videoKey: string;
}

/**
 * Kanpachi chaos: with probability PLAYER_CHAOS_ERROR_RATE, a few seconds into
 * playback, fire a fatal player error. The error is injected through hls.js's
 * real ERROR event, so it flows through every telemetry layer already listening
 * — our kanpachi.hiss metric and the New Relic Video Agent — exactly like a
 * genuine CDN / fragment-load failure would.
 *
 * Returns `errored` (so the player can show an overlay) and `reset()`.
 */
export function useKanpachiPlayerChaos({ videoEl, hls, videoKey }: Args) {
  const [errored, setErrored] = useState(false);
  const armedRef = useRef(false);

  useEffect(() => {
    setErrored(false);
    armedRef.current = false;
    if (!videoEl || PLAYER_CHAOS_ERROR_RATE <= 0) return;
    // Roll once per playback; most plays are spared.
    if (Math.random() >= PLAYER_CHAOS_ERROR_RATE) return;

    let timer: number | undefined;

    const fire = () => {
      try {
        videoEl.pause();
      } catch {
        /* ignore */
      }
      setErrored(true);
      if (hls) {
        const HlsLib = hls.constructor as typeof Hls;
        // Emit a real hls.js ERROR → kanpachi.hiss + NR Video Agent error.
        const anyHls = hls as unknown as {
          trigger: (event: unknown, data: unknown) => void;
        };
        anyHls.trigger(HlsLib.Events.ERROR, {
          type: HlsLib.ErrorTypes.NETWORK_ERROR,
          details: HlsLib.ErrorDetails.FRAG_LOAD_ERROR,
          fatal: true,
          error: new Error("kanpachi chaos: simulated player error"),
          reason: "chaos", // marker to distinguish from real incidents
        });
      }
    };

    const onPlaying = () => {
      if (armedRef.current) return;
      armedRef.current = true;
      const delay =
        MIN_DELAY_MS + Math.random() * (MAX_DELAY_MS - MIN_DELAY_MS);
      timer = window.setTimeout(fire, delay);
    };

    videoEl.addEventListener("playing", onPlaying);
    return () => {
      videoEl.removeEventListener("playing", onPlaying);
      if (timer) window.clearTimeout(timer);
    };
  }, [videoEl, hls, videoKey]);

  return { errored, reset: () => setErrored(false) };
}
