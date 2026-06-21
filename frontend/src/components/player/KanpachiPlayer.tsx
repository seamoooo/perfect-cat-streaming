import { useCallback, useState } from "react";
import type Hls from "hls.js";
import { BinchoPlayer } from "./BinchoPlayer";
import {
  useKanpachiMetrics,
  type VideoMeta,
} from "../../hooks/useKanpachiMetrics";
import { useNewRelicVideoTracker } from "../../hooks/useNewRelicVideoTracker";
import { useKanpachiPlayerChaos } from "../../hooks/useKanpachiPlayerChaos";
import type { KanpachiSink } from "../../lib/telemetry";

interface Props {
  src: string;
  poster?: string;
  autoPlay?: boolean;
  controls?: boolean;
  videoMeta: VideoMeta;
  sink: KanpachiSink;
  sessionId: string;
  /** developer demo: always trigger the simulated player error */
  forcePlayerError?: boolean;
}

// Kanpachi (ベンガル) — instrumented player. Wraps the elegant Bincho player and
// hunts QoE metrics like a Bengal stalking prey. Use this in production pages.
//
// Two telemetry layers run side by side:
//   1. useKanpachiMetrics  — our in-house event names (kanpachi.first_pounce
//      / hairball_start|end / hiss / stretch) routed through the KanpachiSink
//      abstraction (console in dev, NR addPageAction in prod).
//   2. useNewRelicVideoTracker — official NR Video Agent (@newrelic/video-html5)
//      auto-emits CONTENT_REQUEST / CONTENT_START / CONTENT_BUFFER_START|END /
//      CONTENT_RENDITION_CHANGE etc. The returned handle exposes
//      attachIMAAdsTracker so when Google IMA SDK is wired in for ads, the
//      AD_* events will start flowing without touching this component.
export function KanpachiPlayer({
  src,
  poster,
  autoPlay,
  controls,
  videoMeta,
  sink,
  sessionId,
  forcePlayerError = false,
}: Props) {
  const [videoEl, setVideoEl] = useState<HTMLVideoElement | null>(null);
  const [hls, setHls] = useState<Hls | null>(null);
  // Bump to remount BinchoPlayer (fresh hls + reload) on retry.
  const [reloadNonce, setReloadNonce] = useState(0);

  useKanpachiMetrics({ videoEl, hls, videoMeta, sink, sessionId });
  useNewRelicVideoTracker(videoEl, hls, videoMeta);
  const chaos = useKanpachiPlayerChaos({
    videoEl,
    hls,
    videoKey: `${videoMeta.videoId}#${reloadNonce}`,
    force: forcePlayerError,
  });

  const handleReady = useCallback(
    (refs: { videoEl: HTMLVideoElement; hls: Hls | null }) => {
      setVideoEl(refs.videoEl);
      setHls(refs.hls);
    },
    [],
  );

  const retry = () => {
    chaos.reset();
    setReloadNonce((n) => n + 1);
  };

  return (
    <div style={{ position: "relative" }}>
      <BinchoPlayer
        key={reloadNonce}
        src={src}
        poster={poster}
        autoPlay={autoPlay}
        controls={controls}
        onReady={handleReady}
      />
      {chaos.errored && (
        <div
          role="alert"
          style={{
            position: "absolute",
            inset: 0,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            background: "rgba(12, 10, 8, 0.82)",
            borderRadius: 8,
            backdropFilter: "blur(2px)",
            textAlign: "center",
            padding: 24,
          }}
        >
          <div>
            <div style={{ fontSize: 40, marginBottom: 8 }}>🙀</div>
            <p style={{ margin: "0 0 4px", fontWeight: 700, fontSize: 18 }}>
              再生エラーが発生しました
            </p>
            <p style={{ margin: "0 0 16px", fontSize: 13, opacity: 0.75 }}>
              ネットワーク起因の再生エラー（New Relic デモ用カオス）
            </p>
            <button type="button" className="btn btn-primary" onClick={retry}>
              ▶ もう一度再生
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
