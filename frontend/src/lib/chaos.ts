// Developer demo chaos directives, triggered by a keyword in a video's
// description. Mirrors backend/internal/chaos/directive.go — keep the two in
// sync.
//
//   "sre"      → backend transcode throughput collapses (handled server-side)
//   "player"   → the HLS player throws a fatal error
//   "frontend" → a browser-side render error is raised
//   "backend"  → the API returns HTTP 500 (handled server-side)
//
// By contract only one keyword is expected per description; if several appear
// the first in CHAOS_ORDER wins so behaviour is deterministic.

export type ChaosMode = "sre" | "player" | "frontend" | "backend";

const CHAOS_ORDER: ChaosMode[] = ["sre", "player", "frontend", "backend"];

/**
 * Returns the chaos mode requested by a description, or null when none is
 * present. The keyword must appear as a standalone ASCII token (case
 * insensitive), so ordinary Japanese prose never triggers it.
 */
export function chaosDirective(
  description: string | undefined | null,
): ChaosMode | null {
  if (!description) return null;
  const present = new Set(
    description
      .toLowerCase()
      .split(/[^a-z0-9]+/)
      .filter(Boolean),
  );
  for (const mode of CHAOS_ORDER) {
    if (present.has(mode)) return mode;
  }
  return null;
}
