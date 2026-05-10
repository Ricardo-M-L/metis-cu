package platform

// Frontmost-app subprocess timeout shared across darwin (osascript)
// and linux (xdotool). DD-3 turns this into a config-driven knob —
// pkg/server reads [gate] frontmost_timeout_ms from config.toml and
// calls SetFrontmostProbeTimeout at boot.
//
// Default 1.5s — well above the normal frontmost lookup latency
// (~30ms on macOS, ~5ms on Linux) and below the user-perceptible
// stall threshold for a single tool call.

import (
	"sync"
	"time"
)

var (
	frontmostProbeMu      sync.RWMutex
	frontmostProbeTimeout = 1500 * time.Millisecond
)

// frontmostProbeDuration returns the current per-call subprocess cap.
// Used by darwin's osascript invocation and linux's xdotool
// invocation in their respective FrontmostApp implementations.
func frontmostProbeDuration() time.Duration {
	frontmostProbeMu.RLock()
	defer frontmostProbeMu.RUnlock()
	return frontmostProbeTimeout
}

// SetFrontmostProbeTimeout overrides the cap. Non-positive values
// are silently ignored.
func SetFrontmostProbeTimeout(d time.Duration) {
	if d <= 0 {
		return
	}
	frontmostProbeMu.Lock()
	defer frontmostProbeMu.Unlock()
	frontmostProbeTimeout = d
}
