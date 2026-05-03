//go:build linux

package platform

import (
	"fmt"
	"image"

	"github.com/kbinani/screenshot"
)

// Linux implementation lives in five sibling files (mirroring the darwin
// split for parity):
//   - platform_linux_core.go      (this) — vision + lifecycle
//   - platform_linux_mouse.go     — pointer events via robotgo
//   - platform_linux_keyboard.go  — key events via robotgo
//   - platform_linux_clipboard.go — clipboard via golang.design/x/clipboard
//   - platform_linux_app.go       — X11 frontmost-app gate + launch
//
// Tooling assumptions:
//   - X11 session (Wayland under XWayland works; native Wayland frontmost
//     detection is not implemented — `swaymsg`/`hyprctl` would be needed).
//   - `xdotool` present in PATH for FrontmostApp (the most reliable cross-
//     desktop way to get the active window class without doing raw X11).
//   - `xdg-open` or `gtk-launch` present for OpenApplication.

type linuxPlatform struct {
	activeDisplay int
}

// New returns the platform implementation for the current GOOS.
func New() (Platform, error) { return &linuxPlatform{activeDisplay: 0}, nil }

func (p *linuxPlatform) Close() error { return nil }

// Screenshot captures the current active display via X11 (or Wayland's
// XWayland bridge). kbinani/screenshot uses XGetImage under the hood
// which works across most Linux desktops without extra permissions.
func (p *linuxPlatform) Screenshot() (image.Image, error) {
	n := screenshot.NumActiveDisplays()
	if n <= 0 {
		return nil, fmt.Errorf("no active displays found (X11 / DISPLAY env may be missing)")
	}
	idx := p.activeDisplay
	if idx < 0 || idx >= n {
		idx = 0
	}
	bounds := screenshot.GetDisplayBounds(idx)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, fmt.Errorf("capture display %d: %w", idx, err)
	}
	return img, nil
}

func (p *linuxPlatform) DisplayCount() (int, error) {
	n := screenshot.NumActiveDisplays()
	if n <= 0 {
		return 0, fmt.Errorf("no active displays found (X11 / DISPLAY env may be missing)")
	}
	return n, nil
}

func (p *linuxPlatform) SwitchDisplay(idx int) error {
	n := screenshot.NumActiveDisplays()
	if idx < 0 || idx >= n {
		return fmt.Errorf("display %d out of range (have %d)", idx, n)
	}
	p.activeDisplay = idx
	return nil
}
