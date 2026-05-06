import { useEffect, useRef, useState } from "react";
import Hls from "hls.js";

interface Props {
  src: string;
  poster?: string;
  autoPlay?: boolean;
  controls?: boolean;
  className?: string;
  onReady?: (refs: { videoEl: HTMLVideoElement; hls: Hls | null }) => void;
}

const PLAYBACK_RATES = [0.5, 1, 1.25, 1.5, 2] as const;

// Bincho (シャム) — elegant, minimal HLS player.
// Wraps hls.js + <video>, no telemetry, no chrome. Compose this with
// <KanpachiPlayer> when you want metrics.
export function BinchoPlayer({ src, poster, autoPlay, controls = true, className, onReady }: Props) {
  const videoRef = useRef<HTMLVideoElement | null>(null);
  // Latest onReady kept in a ref so we don't tear down hls when the parent
  // passes a new inline callback on every render.
  const onReadyRef = useRef(onReady);
  onReadyRef.current = onReady;

  const [rate, setRate] = useState(1);
  const [loop, setLoop] = useState(false);

  useEffect(() => {
    const videoEl = videoRef.current;
    if (!videoEl) return;

    let hls: Hls | null = null;

    if (Hls.isSupported()) {
      hls = new Hls({ enableWorker: true });
      hls.loadSource(src);
      hls.attachMedia(videoEl);
    } else if (videoEl.canPlayType("application/vnd.apple.mpegurl")) {
      // Native HLS (Safari)
      videoEl.src = src;
    } else {
      console.error("[BinchoPlayer] HLS unsupported in this browser.");
    }

    onReadyRef.current?.({ videoEl, hls });

    return () => {
      if (hls) {
        hls.destroy();
      }
    };
  }, [src]);

  // Keep local state in sync with the underlying media element.
  useEffect(() => {
    const v = videoRef.current;
    if (!v) return;
    const onRate = () => setRate(v.playbackRate);
    v.addEventListener("ratechange", onRate);
    return () => v.removeEventListener("ratechange", onRate);
  }, []);

  const skip = (delta: number) => {
    const v = videoRef.current;
    if (!v) return;
    const dur = isFinite(v.duration) ? v.duration : Infinity;
    v.currentTime = Math.max(0, Math.min(dur, v.currentTime + delta));
  };

  const changeRate = (r: number) => {
    const v = videoRef.current;
    if (!v) return;
    v.playbackRate = r;
    setRate(r);
  };

  const toggleLoop = () => {
    const v = videoRef.current;
    if (!v) return;
    v.loop = !v.loop;
    setLoop(v.loop);
  };

  return (
    <div className={className} style={{ maxWidth: 720, margin: "0 auto" }}>
      <video
        ref={videoRef}
        poster={poster}
        autoPlay={autoPlay}
        controls={controls}
        playsInline
        style={{
          display: "block",
          width: "100%",
          maxHeight: "min(60vh, 480px)",
          background: "#0c0a08",
          borderRadius: 8,
          objectFit: "contain",
        }}
      />
      <div
        style={{
          display: "flex",
          flexWrap: "wrap",
          gap: 8,
          alignItems: "center",
          justifyContent: "center",
          marginTop: 10,
        }}
      >
        <button type="button" onClick={() => skip(-10)} title="10秒戻る">
          ⏪ 10s
        </button>
        <button type="button" onClick={() => skip(10)} title="10秒進む">
          10s ⏩
        </button>
        <span style={{ marginLeft: 8, fontSize: 12, opacity: 0.7 }}>速度</span>
        <div style={{ display: "inline-flex", gap: 4 }}>
          {PLAYBACK_RATES.map((r) => {
            const active = Math.abs(r - rate) < 0.001;
            return (
              <button
                key={r}
                type="button"
                onClick={() => changeRate(r)}
                aria-pressed={active}
                style={{
                  padding: "4px 8px",
                  background: active ? "var(--accent-kanpachi, #d68c45)" : "transparent",
                  color: active ? "#2a1810" : "inherit",
                  border: "1px solid var(--card-border, #d0d7de)",
                  borderRadius: 4,
                  fontWeight: active ? 700 : 400,
                  cursor: "pointer",
                }}
              >
                {r}x
              </button>
            );
          })}
        </div>
        <button
          type="button"
          onClick={toggleLoop}
          aria-pressed={loop}
          title="ループ再生"
          style={{
            marginLeft: 8,
            background: loop ? "var(--accent-kanpachi, #d68c45)" : "transparent",
            color: loop ? "#2a1810" : "inherit",
            border: "1px solid var(--card-border, #d0d7de)",
          }}
        >
          🔁 {loop ? "ループON" : "ループOFF"}
        </button>
      </div>
    </div>
  );
}
