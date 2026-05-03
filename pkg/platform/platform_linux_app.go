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
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

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
// The exec error is wrapped with the stderr so the LLM sees actionable
// detail.
func (p *linuxPlatform) OpenApplication(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("open application: name is empty")
	}
	// Prefer xdg-open for URLs, file paths, and registered MIME types.
	if path, err := exec.LookPath("xdg-open"); err == nil {
		cmd := exec.Command(path, name)
		out, err := cmd.CombinedOutput()
		if err == nil {
			return nil
		}
		// Fall through to gtk-launch for .desktop name resolution.
		_ = out
	}
	if path, err := exec.LookPath("gtk-launch"); err == nil {
		cmd := exec.Command(path, name)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("gtk-launch %q: %w (output: %s)", name, err, strings.TrimSpace(string(out)))
		}
		return nil
	}
	return fmt.Errorf("open application: neither xdg-open nor gtk-launch found in PATH")
}

// FrontmostApp uses xdotool to query the active window class (the X11
// equivalent of the macOS frontmost-app concept). Falls back to TierFull
// when xdotool is unavailable rather than failing closed — Linux desktop
// access is less strictly gated than macOS by default, and an absent
// xdotool typically means "developer headless box, no GUI to gate on".
func (p *linuxPlatform) FrontmostApp() (string, AccessTier, error) {
	ensureGrantedLoaded()
	xdotoolPath, err := exec.LookPath("xdotool")
	if err != nil {
		return "", TierFull, fmt.Errorf("xdotool not found in PATH (install xdotool to enable frontmost-app gating)")
	}
	// Two-step: getactivewindow returns the X11 window ID; we then ask
	// for its WM_CLASS instance name. Combining them in one call also
	// works (xdotool getactivewindow getwindowclassname) but the chained
	// form is hostile to error reporting — split for clarity.
	winCmd := exec.Command(xdotoolPath, "getactivewindow", "getwindowclassname")
	out, err := winCmd.Output()
	if err != nil {
		return "", TierFull, fmt.Errorf("xdotool getactivewindow: %w", err)
	}
	name := strings.TrimSpace(string(out))
	if name == "" {
		return "", TierFull, fmt.Errorf("xdotool getactivewindow: empty class name")
	}
	grantedMu.RLock()
	if tier, ok := granted[name]; ok {
		grantedMu.RUnlock()
		return name, tier, nil
	}
	grantedMu.RUnlock()
	if tier, ok := defaultTiersLinux[name]; ok {
		return name, tier, nil
	}
	return name, TierFull, nil
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

func (p *linuxPlatform) RequestAccess(apps []string) (map[string]AccessTier, error) {
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
