//go:build darwin

package platform

// Application control + frontmost-app gate. Sprint 2-D implementation:
// shells out to `open -a` for launch and to osascript for the frontmost
// lookup, then maps the process name to one of the three AccessTiers
// (read / click / full). User-granted overrides (RequestAccess) are
// persisted to $HOME/.metis-cu/granted.json so approvals survive
// restarts of the MCP server.

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// defaultTiers is the hard-coded classification table. Browsers are
// "read" (visible but no input), terminals/IDEs are "click" (left-click
// only — protects from accidental destructive typing), everything else
// falls through to TierFull.
var defaultTiers = map[string]AccessTier{
	// browsers — read-only (no input allowed)
	"Safari":         TierRead,
	"Google Chrome":  TierRead,
	"Firefox":        TierRead,
	"Microsoft Edge": TierRead,
	"Arc":            TierRead,
	"Brave Browser":  TierRead,

	// terminals + IDEs — click-only (no typing) — protects from accidental destructive input
	"Terminal":           TierClick,
	"iTerm2":             TierClick,
	"Alacritty":          TierClick,
	"Visual Studio Code": TierClick,
	"Code":               TierClick,
	"Cursor":             TierClick,
	"Xcode":              TierClick,
	"GoLand":             TierClick,
	"IntelliJ IDEA":      TierClick,
	"PyCharm":            TierClick,
	"WebStorm":           TierClick,
	"RubyMine":           TierClick,

	// everything else falls through to TierFull
}

// granted state + persistence helpers live in granted.go (no build
// tag, shared across darwin / linux / windows).

// OpenApplication shells out to `open -a <name>` to launch a macOS
// application by display name. The exec returns non-zero with stderr
// like "Unable to find application named ..." which we wrap so the LLM
// sees a useful message.
func (p *darwinPlatform) OpenApplication(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("open application: name is empty")
	}
	cmd := exec.Command("open", "-a", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("open -a %q: %w (output: %s)", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// FrontmostApp returns the name of the app currently in the foreground
// plus its assigned tier. Looks up the name in the user-granted map
// first (so explicit approvals override defaults) and falls back to the
// hard-coded defaultTiers; anything else is TierFull.
func (p *darwinPlatform) FrontmostApp() (string, AccessTier, error) {
	ensureGrantedLoaded()
	cmd := exec.Command("osascript", "-e",
		`tell application "System Events" to get name of first process whose frontmost is true`)
	out, err := cmd.Output()
	if err != nil {
		return "", TierFull, fmt.Errorf("osascript frontmost: %w", err)
	}
	name := strings.TrimSpace(string(out))
	if name == "" {
		return "", TierFull, fmt.Errorf("osascript frontmost: empty name")
	}
	grantedMu.RLock()
	if tier, ok := granted[name]; ok {
		grantedMu.RUnlock()
		return name, tier, nil
	}
	grantedMu.RUnlock()
	if tier, ok := defaultTiers[name]; ok {
		return name, tier, nil
	}
	return name, TierFull, nil
}

// GrantedApplications returns the union of the default classification
// table keys and any user-granted overrides, sorted alphabetically. The
// MCP `list_granted_applications` tool surfaces this so the LLM can
// reason about which apps it can drive.
func (p *darwinPlatform) GrantedApplications() ([]string, error) {
	ensureGrantedLoaded()
	seen := map[string]struct{}{}
	for app := range defaultTiers {
		seen[app] = struct{}{}
	}
	grantedMu.RLock()
	for app := range granted {
		seen[app] = struct{}{}
	}
	grantedMu.RUnlock()
	out := make([]string, 0, len(seen))
	for app := range seen {
		out = append(out, app)
	}
	sort.Strings(out)
	return out, nil
}

// RequestAccess records the apps as user-approved at TierFull and
// persists the result. NOTE: the tool spec implies this prompts the
// user — for now we trust the agent's caller to act as a no-prompt
// approve-list. A future iteration can wrap this with an OS dialog.
func (p *darwinPlatform) RequestAccess(apps []string) (map[string]AccessTier, error) {
	ensureGrantedLoaded()
	if len(apps) == 0 {
		return map[string]AccessTier{}, nil
	}
	out := make(map[string]AccessTier, len(apps))
	grantedMu.Lock()
	for _, app := range apps {
		app = strings.TrimSpace(app)
		if app == "" {
			continue
		}
		granted[app] = TierFull
		out[app] = TierFull
	}
	err := saveGranted()
	grantedMu.Unlock()
	if err != nil {
		return out, fmt.Errorf("persist granted: %w", err)
	}
	return out, nil
}
