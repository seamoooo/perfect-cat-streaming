import { useEffect } from "react";

const BASE = "Perfect Cat Streaming";

/**
 * Sets document.title per route. SPAs share one index.html <title>, so updating
 * it on navigation gives each page a distinct, descriptive title for search
 * engines (Googlebot renders the SPA) and browser tabs / history.
 */
export function useDocumentTitle(title?: string) {
  useEffect(() => {
    document.title = title
      ? `${title} | ${BASE}`
      : `${BASE} — ねこチャンの動画配信 | New Relic デモ`;
  }, [title]);
}
