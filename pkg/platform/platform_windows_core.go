//go:build windows

package platform

import (
	"fmt"
	"image"

	"github.com/kbinani/screenshot"
)

// Windows implementation lives in five sibling files (mirroring the
// darwin / linux split):
//   - platform_windows_core.go      (this) — vision + lifecycle
//   - platform_windows_mouse.go     — pointer events via robotgo
//   - platform_windows_keyboard.go  — key events via robotgo
//   - platform_windows_clipboard.go — clipboard via golang.design/x/clipboard
//   - platform_windows_app.go       — Win32 frontmost app + ShellExecute
//
// Vision uses kbinani/screenshot which wraps GDI BitBlt on Windows.
// MouseFin uses robotgo (mouse_event / SendInput). Clipboard uses the
// cross-platform golang.design/x/clipboard.

type windowsPlatform struct {
	activeDisplay int
}

// New returns the platform implementation for the current GOOS.
func New() (Platform, error) { return &windowsPlatform{activeDisplay: 0}, nil }

func (p *windowsPlatform) Close() error { return nil }

func (p *windowsPlatform) Screenshot() (image.Image, error) {
	n := screenshot.NumActiveDisplays()
	if n <= 0 {
		return nil, fmt.Errorf("no active displays found")
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

func (p *windowsPlatform) DisplayCount() (int, error) {
	n := screenshot.NumActiveDisplays()
	if n <= 0 {
		return 0, fmt.Errorf("no active displays found")
	}
	return n, nil
}

func (p *windowsPlatform) SwitchDisplay(idx int) error {
	n := screenshot.NumActiveDisplays()
	if idx < 0 || idx >= n {
		return fmt.Errorf("display %d out of range (have %d)", idx, n)
	}
	p.activeDisplay = idx
	return nil
}
