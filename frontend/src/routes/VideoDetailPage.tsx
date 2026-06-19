import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { deleteVideo, getVideo } from "../lib/api";
import { Layout } from "../components/Layout";
import { KanpachiPlayer } from "../components/player/KanpachiPlayer";
import type { Video } from "../types/video";
import type { KanpachiSink } from "../lib/telemetry";

interface Props {
  sink: KanpachiSink;
  sessionId: string;
}

export function VideoDetailPage({ sink, sessionId }: Props) {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const [video, setVideo] = useState<Video | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);

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

  const onDeleteClick = async () => {
    if (!video || deleting) return;
    setDeleting(true);
    try {
      await deleteVideo(video.id);
      navigate("/videos");
    } catch (e) {
      window.alert(`削除失敗: ${e instanceof Error ? e.message : String(e)}`);
      setDeleting(false);
    }
  };

  return (
    <Layout>
      <p>
        <Link to="/videos">← 一覧へ戻る</Link>
      </p>
      {error && <p className="error-text">エラー: {error}</p>}
      {!video && !error && <p>読み込み中…</p>}
      {video && (
        <>
          <header style={{ marginBottom: 16 }}>
            <h1 style={{ margin: 0 }}>{video.title}</h1>
            <p style={{ marginTop: 4, opacity: 0.8 }}>
              {video.catName} ({video.breed})
              {video.description ? ` — ${video.description}` : ""}
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

          <div
            style={{
              marginTop: 32,
              paddingTop: 16,
              borderTop: "1px solid var(--card-border)",
              display: "flex",
              gap: 12,
              alignItems: "center",
              justifyContent: "flex-end",
            }}
          >
            <button
              type="button"
              onClick={onDeleteClick}
              disabled={deleting}
              style={{
                background: deleting ? "#777" : "#c03838",
                color: "#fff",
                border: "none",
                padding: "8px 16px",
                borderRadius: 6,
                fontWeight: 600,
                cursor: deleting ? "wait" : "pointer",
              }}
            >
              🗑 {deleting ? "削除中…" : "この動画を削除"}
            </button>
          </div>
        </>
      )}
    </Layout>
  );
}
