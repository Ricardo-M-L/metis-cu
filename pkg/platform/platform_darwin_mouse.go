//go:build darwin

package platform

// Pointer events + cursor query. Implementation talks to the macOS
// CGEvent stream via robotgo so we get genuine OS-level synthetic
// events (the same path Anthropic's reference computer-use uses) rather
// than rolling raw CGEventCreateMouseEvent calls and risking an axis /
// button-mask drift bug per OS update.
//
// Coordinate system: top-left origin, logical (CSS) pixels — matches
// Anthropic's wire format. robotgo's Move() handles Retina scaling
// when robotgo.Scale is true; we leave that at its default so positions
// match what screenshot.go reports.

import (
	"github.com/go-vgo/robotgo"
)

// CursorPosition returns the current cursor location in logical pixels
// from the top-left of the active virtual desktop.
func (p *darwinPlatform) CursorPosition() (Point, error) {
	x, y := robotgo.Location()
	return Point{X: x, Y: y}, nil
}

// MouseMove warps the cursor to the supplied point. robotgo.Move is
// instantaneous (no smoothing) — for human-like motion use MouseDrag
// which calls MoveSmooth internally.
func (p *darwinPlatform) MouseMove(pt Point) error {
	robotgo.Move(pt.X, pt.Y)
	return nil
}

// MouseClick moves to pt, then issues count clicks of btn. count==2
// uses robotgo's native double-click path (a single CGEvent burst the
// OS recognizes as a double-click); count==3 falls back to three
// single clicks paced ~50ms apart to land within the system
// double-click interval and trigger a triple-click select.
func (p *darwinPlatform) MouseClick(pt Point, btn Button, count int) error {
	robotgo.Move(pt.X, pt.Y)
	name := buttonString(btn)
	switch {
	case count <= 1:
		return robotgo.Click(name, false)
	case count == 2:
		return robotgo.Click(name, true)
	default:
		// Triple-click (or higher): emit single clicks back-to-back.
		// Sleep is short enough to stay within the OS multi-click
		// window but long enough that each event is processed.
		for i := 0; i < count; i++ {
			if err := robotgo.Click(name, false); err != nil {
				return err
			}
			if i < count-1 {
				robotgo.MilliSleep(50)
			}
		}
		return nil
	}
}

// MouseDown moves to pt and presses btn without releasing. Pair with
// MouseUp for caller-driven drags (the `left_mouse_down` /
// `left_mouse_up` tools).
func (p *darwinPlatform) MouseDown(pt Point, btn Button) error {
	robotgo.Move(pt.X, pt.Y)
	return robotgo.Toggle(buttonString(btn), "down")
}

// MouseUp releases btn at the current cursor location. The pt argument
// is honored (we move first) so callers that want a precise drag
// endpoint can supply it; pass the same coordinates as the prior
// MouseDown if that's irrelevant.
func (p *darwinPlatform) MouseUp(pt Point, btn Button) error {
	robotgo.Move(pt.X, pt.Y)
	return robotgo.Toggle(buttonString(btn), "up")
}

// MouseDrag presses btn at `from`, smoothly drags to `to`, and
// releases. MoveSmooth's 1.0/1.0 low/high makes the motion brisk but
// non-instant so apps that distinguish drag from teleport (e.g.
// canvas-based editors) treat it as a real drag.
func (p *darwinPlatform) MouseDrag(from, to Point, btn Button) error {
	name := buttonString(btn)
	robotgo.Move(from.X, from.Y)
	if err := robotgo.Toggle(name, "down"); err != nil {
		return err
	}
	robotgo.MoveSmooth(to.X, to.Y, 1.0, 1.0)
	return robotgo.Toggle(name, "up")
}

// Scroll moves to pt and emits dx horizontal / dy vertical wheel
// ticks. Per Anthropic's spec positive dy scrolls *down*, which
// matches robotgo's convention on macOS (CGEvent natural-scroll is
// not respected — these are raw wheel ticks).
func (p *darwinPlatform) Scroll(pt Point, dx, dy int) error {
	robotgo.Move(pt.X, pt.Y)
	robotgo.Scroll(dx, dy)
	return nil
}

// buttonString maps our typed Button enum onto robotgo's stringly-
// typed button names. Unknown values default to "left" — the safest
// fallback, since left-click is a no-op on most surfaces if the
// caller meant something else, whereas right-click triggers context
// menus that are hard to dismiss programmatically.
func buttonString(b Button) string {
	switch b {
	case ButtonRight:
		return "right"
	case ButtonMiddle:
		return "center"
	case ButtonLeft:
		return "left"
	default:
		return "left"
	}
}
