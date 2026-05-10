package tools

import (
	"fmt"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

// requiredTier maps every input-bearing tool to the minimum frontmost-app
// tier it needs. Tools NOT in the map (vision: screenshot/cursor_position/
// switch_display/zoom; metadata: list_granted_applications/wait;
// administrative: request_access; reads: read_clipboard) skip gating —
// they're either harmless or self-gating.
//
// TierClick = pointer-only operations (clicks/move/scroll) — safe to
// dispatch into a terminal/IDE because they can't enter destructive text.
// TierFull = anything that can mutate frontmost content (typing, key
// combos, clipboard write, launching apps).
var requiredTier = map[string]platform.AccessTier{
	// mouse — TierClick
	"mouse_move":      platform.TierClick,
	"left_click":      platform.TierClick,
	"right_click":     platform.TierClick,
	"middle_click":    platform.TierClick,
	"double_click":    platform.TierClick,
	"triple_click":    platform.TierClick,
	"left_click_drag": platform.TierClick,
	"left_mouse_down": platform.TierClick,
	"left_mouse_up":   platform.TierClick,
	"scroll":          platform.TierClick,

	// keyboard / write — TierFull (can change frontmost content)
	"key":              platform.TierFull,
	"hold_key":         platform.TierFull,
	"type":             platform.TierFull,
	"write_clipboard":  platform.TierFull,
	"open_application": platform.TierFull,
}

// gateOrDeny is the per-handler entrypoint guard. Looks up the tool's
// required tier and runs EnforceTier; on deny returns a *Result the
// handler should return verbatim (`return res, nil`) and `true`. On
// allow (or when the tool isn't gated at all) returns `(nil, false)`.
func gateOrDeny(plat platform.Platform, toolName string) (*Result, bool) {
	req, ok := requiredTier[toolName]
	if !ok {
		return nil, false
	}
	if msg, allowed := EnforceTier(plat, req); !allowed {
		return &Result{Text: msg, IsError: true}, true
	}
	return nil, false
}

// EnforceTier asserts the frontmost app's tier is at least `required`.
// Tier ordering: TierRead < TierClick < TierFull.
//   - TierRead permits no input (only screenshot / read_clipboard etc.)
//   - TierClick permits left_click / mouse_move / scroll only
//   - TierFull permits everything
//
// Returns (deny-message, false) when the frontmost-app tier does not
// satisfy `required` — adapters short-circuit with an IsError Result
// using the message. Returns ("", true) when the operation is allowed.
//
// Per-tool dispatch goes through gateOrDeny which consults requiredTier;
// EnforceTier remains exported for tests and for ad-hoc use by tools
// that need a non-standard required tier.
func EnforceTier(plat platform.Platform, required platform.AccessTier) (string, bool) {
	name, current, err := plat.FrontmostApp()
	if err != nil {
		// If we can't determine the frontmost app, fail closed — deny
		// input. The user can RequestAccess to override.
		return fmt.Sprintf("frontmost app lookup failed: %v", err), false
	}
	if !tierAllows(current, required) {
		return fmt.Sprintf("tier %q on app %q does not permit %q operations",
			current, name, required), false
	}
	return "", true
}

// tierAllows answers whether `have` is at least as permissive as
// `required`. Read < Click < Full; an unknown tier on either side
// ranks 0 and therefore fails closed.
func tierAllows(have, required platform.AccessTier) bool {
	rank := map[platform.AccessTier]int{
		platform.TierRead:  1,
		platform.TierClick: 2,
		platform.TierFull:  3,
	}
	return rank[have] >= rank[required]
}
