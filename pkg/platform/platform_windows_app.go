//go:build windows

package platform

// Windows app launch + frontmost-app gate.
//
//   - FrontmostApp: GetForegroundWindow + GetWindowThreadProcessId via
//     direct user32 syscalls, then gopsutil for the process executable
//     name. Mapping the .exe name (lowercased, no ".exe") onto access
//     tiers gives parity with macOS / linux.
//   - OpenApplication: shells out to `cmd /c start "" "<name>"` which
//     resolves both .exe paths and registered protocols.
//   - Granted state shared via granted.go.

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"syscall"
	"unsafe"

	"github.com/shirou/gopsutil/v4/process"
	"golang.org/x/sys/windows"
)

// defaultTiersWindows maps lowercased .exe basenames (without ".exe")
// to access tiers. Windows app names are reasonably stable (Microsoft
// breaks fewer process names than macOS rebrands), so this list is
// shorter than the linux equivalent.
var defaultTiersWindows = map[string]AccessTier{
	// browsers
	"chrome":    TierRead,
	"firefox":   TierRead,
	"msedge":    TierRead,
	"brave":     TierRead,
	"opera":     TierRead,
	"vivaldi":   TierRead,
	"librewolf": TierRead,

	// terminals + IDEs — click-only
	"windowsterminal": TierClick,
	"conhost":         TierClick,
	"cmd":             TierClick,
	"powershell":      TierClick,
	"pwsh":            TierClick,
	"alacritty":       TierClick,
	"wezterm":         TierClick,
	"code":            TierClick,
	"cursor":          TierClick,
	"idea64":          TierClick,
	"goland64":        TierClick,
	"pycharm64":       TierClick,
	"webstorm64":      TierClick,
	"rubymine64":      TierClick,

	// everything else falls through to TierFull
}

var (
	user32                       = windows.NewLazySystemDLL("user32.dll")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
)

// foregroundProcessID returns the PID of the process owning the current
// foreground window. Returns 0 when the desktop has no foreground
// window (e.g. login screen) — caller should treat as "no app".
func foregroundProcessID() (uint32, error) {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return 0, fmt.Errorf("no foreground window")
	}
	var pid uint32
	r1, _, callErr := procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if r1 == 0 {
		return 0, fmt.Errorf("GetWindowThreadProcessId failed: %v", callErr)
	}
	if pid == 0 {
		return 0, fmt.Errorf("foreground window has no owning PID")
	}
	return pid, nil
}

// processBaseName returns the lowercased executable base name (no
// ".exe") for a PID. Used to key into defaultTiersWindows / granted.
func processBaseName(pid uint32) (string, error) {
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return "", fmt.Errorf("process(%d): %w", pid, err)
	}
	name, err := proc.Name()
	if err != nil {
		return "", fmt.Errorf("process(%d).Name: %w", pid, err)
	}
	name = strings.ToLower(strings.TrimSuffix(name, ".exe"))
	return name, nil
}

// OpenApplication shells out to `cmd /c start "" "<name>"` which
// handles both absolute paths to .exe files and registered protocols
// (e.g. "ms-settings:" or a URL). The empty "" before <name> is the
// window-title argument that `start` requires when the first quoted
// arg is the path.
func (p *windowsPlatform) OpenApplication(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("open application: name is empty")
	}
	cmd := exec.Command("cmd", "/c", "start", "", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("start %q: %w (output: %s)", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (p *windowsPlatform) FrontmostApp() (string, AccessTier, error) {
	ensureGrantedLoaded()
	pid, err := foregroundProcessID()
	if err != nil {
		return "", TierFull, err
	}
	name, err := processBaseName(pid)
	if err != nil {
		return "", TierFull, err
	}
	if name == "" {
		return "", TierFull, fmt.Errorf("foreground process name is empty")
	}
	grantedMu.RLock()
	if tier, ok := granted[name]; ok {
		grantedMu.RUnlock()
		return name, tier, nil
	}
	grantedMu.RUnlock()
	if tier, ok := defaultTiersWindows[name]; ok {
		return name, tier, nil
	}
	return name, TierFull, nil
}

func (p *windowsPlatform) GrantedApplications() ([]string, error) {
	ensureGrantedLoaded()
	seen := map[string]struct{}{}
	for app := range defaultTiersWindows {
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

func (p *windowsPlatform) RequestAccess(apps []string) (map[string]AccessTier, error) {
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

// silence unused-import warning when only some symbols are referenced
// (syscall is used via the windows package internally).
var _ = syscall.Handle(0)
