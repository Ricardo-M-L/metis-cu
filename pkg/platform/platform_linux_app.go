//go:build linux

package platform

// Linux app launch + frontmost-app gate. Strategy parallels darwin:
//   - shell out to `xdotool getactivewindow getwindowclassname` for
//     frontmost app detection (more reliable than parsing xprop output)
//   - shell out to `xdg-open` (preferred) or `gtk-launch` for application
//     launch by .desktop name
//   - granted state + persistence shared with darwin/windows via
//     granted.go
//
// Wayland note: the x11/Xwayland session works as-is; native Wayland
// (no XWayland) needs a per-compositor probe (sway: swaymsg, hyprland:
// hyprctl, gnome: dbus org.gnome.Shell). That's deliberately not
// implemented yet — most desktops still ship XWayland by default.

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// linuxOpenTimeout caps xdg-open / gtk-launch. App launches can take
// a while (gtk-launch fork-execs the .desktop), but anything past 10s
// is hostile UX.
const linuxOpenTimeout = 10 * time.Second

// defaultTiersLinux maps common Linux app class/exec names to access
// tiers. xdotool returns the WM_CLASS instance name (lowercase, e.g.
// "google-chrome"), which is what we key on. Names are best-effort
// across distros; user can override via RequestAccess.
var defaultTiersLinux = map[string]AccessTier{
	// browsers — read-only (no input allowed)
	"google-chrome":  TierRead,
	"chromium":       TierRead,
	"chrome":         TierRead,
	"firefox":        TierRead,
	"Firefox":        TierRead,
	"brave-browser":  TierRead,
	"microsoft-edge": TierRead,
	"vivaldi-stable": TierRead,
	"librewolf":      TierRead,

	// terminals + IDEs — click-only (no typing) — protects from
	// accidental destructive input
	"gnome-terminal-server": TierClick,
	"konsole":               TierClick,
	"xterm":                 TierClick,
	"alacritty":             TierClick,
	"Alacritty":             TierClick,
	"kitty":                 TierClick,
	"wezterm":               TierClick,
	"st-256color":           TierClick,
	"tilix":                 TierClick,
	"foot":                  TierClick,
	"code":                  TierClick,
	"Code":                  TierClick,
	"code-oss":              TierClick,
	"cursor":                TierClick,
	"Cursor":                TierClick,
	"jetbrains-idea":        TierClick,
	"jetbrains-goland":      TierClick,
	"jetbrains-pycharm":     TierClick,
	"jetbrains-webstorm":    TierClick,
	"jetbrains-rubymine":    TierClick,

	// everything else falls through to TierFull
}

// OpenApplication tries `xdg-open` first (broadest desktop coverage),
// falling back to `gtk-launch` for .desktop names that aren't URIs.
// Each subprocess is bounded by ctx + linuxOpenTimeout — whichever
// fires first cancels (DD-2 + BUG-8).
//
// Error reporting (BUG-9 fix): when xdg-open is found AND runs but
// returns non-zero, we capture its error and only fall through to
// gtk-launch if the latter is present.
func (p *linuxPlatform) OpenApplication(parentCtx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("open application: name is empty")
	}

	var xdgErr error
	xdgPath, xdgLookupErr := exec.LookPath("xdg-open")
	if xdgLookupErr == nil {
		ctx, cancel := context.WithTimeout(parentCtx, linuxOpenTimeout)
		cmd := exec.CommandContext(ctx, xdgPath, name)
		out, runErr := cmd.CombinedOutput()
		cancel()
		if runErr == nil {
			return nil
		}
		if ctx.Err() == context.DeadlineExceeded {
			xdgErr = fmt.Errorf("xdg-open %q: timed out after %s", name, linuxOpenTimeout)
		} else if ctx.Err() == context.Canceled {
			return fmt.Errorf("xdg-open %q: cancelled by caller", name)
		} else {
			xdgErr = fmt.Errorf("xdg-open %q: %w (output: %s)", name, runErr, strings.TrimSpace(string(out)))
		}
	}

	if gtkPath, err := exec.LookPath("gtk-launch"); err == nil {
		ctx, cancel := context.WithTimeout(parentCtx, linuxOpenTimeout)
		cmd := exec.CommandContext(ctx, gtkPath, name)
		out, runErr := cmd.CombinedOutput()
		cancel()
		if runErr == nil {
			return nil
		}
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("gtk-launch %q: timed out after %s (xdg-open also failed: %v)", name, linuxOpenTimeout, xdgErr)
		}
		if ctx.Err() == context.Canceled {
			return fmt.Errorf("gtk-launch %q: cancelled by caller", name)
		}
		return fmt.Errorf("gtk-launch %q: %w (output: %s); xdg-open also failed: %v",
			name, runErr, strings.TrimSpace(string(out)), xdgErr)
	}

	if xdgErr != nil {
		return xdgErr
	}
	return fmt.Errorf("open application: neither xdg-open nor gtk-launch found in PATH")
}

// FrontmostApp uses xdotool to query the active window class (the X11
// equivalent of the macOS frontmost-app concept). Cached for 250ms via
// frontmost_cache.go to coalesce a burst of gated calls. Bounded by
// frontmostProbeDuration() so a stalled X server doesn't wedge the MCP request.
//
// Behaviour when xdotool is missing: returns ErrNotImplemented up the
// chain. The gate (pkg/tools/gate.go) currently fails closed on any
// frontmost lookup error, including this one — install xdotool (or run
// in a session with X11 access) to enable input gating on Linux. See
// BUG-19 for the install-message UX caveat.
func (p *linuxPlatform) FrontmostApp() (string, AccessTier, error) {
	ensureGrantedLoaded()
	if name, tier, err, ok := cachedFrontmost(); ok {
		return name, tier, err
	}

	xdotoolPath, err := exec.LookPath("xdotool")
	if err != nil {
		err = fmt.Errorf("xdotool not found in PATH (install xdotool to enable frontmost-app gating on X11; native Wayland is not yet supported)")
		storeFrontmost("", TierFull, err)
		return "", TierFull, err
	}
	// Two-step: getactivewindow returns the X11 window ID; we then ask
	// for its WM_CLASS instance name. Combining them in one call also
	// works (xdotool getactivewindow getwindowclassname) but the chained
	// form is hostile to error reporting — split for clarity.
	ctx, cancel := context.WithTimeout(context.Background(), frontmostProbeDuration())
	defer cancel()
	winCmd := exec.CommandContext(ctx, xdotoolPath, "getactivewindow", "getwindowclassname")
	out, err := winCmd.Output()
	if ctx.Err() == context.DeadlineExceeded {
		err = fmt.Errorf("xdotool getactivewindow: timed out after %s", frontmostProbeDuration())
		storeFrontmost("", TierFull, err)
		return "", TierFull, err
	}
	if err != nil {
		err = fmt.Errorf("xdotool getactivewindow: %w", err)
		storeFrontmost("", TierFull, err)
		return "", TierFull, err
	}
	name := strings.TrimSpace(string(out))
	if name == "" {
		err = fmt.Errorf("xdotool getactivewindow: empty class name")
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
	if tier, ok := defaultTiersLinux[name]; ok {
		storeFrontmost(name, tier, nil)
		return name, tier, nil
	}
	// Fail CLOSED for unrecognised apps (see darwin twin): TierClick blocks
	// keystroke injection / app-launch into unapproved apps; old default was
	// the fail-open TierFull.
	storeFrontmost(name, TierClick, nil)
	return name, TierClick, nil
}

func (p *linuxPlatform) GrantedApplications() ([]string, error) {
	ensureGrantedLoaded()
	seen := map[string]struct{}{}
	for app := range defaultTiersLinux {
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

// Confirm is unimplemented on Linux until we pick a dialog backend
// (zenity / kdialog / notify-send vary by desktop). Returning
// ErrNotImplemented makes the request_access tool surface a clear
// error rather than silently auto-approving.
func (p *linuxPlatform) Confirm(message string) (bool, error) {
	return false, ErrNotImplemented
}

// RequestAccess: see darwin twin for the BUG-12 + BUG-20 fixes.
func (p *linuxPlatform) RequestAccess(apps []string, tier AccessTier) (map[string]AccessTier, error) {
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

// Tier: linux twin of darwin's Tier — consults grants then linux
// defaults, falls through to TierFull.
func (p *linuxPlatform) Tier(name string) AccessTier {
	ensureGrantedLoaded()
	grantedMu.RLock()
	if t, ok := granted[name]; ok {
		grantedMu.RUnlock()
		return t
	}
	grantedMu.RUnlock()
	if t, ok := defaultTiersLinux[name]; ok {
		return t
	}
	return TierFull
}
