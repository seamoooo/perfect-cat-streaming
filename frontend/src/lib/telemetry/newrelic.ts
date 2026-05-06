import type { KanpachiAttrs, KanpachiSink } from "./index";

// Use the New Relic Browser Agent when present on the page.
// Drop the snippet (with VITE_NEW_RELIC_LICENSE_KEY / VITE_NEW_RELIC_APP_ID)
// into index.html in production to enable this sink. When the agent is not
// loaded, this sink quietly falls back to console for visibility.
declare global {
  interface Window {
    newrelic?: {
      addPageAction: (name: string, attrs: Record<string, unknown>) => void;
      noticeError?: (err: unknown, attrs?: Record<string, unknown>) => void;
    };
  }
}

export class NewRelicKanpachiSink implements KanpachiSink {
  record(eventName: string, attrs: KanpachiAttrs & Record<string, unknown>): void {
    const nr = typeof window !== "undefined" ? window.newrelic : undefined;
    if (nr && typeof nr.addPageAction === "function") {
      nr.addPageAction(eventName, attrs);
      return;
    }
    // Fallback so dev/staging without the agent still emits something visible.
    console.log(`[kanpachi:no-nr] ${eventName}`, attrs);
  }
}
