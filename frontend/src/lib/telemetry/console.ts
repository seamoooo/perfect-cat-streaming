import type { KanpachiAttrs, KanpachiSink } from "./index";

export class ConsoleKanpachiSink implements KanpachiSink {
  record(eventName: string, attrs: KanpachiAttrs & Record<string, unknown>): void {
    // Group log lines so DevTools is readable while developing.

    console.log(
      `%c[kanpachi]%c ${eventName}`,
      "color:#d68c45;font-weight:bold;",
      "color:inherit;",
      attrs,
    );
  }
}
