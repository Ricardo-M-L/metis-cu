//go:build darwin

package platform

import (
	"fmt"
	"image"

	"github.com/kbinani/screenshot"
)

// macOS implementation lives in five sibling files (split for parallel
// authorship — each Sprint 2 sub-task owns one file):
//   - platform_darwin_core.go       (this) — vision + lifecycle
//   - platform_darwin_mouse.go      — pointer events
//   - platform_darwin_keyboard.go   — key events + text input
//   - platform_darwin_clipboard.go  — NSPasteboard read/write
//   - platform_darwin_app.go        — NSWorkspace open + frontmost app
//
// Real implementations land per-file as PRs merge; methods left
// stubbed return ErrNotImplemented so MCP clients see a clear
// "not implemented" error rather than a panic or silent zero.

type darwinPlatform struct {
	activeDisplay int
}

// New returns the platform implementation for the current GOOS.
func New() (Platform, error) { return &darwinPlatform{activeDisplay: 0}, nil }

func (p *darwinPlatform) Close() error { return nil }

// Screenshot captures the current active display via macOS CGImage and
// returns it normalised to LOGICAL pixels (the same coordinate space
// MouseClick / MouseMove operate in). On a Retina (2x) display
// ScreenCaptureKit returns a 2880x1800 image even though the CG bounds
// are 1440x900 logical — without normalisation the model would read
// coordinates from a 2880-wide PNG and emit clicks that land at 2x the
// intended position (BUG-7). The downsample below collapses that
// mismatch so caller-side coords always match what the user sees.
//
// Errors when display index is out of range or screen-capture access
// has not been granted by the user (System Settings → Privacy &
// Security → Screen Recording).
func (p *darwinPlatform) Screenshot() (image.Image, error) {
	n := screenshot.NumActiveDisplays()
	if n <= 0 {
		return nil, fmt.Errorf("no active displays found (screen capture access may be required)")
	}
	idx := p.activeDisplay
	if idx < 0 || idx >= n {
		idx = 0
	}
	bounds := screenshot.GetDisplayBounds(idx) // logical px
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, fmt.Errorf("capture display %d: %w", idx, err)
	}
	return normaliseToLogical(img, bounds.Dx(), bounds.Dy()), nil
}

// DisplayCount returns the number of attached active displays. Required for
// SwitchDisplay range validation and for the MCP `switch_display` tool's
// "list available monitors" branch.
func (p *darwinPlatform) DisplayCount() (int, error) {
	n := screenshot.NumActiveDisplays()
	if n <= 0 {
		return 0, fmt.Errorf("no active displays found (screen capture access may be required)")
	}
	return n, nil
}

// SwitchDisplay sets which display Screenshot will capture next. Index is
// 0-based; use DisplayCount to enumerate. Validates against active count
// rather than silently clamping so the caller can surface the error.
func (p *darwinPlatform) SwitchDisplay(idx int) error {
	n := screenshot.NumActiveDisplays()
	if idx < 0 || idx >= n {
		return fmt.Errorf("display %d out of range (have %d)", idx, n)
	}
	p.activeDisplay = idx
	return nil
}

// DisplayBounds returns the logical-pixel rectangle of display idx.
// Backed by kbinani/screenshot.GetDisplayBounds — same source the
// Screenshot() path uses, so the values match the canvas clicks land in.
func (p *darwinPlatform) DisplayBounds(idx int) (image.Rectangle, error) {
	n := screenshot.NumActiveDisplays()
	if n <= 0 {
		return image.Rectangle{}, fmt.Errorf("no active displays found")
	}
	if idx < 0 || idx >= n {
		return image.Rectangle{}, fmt.Errorf("display %d out of range (have %d)", idx, n)
	}
	return screenshot.GetDisplayBounds(idx), nil
}
