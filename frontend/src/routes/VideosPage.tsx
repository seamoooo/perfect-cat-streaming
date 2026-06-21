import { Link } from "react-router-dom";
import { Layout } from "../components/Layout";
import { CatCard } from "../components/CatCard";
import { useCatClips } from "../hooks/useCatClips";
import { useDocumentTitle } from "../hooks/useDocumentTitle";

/** Video list page — browse and play the cat clips. */
export function VideosPage() {
  useDocumentTitle("ねこチャンの動画一覧");
  const { videos, loading, error, refresh } = useCatClips();

  return (
    <Layout>
      <span className="kicker">Watch</span>
      <div className="section-head">
        <h2>ねこチャンの動画一覧</h2>
        <button className="btn btn-outline spacer" onClick={refresh}>
          再読み込み
        </button>
      </div>
      <p className="muted" style={{ marginTop: -8 }}>
        ねこチャンの動画が見れます。カードをクリックすると再生ページへ移動します。
      </p>

      {loading && <p className="muted">読み込み中…</p>}
      {error && <p className="error-text">エラー: {error}</p>}
      {!loading && videos.length === 0 && (
        <div className="card" style={{ textAlign: "center" }}>
          <p className="muted" style={{ marginTop: 0 }}>
            まだ動画がありません。
          </p>
          <Link to="/upload" className="btn btn-primary">
            ⬆ 最初の動画をアップロード
          </Link>
        </div>
      )}

      <div className="video-grid">
        {videos.map((v) => (
          <CatCard key={v.id} video={v} onDeleted={() => refresh()} />
        ))}
      </div>
    </Layout>
  );
}
