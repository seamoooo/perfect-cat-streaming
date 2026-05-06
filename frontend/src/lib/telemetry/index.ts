// Kanpachi (ベンガル) hunts QoE metrics like prey. Any sink that satisfies
// KanpachiSink can receive them — swap implementations without touching the
// player or the metrics hook.

export interface KanpachiAttrs {
  videoId: string;
  catName: string;
  breed: string;
  title: string;
  tags?: string[];
  sessionId: string;
  playerVersion: string;
  userAgent: string;
  [key: string]: unknown;
}

export interface KanpachiSink {
  /**
   * Send one telemetry event.
   * @param eventName e.g. "kanpachi.first_pounce", "kanpachi.hairball_start"
   * @param attrs custom attributes; videoId/catName/breed/title come from videoMeta automatically
   */
  record(eventName: string, attrs: KanpachiAttrs & Record<string, unknown>): void;
}

export { ConsoleKanpachiSink } from "./console";
export { NewRelicKanpachiSink } from "./newrelic";

export const PLAYER_VERSION = "1.0.0-bincho-kanpachi";
