package platform

// Per-user granted-app overrides — shared across darwin / linux / windows.
// Persisted to $HOME/.metis-cu/granted.json so approvals survive MCP
// server restarts. The map is not OS-specific: app names from different
// platforms coexist (e.g. macOS "Safari" and linux "firefox" can both
// be entries) but each OS implementation only consults entries that
// match its own naming convention.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	grantedMu   sync.RWMutex
	granted     = map[string]AccessTier{}
	grantedOnce = &sync.Once{}
)

// resetGrantedForTest wipes the in-memory map and discards the lazy-load
// sentinel so the next FrontmostApp / GrantedApplications / RequestAccess
// call re-reads from disk under the test's HOME. Only safe to call from
// tests with no concurrent readers.
func resetGrantedForTest() {
	grantedMu.Lock()
	granted = map[string]AccessTier{}
	grantedOnce = &sync.Once{}
	grantedMu.Unlock()
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
func ensureGrantedLoaded() {
	grantedMu.RLock()
	once := grantedOnce
	grantedMu.RUnlock()
	once.Do(loadGranted)
}

// saveGranted persists the in-memory map atomically. Caller must hold
// grantedMu (read lock is sufficient — we only read the map). Writes to
// a sibling tempfile and renames so a crash mid-write cannot leave the
// file truncated.
func saveGranted() error {
	wire := map[string]string{}
	for app, tier := range granted {
		wire[app] = string(tier)
	}
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
