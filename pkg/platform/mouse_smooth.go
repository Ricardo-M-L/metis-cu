package platform

// Mouse-drag smoothing speed shared across darwin / linux / windows.
// robotgo.MoveSmooth takes (low, high) easing parameters — higher
// values = slower, more "human-like" motion; lower = brisker.
//
// Defaults match the pre-DD-3 hardcoded 1.0 / 1.0 — brisk but
// non-instant, so canvas/editor apps that distinguish drag from
// teleport still see a real drag. pkg/server overrides via
// SetMouseSmooth from config.toml [mouse] section.
//
// Reads/writes are not synchronised — values are set once at boot
// from config and never mutated again.

import "sync"

var (
	mouseSmoothMu   sync.RWMutex
	mouseSmoothLow  = 1.0
	mouseSmoothHigh = 1.0
)

// mouseSmoothSpeed returns the current (low, high) MoveSmooth params.
func mouseSmoothSpeed() (float64, float64) {
	mouseSmoothMu.RLock()
	defer mouseSmoothMu.RUnlock()
	return mouseSmoothLow, mouseSmoothHigh
}

// SetMouseSmooth overrides the MoveSmooth easing values. Non-positive
// inputs are silently ignored so a partial config can't disable
// smoothing accidentally.
func SetMouseSmooth(low, high float64) {
	mouseSmoothMu.Lock()
	defer mouseSmoothMu.Unlock()
	if low > 0 {
		mouseSmoothLow = low
	}
	if high > 0 {
		mouseSmoothHigh = high
	}
}
