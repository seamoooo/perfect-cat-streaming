import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { deleteVideo, getVideo, updateVideo } from "../lib/api";
import { Layout } from "../components/Layout";
import { ErrorBoundary } from "../components/ErrorBoundary";
import { KanpachiPlayer } from "../components/player/KanpachiPlayer";
import { useDocumentTitle } from "../hooks/useDocumentTitle";
import { breedLabel } from "../lib/breeds";
import { chaosDirective } from "../lib/chaos";
import type { Video } from "../types/video";
import type { KanpachiSink } from "../lib/telemetry";

// Developer "frontend" chaos demo: throwing during render lets the surrounding
// ErrorBoundary capture it and surface the JS error in New Relic Browser,
// exactly like a real client-side crash. Placed inside the boundary subtree.
function FrontendChaosTripwire(): never {
  throw new Error("chaos: simulated frontend render error");
}

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
  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [draft, setDraft] = useState({ title: "", description: "" });
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

  const startEdit = () => {
    if (!video) return;
    setDraft({ title: video.title, description: video.description ?? "" });
    setEditing(true);
  };

  const saveEdit = async () => {
    if (!video || saving) return;
    const title = draft.title.trim();
    if (!title) {
      window.alert("タイトルを入力してください");
      return;
    }
    setSaving(true);
    try {
      const updated = await updateVideo(video.id, {
        title,
        description: draft.description,
      });
      setVideo(updated);
      setEditing(false);
    } catch (e) {
      window.alert(`保存失敗: ${e instanceof Error ? e.message : String(e)}`);
    } finally {
      setSaving(false);
    }
  };

  const chaosMode = chaosDirective(video?.description);
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
        <ErrorBoundary>
          <article className="detail">
            {chaosMode === "frontend" && <FrontendChaosTripwire />}
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
                forcePlayerError={chaosMode === "player"}
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
              {editing ? (
                <div className="detail-edit">
                  <label className="field-label">タイトル</label>
                  <input
                    value={draft.title}
                    onChange={(e) =>
                      setDraft({ ...draft, title: e.target.value })
                    }
                    placeholder="タイトル"
                  />
                  <label className="field-label">説明</label>
                  <textarea
                    value={draft.description}
                    onChange={(e) =>
                      setDraft({ ...draft, description: e.target.value })
                    }
                    placeholder="説明"
                    rows={3}
                  />
                  <div className="detail-edit-actions">
                    <button
                      type="button"
                      className="btn btn-primary"
                      onClick={saveEdit}
                      disabled={saving}
                    >
                      {saving ? "保存中…" : "保存"}
                    </button>
                    <button
                      type="button"
                      className="btn btn-outline"
                      onClick={() => setEditing(false)}
                      disabled={saving}
                    >
                      キャンセル
                    </button>
                  </div>
                </div>
              ) : (
                <>
                  <h1 className="detail-title">{video.title}</h1>
                  {video.description && (
                    <p className="detail-desc">{video.description}</p>
                  )}
                </>
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
              {!editing && (
                <button
                  type="button"
                  className="btn btn-outline"
                  onClick={startEdit}
                  disabled={deleting}
                >
                  ✏️ 編集
                </button>
              )}
              <button
                type="button"
                className="btn btn-danger"
                onClick={onDeleteClick}
                disabled={deleting || editing}
              >
                🗑 {deleting ? "削除中…" : "この動画を削除"}
              </button>
            </div>
          </article>
        </ErrorBoundary>
      )}
    </Layout>
  );
}

function formatDuration(sec: number): string {
  const m = Math.floor(sec / 60);
  const s = Math.round(sec % 60);
  return `${m}:${String(s).padStart(2, "0")}`;
}
