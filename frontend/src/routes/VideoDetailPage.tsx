import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { getVideo } from "../lib/api";
import { KanpachiPlayer } from "../components/player/KanpachiPlayer";
import type { Video } from "../types/video";
import type { KanpachiSink } from "../lib/telemetry";

interface Props {
  sink: KanpachiSink;
  sessionId: string;
}

export function VideoDetailPage({ sink, sessionId }: Props) {
  const { id = "" } = useParams();
  const [video, setVideo] = useState<Video | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    setError(null);
    setVideo(null);
    getVideo(id)
      .then((v) => {
        if (active) setVideo(v);
      })
      .catch((e) => {
        if (active) setError(e instanceof Error ? e.message : String(e));
      });
    return () => {
      active = false;
    };
  }, [id]);

  return (
    <div style={{ maxWidth: 900, margin: "0 auto", padding: 24 }}>
      <p>
        <Link to="/">← 一覧へ戻る</Link>
      </p>
      {error && <p style={{ color: "salmon" }}>エラー: {error}</p>}
      {!video && !error && <p>読み込み中…</p>}
      {video && (
        <>
          <header style={{ marginBottom: 16 }}>
            <h1 style={{ margin: 0 }}>{video.title}</h1>
            <p style={{ marginTop: 4, opacity: 0.8 }}>
              {video.catName} ({video.breed}){video.description ? ` — ${video.description}` : ""}
            </p>
          </header>

          {video.status === "ready" && video.playlistUrl ? (
            <KanpachiPlayer
              src={video.playlistUrl}
              autoPlay
              videoMeta={{
                videoId: video.id,
                catName: video.catName,
                breed: video.breed,
                title: video.title,
                tags: video.tags,
              }}
              sink={sink}
              sessionId={sessionId}
            />
          ) : (
            <p>
              ステータス: <strong>{video.status}</strong>
              {video.errorMsg ? ` — ${video.errorMsg}` : ""}
            </p>
          )}
        </>
      )}
    </div>
  );
}
