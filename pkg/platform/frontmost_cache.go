package platform

// Per-process cache for FrontmostApp results. Lives outside the
// build-tagged platform_*.go files because the caching policy is
// identical on darwin / linux / windows — only the underlying
// subprocess (osascript / xdotool / GetForegroundWindow) differs.
//
// Why cache at all (DD-4):
//   - The gate fires FrontmostApp on EVERY input-bearing tool call
//     (left_click, type, scroll, etc.). A computer_batch with 30
//     steps shells out 30 times in a couple of seconds.
//   - osascript / System Events occasionally wedges under load —
//     a single timeout would normally take down the whole batch.
//     A short TTL lets us return the last known good answer and
//     keep going while a background re-fetch happens on the next
//     call after the TTL expires.
//
// Why so short (250ms):
//   - The user's frontmost app rarely changes during a batch (the
//     model is the one driving focus). 250ms is long enough to
//     coalesce a burst of 30 calls but short enough that human
//     window-switching during a long-running batch still gets
//     picked up within one tool-call.
//   - 250ms is well below the typical time it takes the model to
//     emit the next tool call, so cache hits don't mask a real
//     focus change between two model turns.

import (
	"sync"
	"time"
)

// frontmostCacheTTL is the per-process cache lifetime. Var (not
// const) so tests can crank it down to verify expiry and pkg/server
// can override from config.toml [gate] section (DD-3).
var frontmostCacheTTL = 250 * time.Millisecond

// SetFrontmostCacheTTL overrides the cache lifetime from config.
// Non-positive values are silently ignored. Reads through
// frontmostCacheMu so a concurrent cachedFrontmost call sees a
// coherent value during boot-time mutation.
func SetFrontmostCacheTTL(d time.Duration) {
	if d <= 0 {
		return
	}
	frontmostCacheMu.Lock()
	defer frontmostCacheMu.Unlock()
	frontmostCacheTTL = d
}

type frontmostCacheEntry struct {
	name  string
	tier  AccessTier
	err   error
	stamp time.Time
}

var (
	frontmostCacheMu sync.Mutex
	frontmostCache   frontmostCacheEntry
)

// cachedFrontmost returns the cached result if it's still fresh, or
// (zero entry, false) if the caller must re-probe. The caller is
// responsible for calling storeFrontmost once it has a fresh value.
func cachedFrontmost() (string, AccessTier, error, bool) {
	frontmostCacheMu.Lock()
	defer frontmostCacheMu.Unlock()
	if frontmostCache.stamp.IsZero() {
		return "", "", nil, false
	}
	if time.Since(frontmostCache.stamp) > frontmostCacheTTL {
		return "", "", nil, false
	}
	return frontmostCache.name, frontmostCache.tier, frontmostCache.err, true
}

// storeFrontmost records the outcome of a fresh probe. Errors are
// cached too — a wedged osascript today is likely wedged 50ms from
// now, and surfacing the error consistently is more useful than
// retrying on every gated call.
func storeFrontmost(name string, tier AccessTier, err error) {
	frontmostCacheMu.Lock()
	defer frontmostCacheMu.Unlock()
	frontmostCache = frontmostCacheEntry{
		name:  name,
		tier:  tier,
		err:   err,
		stamp: time.Now(),
	}
}

// invalidateFrontmostCache wipes the cached entry. Tests use this to
// reset between cases; production code does not need to call it.
func invalidateFrontmostCache() {
	frontmostCacheMu.Lock()
	defer frontmostCacheMu.Unlock()
	frontmostCache = frontmostCacheEntry{}
}
