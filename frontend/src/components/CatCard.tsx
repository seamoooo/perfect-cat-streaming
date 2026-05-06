import { Link } from "react-router-dom";
import type { Video } from "../types/video";

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

export function CatCard({ video }: { video: Video }) {
  const breed = breedBadge[video.breed] ?? breedBadge.other;
  return (
    <Link
      to={`/clips/${video.id}`}
      style={{
        display: "block",
        textDecoration: "none",
        color: "inherit",
        background: "var(--card-bg)",
        border: "1px solid var(--card-border)",
        borderRadius: 12,
        padding: 16,
        transition: "transform 120ms ease",
      }}
      onMouseOver={(e) => (e.currentTarget.style.transform = "translateY(-2px)")}
      onMouseOut={(e) => (e.currentTarget.style.transform = "translateY(0)")}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 8 }}>
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
      <h3 style={{ margin: "4px 0", fontSize: 18 }}>{video.title}</h3>
      {video.description && (
        <p style={{ margin: 0, fontSize: 13, opacity: 0.8, lineHeight: 1.5 }}>{video.description}</p>
      )}
      {video.tags?.length ? (
        <div style={{ marginTop: 8, display: "flex", flexWrap: "wrap", gap: 4 }}>
          {video.tags.map((t) => (
            <span key={t} style={{ fontSize: 11, opacity: 0.7 }}>
              #{t}
            </span>
          ))}
        </div>
      ) : null}
    </Link>
  );
}
