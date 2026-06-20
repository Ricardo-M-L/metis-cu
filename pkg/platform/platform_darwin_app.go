//go:build darwin

package platform

// Application control + frontmost-app gate. Sprint 2-D implementation:
// shells out to `open -a` for launch and to osascript for the frontmost
// lookup, then maps the process name to one of the three AccessTiers
// (read / click / full). User-granted overrides (RequestAccess) are
// persisted to $HOME/.metis-cu/granted.json so approvals survive
// restarts of the MCP server.

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// openTimeout caps `open -a` for app launches. Apps can take 10s+ to
// finish launching; we let the caller wait up to 10s and surface a
// timeout otherwise so the LLM can retry.
const openTimeout = 10 * time.Second

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
//
// ctx (DD-2): caller's ctx + openTimeout combined — whichever fires
// first cancels the subprocess. Lets a request-level cancel abort
// during a long cold-launch instead of waiting out the full 10s
// internal cap.
func (p *darwinPlatform) OpenApplication(ctx context.Context, name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("open application: name is empty")
	}
	ctx, cancel := context.WithTimeout(ctx, openTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "open", "-a", name)
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("open -a %q: timed out after %s", name, openTimeout)
	}
	if ctx.Err() == context.Canceled {
		return fmt.Errorf("open -a %q: cancelled by caller", name)
	}
	if err != nil {
		return fmt.Errorf("open -a %q: %w (output: %s)", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// FrontmostApp returns the name of the app currently in the foreground
// plus its assigned tier. Looks up the name in the user-granted map
// first (so explicit approvals override defaults) and falls back to the
// hard-coded defaultTiers; anything else is TierFull.
//
// Cached for 250ms (frontmost_cache.go) — a typical computer_batch
// fires this 5-30 times in a few seconds, and the user's frontmost
// app rarely changes within that window. Bounded by frontmostProbeDuration()
// so a wedged AppleEvent bridge can't block the whole MCP server.
func (p *darwinPlatform) FrontmostApp() (string, AccessTier, error) {
	ensureGrantedLoaded()
	if name, tier, err, ok := cachedFrontmost(); ok {
		return name, tier, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), frontmostProbeDuration())
	defer cancel()
	cmd := exec.CommandContext(ctx, "osascript", "-e",
		`tell application "System Events" to get name of first process whose frontmost is true`)
	out, err := cmd.Output()
	if ctx.Err() == context.DeadlineExceeded {
		err = fmt.Errorf("osascript frontmost: timed out after %s (System Events may be wedged)", frontmostProbeDuration())
		storeFrontmost("", TierFull, err)
		return "", TierFull, err
	}
	if err != nil {
		err = fmt.Errorf("osascript frontmost: %w", err)
		storeFrontmost("", TierFull, err)
		return "", TierFull, err
	}
	name := strings.TrimSpace(string(out))
	if name == "" {
		err = fmt.Errorf("osascript frontmost: empty name")
		storeFrontmost("", TierFull, err)
		return "", TierFull, err
	}
	grantedMu.RLock()
	if tier, ok := granted[name]; ok {
		grantedMu.RUnlock()
		storeFrontmost(name, tier, nil)
		return name, tier, nil
	}
	grantedMu.RUnlock()
	// Host-terminal session override (opt-in via config /
	// METIS_CU_HOST_TERMINAL_TIER env) sits between user-granted
	// entries and the hard-coded defaults. See
	// host_terminal_override.go for the rationale.
	if tier, ok := hostTerminalOverrideFor(name); ok {
		storeFrontmost(name, tier, nil)
		return name, tier, nil
	}
	if tier, ok := defaultTiers[name]; ok {
		storeFrontmost(name, tier, nil)
		return name, tier, nil
	}
	// Fail CLOSED for unrecognised apps: default to TierClick (left-click
	// only), NOT TierFull. The old fail-open default let the model type /
	// press keys / launch apps into ANY unlisted application — a password
	// manager, banking app, etc. — that the operator never approved.
	// TierClick blocks the keystroke-injection (secret-theft) vector while
	// still allowing basic navigation; operators grant TierFull explicitly
	// per app via RequestAccess.
	storeFrontmost(name, TierClick, nil)
	return name, TierClick, nil
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

// confirmScript builds the AppleScript source for the macOS native
// "Allow / Deny" dialog used by Confirm. Pulled out as a helper so
// tests can assert escaping without needing a GUI session — the real
// osascript invocation in Confirm would block on a Mac with no
// session attached. `giving up after 60` matches osascript's default
// dismissal, treated as "deny" by the wrapper.
func confirmScript(message string) string {
	escaped := strings.ReplaceAll(message, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return fmt.Sprintf(
		`display dialog "%s" buttons {"Deny", "Allow"} default button "Deny" `+
			`with title "metis-cu access request" with icon caution giving up after 60`,
		escaped,
	)
}

// hasGUISession reports whether the current process can drive a
// macOS GUI dialog. SSH-into-Mac and launchd-without-WindowServer
// scenarios fail silently otherwise — `osascript display dialog` just
// hangs ~60s before timing out, blocking the whole MCP request for
// the duration. The probe runs `osascript -e "1+1"` with a tight
// 500ms cap; that's enough to round-trip when the AE bridge is
// reachable but bails fast when it isn't (DD-5).
//
// Cached in the frontmost-cache layer? No — GUI presence rarely
// changes mid-session, but a one-call probe is fast enough that
// caching doesn't pay for itself.
func hasGUISession() (bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, "osascript", "-e", "1+1")
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return false, "osascript probe timed out (no GUI session reachable; SSH-into-Mac without console?)"
		}
		return false, fmt.Sprintf("osascript probe failed: %v", err)
	}
	return true, ""
}

// Confirm pops a native macOS dialog and waits for user input. The
// dialog defaults to "Deny" and times out after 60s (also treated as
// deny) so a forgotten prompt never auto-approves.
//
// Pre-flight checks GUI reachability (DD-5) so a missing WindowServer
// surfaces as a clear error instead of a 60s wedge. osascript exits
// non-zero when the user cancels or the dialog times out; both map to
// a clean (false, nil) — the caller's contract is "true means the
// user clicked Allow", not "no error means yes".
//
// We cap the dialog subprocess at 75s (15s slack over the 60s
// `giving up after`) so a stuck osascript can't outlive the dialog
// itself.
func (p *darwinPlatform) Confirm(message string) (bool, error) {
	if ok, why := hasGUISession(); !ok {
		return false, fmt.Errorf("Confirm: no GUI session: %s", why)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "osascript", "-e", confirmScript(message))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, nil
	}
	return strings.Contains(string(out), "button returned:Allow"), nil
}

// RequestAccess records the apps at the given tier and persists the
// result. The actual user prompt lives in the pkg/tools/request_access
// handler — this method is the lower-level "commit the grant"
// primitive. Calling it directly bypasses the confirmation dialog,
// which is intentional: tests and CLI repair flows need a way to seed
// grants without GUI dependence.
//
// Holds grantedMu.Lock only long enough to mutate the map and snapshot
// it; the disk write happens outside the lock so reader goroutines
// don't stall on I/O (BUG-12 fix). The frontmost cache is invalidated
// after the grant lands so a freshly-granted app's new tier takes
// effect on the very next FrontmostApp call.
//
// tier defaults to TierFull when empty / unknown (BUG-20: previously
// every grant collapsed to TierFull regardless of caller intent).
func (p *darwinPlatform) RequestAccess(apps []string, tier AccessTier) (map[string]AccessTier, error) {
	ensureGrantedLoaded()
	if len(apps) == 0 {
		return map[string]AccessTier{}, nil
	}
	tier = normaliseTier(tier)
	out := make(map[string]AccessTier, len(apps))
	grantedMu.Lock()
	for _, app := range apps {
		app = strings.TrimSpace(app)
		if app == "" {
			continue
		}
		granted[app] = tier
		out[app] = tier
	}
	wire := snapshotGranted()
	grantedMu.Unlock()
	invalidateFrontmostCache()
	if err := saveGranted(wire); err != nil {
		return out, fmt.Errorf("persist granted: %w", err)
	}
	return out, nil
}

// Tier returns the assigned tier for `name`. Consults user grants
// first (set via RequestAccess), then the host-terminal override (when
// set), then the hard-coded defaults; falls through to TierFull. Never
// errors — an unknown app is a TierFull app, matching FrontmostApp's
// behaviour.
func (p *darwinPlatform) Tier(name string) AccessTier {
	ensureGrantedLoaded()
	grantedMu.RLock()
	if t, ok := granted[name]; ok {
		grantedMu.RUnlock()
		return t
	}
	grantedMu.RUnlock()
	if t, ok := hostTerminalOverrideFor(name); ok {
		return t
	}
	if t, ok := defaultTiers[name]; ok {
		return t
	}
	return TierFull
}
