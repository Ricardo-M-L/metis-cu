package platform

// Per-user granted-app overrides — shared across darwin / linux / windows.
// Persisted to $HOME/.metis-cu/granted.json so approvals survive MCP
// server restarts. The map is not OS-specific: app names from different
// platforms coexist (e.g. macOS "Safari" and linux "firefox" can both
// be entries) but each OS implementation only consults entries that
// match its own naming convention.
//
// Locking layout (BUG-12 fix):
//   - grantedMu (RWMutex) — protects the in-memory `granted` map. RLock
//     for reads (FrontmostApp lookups, GrantedApplications); Lock for
//     mutations (RequestAccess applying new grants). NEVER held during
//     disk I/O — saveGranted snapshots under Lock then writes outside.
//   - fileMu (Mutex) — serialises the on-disk write so two parallel
//     RequestAccess calls don't interleave their tempfile renames. This
//     is a separate mutex so a slow disk write never blocks reader
//     goroutines on grantedMu.
//   - grantedOnceMu — guards swaps of grantedOnce (only used by the
//     test-only resetGrantedForTest helper).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	grantedMu     sync.RWMutex
	granted       = map[string]AccessTier{}
	fileMu        sync.Mutex
	grantedOnceMu sync.Mutex
	grantedOnce   = &sync.Once{}
)

// resetGrantedForTest wipes the in-memory map and discards the lazy-load
// sentinel so the next FrontmostApp / GrantedApplications / RequestAccess
// call re-reads from disk under the test's HOME. Only safe to call from
// tests with no concurrent readers.
//
// The grantedOnce swap goes under grantedOnceMu so a parallel reader
// in ensureGrantedLoaded sees a coherent pointer (BUG-12 race fix).
func resetGrantedForTest() {
	grantedMu.Lock()
	granted = map[string]AccessTier{}
	grantedMu.Unlock()
	grantedOnceMu.Lock()
	grantedOnce = &sync.Once{}
	grantedOnceMu.Unlock()
	invalidateFrontmostCache()
}

// grantedPath returns the persistence file location. Uses HOME so tests
// can override via t.Setenv.
func grantedPath() string {
	return filepath.Join(os.Getenv("HOME"), ".metis-cu", "granted.json")
}

// loadGranted reads the persisted approvals from disk. Missing file is
// not an error — we just start empty. Malformed JSON is logged-via-
// silence (start empty) since the caller has no signal channel here;
// the next write rewrites the file cleanly.
func loadGranted() {
	data, err := os.ReadFile(grantedPath())
	if err != nil {
		return
	}
	wire := map[string]string{}
	if err := json.Unmarshal(data, &wire); err != nil {
		return
	}
	grantedMu.Lock()
	defer grantedMu.Unlock()
	for app, tierStr := range wire {
		switch AccessTier(tierStr) {
		case TierRead:
			granted[app] = TierRead
		case TierClick:
			granted[app] = TierClick
		case TierFull:
			granted[app] = TierFull
		}
	}
}

// ensureGrantedLoaded triggers the lazy load exactly once per process.
// Reads grantedOnce under its own mutex to coordinate with
// resetGrantedForTest, which swaps the pointer (BUG-12 race fix).
func ensureGrantedLoaded() {
	grantedOnceMu.Lock()
	once := grantedOnce
	grantedOnceMu.Unlock()
	once.Do(loadGranted)
}

// saveGranted persists the granted map atomically. Crucially does NOT
// take grantedMu — the caller is expected to have snapshotted the map
// to `wire` under Lock and then released it BEFORE calling. fileMu
// serialises concurrent writes so two parallel RequestAccess calls
// don't race on the tempfile rename. (BUG-12: previously the disk
// write happened under grantedMu.Lock, blocking every reader for the
// duration of the I/O.)
func saveGranted(wire map[string]string) error {
	fileMu.Lock()
	defer fileMu.Unlock()
	data, err := json.MarshalIndent(wire, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal granted: %w", err)
	}
	path := grantedPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir granted dir: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write granted tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename granted: %w", err)
	}
	return nil
}

// normaliseTier collapses an empty / unknown tier value to TierFull
// (the historical default before BUG-20 added per-app tier choice).
// Centralised so the three platform RequestAccess implementations stay
// consistent without each repeating the same fallback logic.
func normaliseTier(t AccessTier) AccessTier {
	switch t {
	case TierRead, TierClick, TierFull:
		return t
	default:
		return TierFull
	}
}

// snapshotGranted returns a wire-format copy of the granted map. Caller
// must hold grantedMu.RLock during the call; the returned map is
// independent of the source so the caller can release the lock and
// pass the snapshot to saveGranted without risk of concurrent
// mutation. Used by RequestAccess implementations (BUG-12 fix).
func snapshotGranted() map[string]string {
	wire := make(map[string]string, len(granted))
	for app, tier := range granted {
		wire[app] = string(tier)
	}
	return wire
}
