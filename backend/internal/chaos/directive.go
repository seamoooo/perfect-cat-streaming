package chaos

import "strings"

// Directive scans a free-text video description for a developer demo keyword and
// returns the chaos mode it requests, or "" when none is present.
//
// The keyword is matched as a standalone ASCII token (case-insensitive), so a
// description like "ベンガルの動画 backend テスト" triggers the "backend" mode
// while ordinary Japanese prose never does. By contract only one keyword is
// expected per description; if several appear, the first in KEYWORD priority
// order wins so behaviour stays deterministic.
//
// Mirrors the frontend helper in frontend/src/lib/chaos.ts — keep the two in
// sync. This only reads the string; it never builds SQL or shell input from it.
const (
	ModeSRE      = "sre"      // backend: transcode throughput collapses
	ModePlayer   = "player"   // frontend: HLS player throws a fatal error
	ModeFrontend = "frontend" // frontend: browser-side render error
	ModeBackend  = "backend"  // backend: API returns HTTP 500
)

// directiveOrder is the deterministic priority used when scanning tokens.
var directiveOrder = []string{ModeSRE, ModePlayer, ModeFrontend, ModeBackend}

func Directive(description string) string {
	if description == "" {
		return ""
	}
	// Tokenise on anything that isn't an ASCII letter/digit so the keyword must
	// stand alone (not be a substring of a longer word).
	tokens := strings.FieldsFunc(description, func(r rune) bool {
		switch {
		case r >= 'a' && r <= 'z':
			return false
		case r >= 'A' && r <= 'Z':
			return false
		case r >= '0' && r <= '9':
			return false
		default:
			return true
		}
	})
	present := make(map[string]bool, len(tokens))
	for _, t := range tokens {
		present[strings.ToLower(t)] = true
	}
	for _, mode := range directiveOrder {
		if present[mode] {
			return mode
		}
	}
	return ""
}
