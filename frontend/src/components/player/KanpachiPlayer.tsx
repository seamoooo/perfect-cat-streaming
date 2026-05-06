import { useCallback, useState } from "react";
import type Hls from "hls.js";
import { BinchoPlayer } from "./BinchoPlayer";
import { useKanpachiMetrics, type VideoMeta } from "../../hooks/useKanpachiMetrics";
import type { KanpachiSink } from "../../lib/telemetry";

interface Props {
  src: string;
  poster?: string;
  autoPlay?: boolean;
  controls?: boolean;
  videoMeta: VideoMeta;
  sink: KanpachiSink;
  sessionId: string;
}

// Kanpachi (ベンガル) — instrumented player. Wraps the elegant Bincho player and
// hunts QoE metrics like a Bengal stalking prey. Use this in production pages.
export function KanpachiPlayer({ src, poster, autoPlay, controls, videoMeta, sink, sessionId }: Props) {
  const [videoEl, setVideoEl] = useState<HTMLVideoElement | null>(null);
  const [hls, setHls] = useState<Hls | null>(null);

  useKanpachiMetrics({ videoEl, hls, videoMeta, sink, sessionId });

  const handleReady = useCallback(
    (refs: { videoEl: HTMLVideoElement; hls: Hls | null }) => {
      setVideoEl(refs.videoEl);
      setHls(refs.hls);
    },
    [],
  );

  return (
    <BinchoPlayer
      src={src}
      poster={poster}
      autoPlay={autoPlay}
      controls={controls}
      onReady={handleReady}
    />
  );
}
