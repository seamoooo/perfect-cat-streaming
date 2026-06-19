import type { ReactNode } from "react";
import { Link, NavLink } from "react-router-dom";
import nrLogo from "../../icons/newrelic.dark.svg";

const navClass = ({ isActive }: { isActive: boolean }) =>
  isActive ? "nav-link active" : "nav-link";

/** Shared chrome (header + footer) for the New Relic cat-streaming demo. */
export function Layout({ children }: { children: ReactNode }) {
  return (
    <>
      <header className="site-header">
        <Link to="/" className="site-brand">
          <img src={nrLogo} alt="New Relic" />
          <span>Perfect Cat Streaming</span>
        </Link>
        <nav className="site-nav">
          <NavLink to="/" end className={navClass}>
            トップ
          </NavLink>
          <NavLink to="/videos" className={navClass}>
            動画一覧
          </NavLink>
          <NavLink to="/upload" className={navClass}>
            アップロード
          </NavLink>
        </nav>
      </header>

      <main className="site-main">{children}</main>

      <footer className="site-footer">
        <img src={nrLogo} alt="New Relic" />
        New Relic 製品デモ — ねこチャンの動画でオブザーバビリティを体験 🐾
      </footer>
    </>
  );
}
