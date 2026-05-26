package platform

import (
	"testing"
)

// TestHostTerminalOverride_DisabledByDefault — pinning the no-op
// default: an unconfigured server must not surface any override.
// Regressing this would silently change tier semantics for every
// non-metis MCP client that consumes metis-cu.
func TestHostTerminalOverride_DisabledByDefault(t *testing.T) {
	// Ensure fresh state for tests run out of order.
	SetHostTerminalOverride("")

	if got := HostTerminalOverride(); got != "" {
		t.Errorf("default override = %q, want empty", got)
	}
	if _, ok := hostTerminalOverrideFor("iTerm2"); ok {
		t.Error("iTerm2 must NOT be overridden when no tier is set")
	}
}

// TestHostTerminalOverride_PromotesHostTerminals — when configured,
// every name in hostTerminalApps returns the configured tier.
// Iterates the whole allow-list so a future addition to the map can't
// silently miss the override.
func TestHostTerminalOverride_PromotesHostTerminals(t *testing.T) {
	SetHostTerminalOverride(TierFull)
	t.Cleanup(func() { SetHostTerminalOverride("") })

	for _, app := range HostTerminalApps() {
		tier, ok := hostTerminalOverrideFor(app)
		if !ok {
			t.Errorf("hostTerminalOverrideFor(%q) ok=false, want true", app)
			continue
		}
		if tier != TierFull {
			t.Errorf("hostTerminalOverrideFor(%q) = %q, want full", app, tier)
		}
	}
}

// TestHostTerminalOverride_LeavesNonTerminalsAlone — the override
// must NOT bleed into apps outside the host-terminal allow-list.
// Without this guarantee setting tier=full would silently weaken
// Safari/Chrome's read-only protection.
func TestHostTerminalOverride_LeavesNonTerminalsAlone(t *testing.T) {
	SetHostTerminalOverride(TierFull)
	t.Cleanup(func() { SetHostTerminalOverride("") })

	for _, app := range []string{"Safari", "Google Chrome", "Visual Studio Code", "Slack", "Discord"} {
		if _, ok := hostTerminalOverrideFor(app); ok {
			t.Errorf("non-terminal app %q was matched by host-terminal override", app)
		}
	}
}

// TestHostTerminalOverride_RejectsUnknownTier — Set with an invalid
// tier value must disable the override (no surprise default like
// silently picking TierFull).
func TestHostTerminalOverride_RejectsUnknownTier(t *testing.T) {
	SetHostTerminalOverride(TierFull) // arm it first
	SetHostTerminalOverride("nonsense")
	t.Cleanup(func() { SetHostTerminalOverride("") })

	if got := HostTerminalOverride(); got != "" {
		t.Errorf("after invalid set, override = %q, want empty (disabled)", got)
	}
}

// TestHostTerminalOverride_CaseInsensitive — kitty vs Kitty, etc.
// Process names vs display names differ across platforms; the
// override should not be sensitive to that.
func TestHostTerminalOverride_CaseInsensitive(t *testing.T) {
	SetHostTerminalOverride(TierFull)
	t.Cleanup(func() { SetHostTerminalOverride("") })

	for _, variant := range []string{"iterm2", "ITERM2", "iTerm2", "Kitty", "KITTY"} {
		if _, ok := hostTerminalOverrideFor(variant); !ok {
			t.Errorf("host-terminal lookup should be case-insensitive; %q not matched", variant)
		}
	}
}
