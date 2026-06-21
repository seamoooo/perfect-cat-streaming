import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { deleteVideo, getVideo } from "../lib/api";
import { Layout } from "../components/Layout";
import { KanpachiPlayer } from "../components/player/KanpachiPlayer";
import { useDocumentTitle } from "../hooks/useDocumentTitle";
import { breedLabel } from "../lib/breeds";
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
  useDocumentTitle(video?.title);

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

  const isReady = video?.status === "ready" && !!video.playlistUrl;
  const posterUrl =
    isReady && video?.playlistUrl
      ? video.playlistUrl.replace(/index\.m3u8.*$/, "poster.jpg")
      : null;

  return (
    <Layout>
      {posterUrl && (
        <div
          className="detail-backdrop"
          style={{ backgroundImage: `url(${posterUrl})` }}
        />
      )}

      <p>
        <Link to="/videos" className="back-link">
          ← 一覧へ戻る
        </Link>
      </p>
      {error && <p className="error-text">エラー: {error}</p>}
      {!video && !error && <p className="muted">読み込み中…</p>}

      {video && (
        <article className="detail">
          {isReady ? (
            <KanpachiPlayer
              src={video.playlistUrl!}
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
            <div className="detail-pending">
              <div style={{ fontSize: 44, marginBottom: 8 }}>
                {video.status === "error" ? "🙀" : "⏳"}
              </div>
              <p style={{ margin: 0, fontWeight: 700 }}>
                {video.status === "error"
                  ? "変換に失敗しました"
                  : "変換中… もう少しお待ちください"}
              </p>
              {video.errorMsg && (
                <p className="muted" style={{ marginTop: 6, fontSize: 13 }}>
                  {video.errorMsg}
                </p>
              )}
            </div>
          )}

          <div className="detail-info">
            <div className="detail-meta-row">
              <span className="breed-chip">{breedLabel(video.breed)}</span>
              <span className="detail-catname">🐾 {video.catName}</span>
              {video.durationSec > 0 && (
                <span className="muted">
                  {formatDuration(video.durationSec)}
                </span>
              )}
            </div>
            <h1 className="detail-title">{video.title}</h1>
            {video.description && (
              <p className="detail-desc">{video.description}</p>
            )}
            {video.tags?.length ? (
              <div className="hashtags" style={{ marginTop: 12 }}>
                {video.tags.map((t) => (
                  <span key={t} className="hashtag">
                    #{t}
                  </span>
                ))}
              </div>
            ) : null}
          </div>

          <div className="detail-actions">
            <button
              type="button"
              className="btn btn-danger"
              onClick={onDeleteClick}
              disabled={deleting}
            >
              🗑 {deleting ? "削除中…" : "この動画を削除"}
            </button>
          </div>
        </article>
      )}
    </Layout>
  );
}

function formatDuration(sec: number): string {
  const m = Math.floor(sec / 60);
  const s = Math.round(sec % 60);
  return `${m}:${String(s).padStart(2, "0")}`;
}
