package tools

import (
	"fmt"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

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
// Phase 2-D wires this helper but does not yet plumb it through the
// mouse / keyboard / clipboard adapters; Phase 3 finishes the chain so
// every input-bearing tool inherits gating uniformly.
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
