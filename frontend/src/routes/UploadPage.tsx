import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Layout } from "../components/Layout";
import { uploadVideo } from "../lib/api";

/** Upload page — submit a new cat clip for transcoding. */
export function UploadPage() {
  const navigate = useNavigate();
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
      // After upload, jump to the list so the user can watch it go ready.
      navigate("/videos");
    } catch (err) {
      alert(`アップロード失敗: ${err instanceof Error ? err.message : err}`);
    } finally {
      setUploading(false);
    }
  };

  return (
    <Layout>
      <span className="kicker">Upload</span>
      <div className="section-head">
        <h2>ねこチャンの動画をアップロード</h2>
      </div>
      <p className="muted" style={{ marginTop: -8 }}>
        ねこちゃんの動画をアップロードできます。変換が終わると一覧から再生できます。
      </p>

      <section className="card" style={{ maxWidth: 560 }}>
        <form onSubmit={onSubmit} style={{ display: "grid", gap: 12 }}>
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
              onChange={(e) =>
                setForm({ ...form, breed: e.target.value as typeof form.breed })
              }
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
          <button
            type="submit"
            className="btn btn-primary"
            disabled={!file || uploading}
          >
            {uploading ? "アップロード中…" : "⬆ アップロード"}
          </button>
        </form>
      </section>
    </Layout>
  );
}
