import { useState } from "react";
import { Link } from "react-router-dom";
import type { Video } from "../types/video";
import { deleteVideo } from "../lib/api";
import { breedLabel } from "../lib/breeds";

const statusLabel: Record<string, string> = {
  pending: "受付済み",
  processing: "変換中…",
  ready: "再生可",
  error: "エラー",
};

interface Props {
  video: Video;
  onDeleted?: (id: string) => void;
}

export function CatCard({ video, onDeleted }: Props) {
  const [hover, setHover] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [imgFailed, setImgFailed] = useState(false);

  // Poster lives next to the HLS playlist (index.m3u8 → poster.jpg).
  const posterUrl =
    video.status === "ready" && video.playlistUrl
      ? video.playlistUrl.replace(/index\.m3u8.*$/, "poster.jpg")
      : null;
  const showPoster = posterUrl && !imgFailed;
  const isReady = video.status === "ready";

  const onDeleteClick = async (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (deleting) return;
    setDeleting(true);
    try {
      await deleteVideo(video.id);
      onDeleted?.(video.id);
    } catch (err) {
      window.alert(
        `削除失敗: ${err instanceof Error ? err.message : String(err)}`,
      );
      setDeleting(false);
    }
  };

  return (
    <div
      className={`nf-card${hover ? " is-hover" : ""}`}
      style={{ opacity: deleting ? 0.45 : 1 }}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
    >
      <Link
        to={`/clips/${video.id}`}
        className="nf-link"
        style={{ pointerEvents: deleting ? "none" : "auto" }}
      >
        <div className="nf-poster">
          {showPoster ? (
            <img
              className="nf-poster-img"
              src={posterUrl}
              alt={video.title}
              loading="lazy"
              onError={() => setImgFailed(true)}
            />
          ) : (
            <div className="nf-poster-ph">
              <span>{isReady ? "🐾" : "⏳"}</span>
            </div>
          )}

          {/* top badge */}
          <div className="nf-top">
            <span className="breed-chip">{breedLabel(video.breed)}</span>
          </div>

          {/* hover play button */}
          <div className="nf-play" aria-hidden>
            <span>▶</span>
          </div>

          {/* bottom gradient + info */}
          <div className="nf-overlay">
            <div className="nf-meta">
              <strong className="nf-title">{video.title}</strong>
              <div className="nf-sub">
                {video.catName}
                {!isReady && (
                  <span className="nf-status">
                    {statusLabel[video.status] ?? video.status}
                  </span>
                )}
              </div>
              {video.description && (
                <p className="nf-desc">{video.description}</p>
              )}
              {video.tags?.length ? (
                <div className="hashtags nf-tags">
                  {video.tags.slice(0, 4).map((t) => (
                    <span key={t} className="hashtag">
                      #{t}
                    </span>
                  ))}
                </div>
              ) : null}
            </div>
          </div>
        </div>
      </Link>

      <button
        type="button"
        onClick={onDeleteClick}
        disabled={deleting}
        aria-label="動画を削除"
        title="削除"
        className="nf-delete"
        style={{ opacity: hover || deleting ? 1 : 0 }}
      >
        {deleting ? "…" : "×"}
      </button>
    </div>
  );
}
