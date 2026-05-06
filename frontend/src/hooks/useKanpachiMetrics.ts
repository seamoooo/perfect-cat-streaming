import { useEffect, useRef } from "react";
import type Hls from "hls.js";
import type { KanpachiSink, KanpachiAttrs } from "../lib/telemetry";
import { PLAYER_VERSION } from "../lib/telemetry";

export interface VideoMeta {
  videoId: string;
  catName: string;
  breed: string;
  title: string;
  tags?: string[];
}

interface Args {
  videoEl: HTMLVideoElement | null;
  hls: Hls | null;
  videoMeta: VideoMeta;
  sink: KanpachiSink;
  /** static, generated once per session by App */
  sessionId: string;
}

// Kanpachi (ベンガル) hunts the QoE metrics:
//   kanpachi.first_pounce   = TTFF  (first frame after play())
//   kanpachi.hairball_start = rebuffer started (waiting)
//   kanpachi.hairball_end   = rebuffer ended (playing again)
//   kanpachi.hiss           = error (video element or hls.js)
//   kanpachi.stretch        = bitrate change (LEVEL_SWITCHED)
export function useKanpachiMetrics({ videoEl, hls, videoMeta, sink, sessionId }: Args) {
  const playStartedAtRef = useRef<number | null>(null);
  const ttffEmittedRef = useRef(false);
  const rebufferStartRef = useRef<number | null>(null);
  const rebufferTotalMsRef = useRef(0);
  const rebufferCountRef = useRef(0);

  useEffect(() => {
    if (!videoEl) return;

    const baseAttrs = (): KanpachiAttrs => ({
      videoId: videoMeta.videoId,
      catName: videoMeta.catName,
      breed: videoMeta.breed,
      title: videoMeta.title,
      tags: videoMeta.tags,
      sessionId,
      playerVersion: PLAYER_VERSION,
      userAgent: navigator.userAgent,
    });

    const onPlay = () => {
      if (playStartedAtRef.current == null) {
        playStartedAtRef.current = performance.now();
      }
    };

    const onPlaying = () => {
      // TTFF (first time only)
      if (!ttffEmittedRef.current && playStartedAtRef.current != null) {
        const ttffMs = Math.round(performance.now() - playStartedAtRef.current);
        ttffEmittedRef.current = true;
        sink.record("kanpachi.first_pounce", { ...baseAttrs(), ttffMs });
      }
      // rebuffer ended
      if (rebufferStartRef.current != null) {
        const dur = Math.round(performance.now() - rebufferStartRef.current);
        rebufferTotalMsRef.current += dur;
        sink.record("kanpachi.hairball_end", {
          ...baseAttrs(),
          durationMs: dur,
          rebufferCount: rebufferCountRef.current,
          rebufferTotalMs: rebufferTotalMsRef.current,
        });
        rebufferStartRef.current = null;
      }
    };

    const onWaiting = () => {
      // Ignore "waiting" before first play (initial buffering counted by TTFF).
      if (playStartedAtRef.current == null) return;
      if (rebufferStartRef.current != null) return;
      rebufferStartRef.current = performance.now();
      rebufferCountRef.current += 1;
      sink.record("kanpachi.hairball_start", {
        ...baseAttrs(),
        rebufferCount: rebufferCountRef.current,
      });
    };

    const onError = () => {
      const err = videoEl.error;
      sink.record("kanpachi.hiss", {
        ...baseAttrs(),
        source: "video.error",
        code: err?.code,
        message: err?.message,
      });
    };

    videoEl.addEventListener("play", onPlay);
    videoEl.addEventListener("playing", onPlaying);
    videoEl.addEventListener("waiting", onWaiting);
    videoEl.addEventListener("error", onError);

    return () => {
      videoEl.removeEventListener("play", onPlay);
      videoEl.removeEventListener("playing", onPlaying);
      videoEl.removeEventListener("waiting", onWaiting);
      videoEl.removeEventListener("error", onError);
    };
  }, [videoEl, videoMeta, sink, sessionId]);

  // hls.js-specific listeners (bitrate transitions, fatal errors)
  useEffect(() => {
    if (!hls) return;

    const baseAttrs = (): KanpachiAttrs => ({
      videoId: videoMeta.videoId,
      catName: videoMeta.catName,
      breed: videoMeta.breed,
      title: videoMeta.title,
      tags: videoMeta.tags,
      sessionId,
      playerVersion: PLAYER_VERSION,
      userAgent: navigator.userAgent,
    });

    const HlsLib = (hls.constructor as typeof Hls);
    const Events = HlsLib.Events;

    const onLevelSwitched = (_e: unknown, data: { level: number }) => {
      const level = hls.levels?.[data.level];
      sink.record("kanpachi.stretch", {
        ...baseAttrs(),
        levelIndex: data.level,
        bitrate: level?.bitrate,
        height: level?.height,
        width: level?.width,
      });
    };

    const onError = (_e: unknown, data: { type: string; details: string; fatal: boolean }) => {
      sink.record("kanpachi.hiss", {
        ...baseAttrs(),
        source: "hls.js",
        type: data.type,
        details: data.details,
        fatal: data.fatal,
      });
    };

    hls.on(Events.LEVEL_SWITCHED, onLevelSwitched);
    hls.on(Events.ERROR, onError);
    return () => {
      hls.off(Events.LEVEL_SWITCHED, onLevelSwitched);
      hls.off(Events.ERROR, onError);
    };
  }, [hls, videoMeta, sink, sessionId]);
}
