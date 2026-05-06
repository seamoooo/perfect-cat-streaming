import { useState } from "react";
import { CatCard } from "../components/CatCard";
import { useCatClips } from "../hooks/useCatClips";
import { uploadVideo } from "../lib/api";

export function GalleryPage() {
  const { videos, loading, error, refresh } = useCatClips();
  const [uploading, setUploading] = useState(false);
  const [form, setForm] = useState({
    title: "",
    description: "",
    catName: "Bincho",
    breed: "siamese" as "siamese" | "bengal" | "other",
    tags: "",
  });
  const [file, setFile] = useState<File | null>(null);

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!file) return;
    setUploading(true);
    try {
      await uploadVideo({
        file,
        title: form.title || file.name,
        description: form.description,
        catName: form.catName,
        breed: form.breed,
        tags: form.tags
          .split(",")
          .map((t) => t.trim())
          .filter(Boolean),
      });
      setFile(null);
      setForm((f) => ({ ...f, title: "", description: "", tags: "" }));
      await refresh();
    } catch (err) {
      alert(`アップロード失敗: ${err instanceof Error ? err.message : err}`);
    } finally {
      setUploading(false);
    }
  };

  return (
    <div style={{ maxWidth: 960, margin: "0 auto", padding: 24 }}>
      <header style={{ marginBottom: 24 }}>
        <h1 style={{ margin: 0 }}>Perfect Cat Streaming</h1>
        <p style={{ marginTop: 4, opacity: 0.75 }}>
          Bincho（シャム）と Kanpachi（ベンガル）のための、完璧なHLSストリーミング 🐾
        </p>
      </header>

      <section
        style={{
          background: "var(--card-bg)",
          border: "1px solid var(--card-border)",
          borderRadius: 12,
          padding: 16,
          marginBottom: 24,
        }}
      >
        <h2 style={{ marginTop: 0, fontSize: 18 }}>動画をアップロード</h2>
        <form onSubmit={onSubmit} style={{ display: "grid", gap: 8 }}>
          <input
            type="file"
            accept="video/mp4,video/*"
            onChange={(e) => setFile(e.target.files?.[0] ?? null)}
            required
          />
          <input
            placeholder="タイトル (空ならファイル名)"
            value={form.title}
            onChange={(e) => setForm({ ...form, title: e.target.value })}
          />
          <input
            placeholder="説明"
            value={form.description}
            onChange={(e) => setForm({ ...form, description: e.target.value })}
          />
          <div style={{ display: "flex", gap: 8 }}>
            <input
              placeholder="猫の名前 (例: Bincho)"
              value={form.catName}
              onChange={(e) => setForm({ ...form, catName: e.target.value })}
              style={{ flex: 1 }}
            />
            <select
              value={form.breed}
              onChange={(e) => setForm({ ...form, breed: e.target.value as typeof form.breed })}
            >
              <option value="siamese">シャム (Bincho)</option>
              <option value="bengal">ベンガル (Kanpachi)</option>
              <option value="other">その他</option>
            </select>
          </div>
          <input
            placeholder="タグ (カンマ区切り)"
            value={form.tags}
            onChange={(e) => setForm({ ...form, tags: e.target.value })}
          />
          <button type="submit" disabled={!file || uploading}>
            {uploading ? "アップロード中…" : "アップロード"}
          </button>
        </form>
      </section>

      <section>
        <div style={{ display: "flex", alignItems: "center", marginBottom: 12 }}>
          <h2 style={{ margin: 0, fontSize: 20 }}>クリップ一覧</h2>
          <button onClick={refresh} style={{ marginLeft: "auto" }}>
            再読み込み
          </button>
        </div>
        {loading && <p>読み込み中…</p>}
        {error && <p style={{ color: "salmon" }}>エラー: {error}</p>}
        {!loading && videos.length === 0 && <p>まだ動画がありません。</p>}
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fill, minmax(280px, 1fr))",
            gap: 16,
          }}
        >
          {videos.map((v) => (
            <CatCard key={v.id} video={v} />
          ))}
        </div>
      </section>
    </div>
  );
}
