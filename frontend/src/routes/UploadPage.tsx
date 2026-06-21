import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Layout } from "../components/Layout";
import { useDocumentTitle } from "../hooks/useDocumentTitle";
import { uploadVideo } from "../lib/api";
import { CAT_BREED_GROUPS } from "../lib/breeds";

// "#かわいい もふもふ, ねこ部" → ["かわいい", "もふもふ", "ねこ部"]
function parseHashtags(input: string): string[] {
  return input
    .split(/[\s,]+/)
    .map((t) => t.replace(/^#+/, "").trim())
    .filter(Boolean);
}

/** Upload page — submit a new cat clip for transcoding. */
export function UploadPage() {
  useDocumentTitle("動画をアップロード");
  const navigate = useNavigate();
  const [uploading, setUploading] = useState(false);
  const [form, setForm] = useState({
    title: "",
    description: "",
    catName: "",
    breed: "siamese",
    tags: "",
  });
  const [file, setFile] = useState<File | null>(null);
  const [thumbnail, setThumbnail] = useState<File | null>(null);

  const thumbPreview = thumbnail ? URL.createObjectURL(thumbnail) : null;

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!file) return;
    setUploading(true);
    try {
      await uploadVideo({
        file,
        thumbnail,
        title: form.title || file.name,
        description: form.description,
        catName: form.catName || "ねこ",
        breed: form.breed,
        tags: parseHashtags(form.tags),
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
        <form onSubmit={onSubmit} className="upload-form">
          <label className="field-label">動画ファイル</label>
          <input
            type="file"
            accept="video/*"
            onChange={(e) => setFile(e.target.files?.[0] ?? null)}
            required
          />

          <label className="field-label">
            サムネイル画像（任意・一覧に表示。未指定なら動画から自動生成）
          </label>
          <div className="upload-thumb">
            <input
              type="file"
              accept="image/*"
              onChange={(e) => setThumbnail(e.target.files?.[0] ?? null)}
            />
            {thumbPreview && (
              <img src={thumbPreview} alt="サムネイルプレビュー" />
            )}
          </div>

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
          <div className="upload-row">
            <input
              placeholder="猫の名前 (例: みけ)"
              value={form.catName}
              onChange={(e) => setForm({ ...form, catName: e.target.value })}
            />
            <select
              value={form.breed}
              onChange={(e) => setForm({ ...form, breed: e.target.value })}
            >
              {CAT_BREED_GROUPS.map((g) => (
                <optgroup key={g.group} label={g.group}>
                  {g.breeds.map((b) => (
                    <option key={b.value} value={b.value}>
                      {b.label}
                    </option>
                  ))}
                </optgroup>
              ))}
            </select>
          </div>
          <input
            placeholder="ハッシュタグ (スペース区切り・例: かわいい もふもふ 寝顔)"
            value={form.tags}
            onChange={(e) => setForm({ ...form, tags: e.target.value })}
          />
          {form.tags.trim() && (
            <div className="hashtags">
              {parseHashtags(form.tags).map((t) => (
                <span key={t} className="hashtag">
                  #{t}
                </span>
              ))}
            </div>
          )}
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
