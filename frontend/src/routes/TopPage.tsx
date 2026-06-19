import { Link } from "react-router-dom";
import { Layout } from "../components/Layout";
import nyarick from "../../images/nyarick.jpg";

/** Landing page for the New Relic product demo. */
export function TopPage() {
  return (
    <Layout>
      <section className="hero">
        <div>
          <span className="kicker">New Relic product demo</span>
          <h1>
            ねこチャンの動画を
            <br />
            <span className="accent">再生・配信</span>しよう
          </h1>
          <p className="lead">
            ねこチャンの動画を再生して、ねこチャンを見て癒やされましょう。
            再生品質やパフォーマンスは New Relic で計測しています。
          </p>
          <div className="hero-cta">
            <Link to="/videos" className="btn btn-primary">
              ▶ 動画を見る
            </Link>
            <Link to="/upload" className="btn btn-outline">
              ⬆ 動画をアップロード
            </Link>
          </div>
        </div>
        <div className="hero-art">
          <img src={nyarick} alt="ねこチャン" />
        </div>
      </section>

      <aside className="disclaimer" role="note">
        <strong>⚠️ ご注意 — 非公式の個人制作デモサイトです</strong>
        <p>
          本サイトは New Relic
          の製品デモを目的として、社員が個人的に制作したものです。New Relic
          社の公式サービスではなく、会社とは一切関係ありません。
          また、安定的な提供を意図したものではなく、予告なく停止・データ削除を行う場合があります。
        </p>
      </aside>
    </Layout>
  );
}
