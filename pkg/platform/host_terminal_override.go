package platform

// host_terminal_override.go — opt-in session-scoped tier override for
// "host terminal" applications (the kind of program that typically
// runs an MCP client like metis itself).
//
// Background. The hard-coded defaultTiers map puts Terminal / iTerm2 /
// Ghostty / VSCode etc. at TierClick "so a stray `type` command can't
// run a destructive shell." For most MCP callers (Claude Desktop on
// its own, etc.) that's the right default. But when the MCP CLIENT
// itself is hosted inside a terminal — which is the metis case — every
// `open_application` call ends up gated on the terminal app's tier
// (Frontmost = iTerm2 = TierClick) and gets rejected:
//
//   tier "click" on app "iTerm2" does not permit "full" operations
//
// (Session 41040bea, 2026-05-26: the user asked "open Douyin and
// search …" and the entire turn got stuck on this gate before the
// model even reached the actual UI automation.)
//
// Fix. Expose a single override: when the host configures it (via the
// config TOML `[gate] host_terminal_tier = "..."`, or by setting the
// METIS_CU_HOST_TERMINAL_TIER env var when spawning the cu server),
// every name in the host-terminal set returns that tier instead of
// the default TierClick. User-explicit grants (RequestAccess) still
// win — the override is consulted AFTER granted, BEFORE defaultTiers.
//
// Default behavior is UNCHANGED for callers that don't set the
// override — Terminal / iTerm2 / … keep returning TierClick exactly
// as before. Non-terminal apps are never affected by this code path.

import (
	"strings"
	"sync"
)

// hostTerminalApps is the closed allow-list of "host terminal"
// applications. macOS display names (FrontmostApp returns these). New
// terminals can be added here without breaking anything: the override
// is opt-in, so an unrecognised terminal just keeps TierClick.
//
// Kept as a set (rather than a slice) so lookups stay O(1) inside the
// hot Tier() / FrontmostApp() path.
var hostTerminalApps = map[string]struct{}{
	"Terminal":   {}, // macOS Apple_Terminal
	"iTerm2":     {},
	"Ghostty":    {},
	"WezTerm":    {},
	"wezterm-gui": {}, // Linux process name some setups expose
	"Alacritty":  {},
	"kitty":      {},
	"Hyper":      {},
	"Tabby":      {},
}

var (
	hostTerminalMu       sync.RWMutex
	hostTerminalOverride AccessTier // empty string = no override active
)

// SetHostTerminalOverride configures the session-scoped tier returned
// for every name in hostTerminalApps. Passing an empty / unknown tier
// disables the override (restores default behaviour).
//
// Idempotent and safe to call multiple times; the last call wins.
// Called once at server boot from pkg/server/server.go after the env
// + config merge.
func SetHostTerminalOverride(t AccessTier) {
	hostTerminalMu.Lock()
	defer hostTerminalMu.Unlock()
	switch t {
	case TierRead, TierClick, TierFull:
		hostTerminalOverride = t
	default:
		hostTerminalOverride = ""
	}
	// Invalidate the frontmost cache so a previously-cached
	// (iTerm2, TierClick) entry can't survive a runtime promotion.
	invalidateFrontmostCache()
}

// HostTerminalOverride returns the currently-configured override
// tier, or empty if none is set. Exported for diagnostics / tests.
func HostTerminalOverride() AccessTier {
	hostTerminalMu.RLock()
	defer hostTerminalMu.RUnlock()
	return hostTerminalOverride
}

// hostTerminalOverrideFor returns (tier, ok) — ok=true means name is
// a known host terminal AND an override is currently active. Both
// FrontmostApp() and Tier() consult this between the user-granted
// map and the defaultTiers fallback.
func hostTerminalOverrideFor(name string) (AccessTier, bool) {
	if name == "" {
		return "", false
	}
	hostTerminalMu.RLock()
	override := hostTerminalOverride
	hostTerminalMu.RUnlock()
	if override == "" {
		return "", false
	}
	// Case-insensitive match against the closed allow-list — some
	// platforms vary capitalisation between display name and process
	// name ("kitty" vs "Kitty"). Linear scan is fine, the list is ~10
	// entries.
	lower := strings.ToLower(name)
	for app := range hostTerminalApps {
		if strings.ToLower(app) == lower {
			return override, true
		}
	}
	return "", false
}

// HostTerminalApps returns a snapshot of the recognised host-terminal
// app names. Exported so the server boot path can log which apps the
// override applies to and so tests can pin the list.
func HostTerminalApps() []string {
	out := make([]string, 0, len(hostTerminalApps))
	for app := range hostTerminalApps {
		out = append(out, app)
	}
	return out
}
