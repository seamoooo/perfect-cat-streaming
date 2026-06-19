import { useState } from "react";
import { Link } from "react-router-dom";
import type { Video } from "../types/video";
import { deleteVideo } from "../lib/api";

const breedBadge: Record<string, { label: string; bg: string; fg: string }> = {
  siamese: { label: "シャム", bg: "#7ec8e3", fg: "#0c2333" },
  bengal: { label: "ベンガル", bg: "#d68c45", fg: "#2a1810" },
  other: { label: "その他", bg: "#cfcfcf", fg: "#222" },
};

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
  const breed = breedBadge[video.breed] ?? breedBadge.other;
  const [hover, setHover] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [imgFailed, setImgFailed] = useState(false);

  // Poster image lives next to the HLS playlist (index.m3u8 → poster.jpg).
  // Only ready videos have a playlist; older clips without a poster fall back
  // to the placeholder via onError.
  const posterUrl =
    video.status === "ready" && video.playlistUrl
      ? video.playlistUrl.replace(/index\.m3u8.*$/, "poster.jpg")
      : null;

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
      style={{ position: "relative" }}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
    >
      <Link
        to={`/clips/${video.id}`}
        style={{
          display: "block",
          textDecoration: "none",
          color: "inherit",
          background: "var(--card-bg)",
          border: "1px solid var(--card-border)",
          borderRadius: 12,
          padding: 0,
          overflow: "hidden",
          transition: "transform 120ms ease",
          transform: hover ? "translateY(-2px)" : "translateY(0)",
          opacity: deleting ? 0.45 : 1,
          pointerEvents: deleting ? "none" : "auto",
        }}
      >
        <div
          style={{
            position: "relative",
            aspectRatio: "16 / 9",
            background:
              "linear-gradient(135deg, var(--card-border), var(--bg-elevated))",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            overflow: "hidden",
          }}
        >
          {posterUrl && !imgFailed ? (
            <img
              src={posterUrl}
              alt={video.title}
              loading="lazy"
              onError={() => setImgFailed(true)}
              style={{
                width: "100%",
                height: "100%",
                objectFit: "cover",
                transition: "transform 200ms ease",
                transform: hover ? "scale(1.05)" : "scale(1)",
              }}
            />
          ) : (
            <span style={{ fontSize: 40, opacity: 0.5 }}>
              {video.status === "ready" ? "🐱" : "⏳"}
            </span>
          )}
        </div>
        <div style={{ padding: 16 }}>
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: 8,
              marginBottom: 8,
            }}
          >
            <span
              style={{
                background: breed.bg,
                color: breed.fg,
                fontSize: 12,
                padding: "2px 8px",
                borderRadius: 999,
                fontWeight: 600,
              }}
            >
              {breed.label}
            </span>
            <strong style={{ fontSize: 14 }}>{video.catName}</strong>
            <span style={{ marginLeft: "auto", fontSize: 12, opacity: 0.7 }}>
              {statusLabel[video.status] ?? video.status}
            </span>
          </div>
          <h3 style={{ margin: "4px 0", fontSize: 18, paddingRight: 24 }}>
            {video.title}
          </h3>
          {video.description && (
            <p
              style={{ margin: 0, fontSize: 13, opacity: 0.8, lineHeight: 1.5 }}
            >
              {video.description}
            </p>
          )}
          {video.tags?.length ? (
            <div
              style={{
                marginTop: 8,
                display: "flex",
                flexWrap: "wrap",
                gap: 4,
              }}
            >
              {video.tags.map((t) => (
                <span key={t} style={{ fontSize: 11, opacity: 0.7 }}>
                  #{t}
                </span>
              ))}
            </div>
          ) : null}
        </div>
      </Link>

      <button
        type="button"
        onClick={onDeleteClick}
        disabled={deleting}
        aria-label="動画を削除"
        title="削除"
        style={{
          position: "absolute",
          top: 8,
          right: 8,
          width: 26,
          height: 26,
          padding: 0,
          borderRadius: "50%",
          background: deleting ? "#888" : "rgba(180, 50, 50, 0.92)",
          color: "#fff",
          border: "1px solid rgba(255,255,255,0.15)",
          fontSize: 16,
          lineHeight: 1,
          fontWeight: 700,
          cursor: deleting ? "wait" : "pointer",
          opacity: hover || deleting ? 1 : 0,
          transition: "opacity 120ms ease",
          boxShadow: "0 2px 4px rgba(0,0,0,0.25)",
        }}
      >
        {deleting ? "…" : "×"}
      </button>
    </div>
  );
}
