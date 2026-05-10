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
	"context"
	"fmt"

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

// MouseClick moves to pt, then issues a click of btn at the given
// multiplicity. macOS apps look at the CGEvent's
// kCGMouseEventClickState field — NOT the time-delta between separate
// clicks — to decide whether a click is a double / triple. Emitting
// three plain single clicks 50ms apart (the old behaviour) made
// TextEdit / Safari / Notes treat them as three independent clicks
// rather than a triple-click "select line" gesture.
//
// robotgo.MultiClick on darwin routes through the C doubleClick
// helper which calls CGEventSetIntegerValueField(kCGMouseEventClickState,
// count) — exactly the bit the OS checks. count=1 stays on the plain
// single-click path because MultiClick adds an unwanted MouseSleep
// even for a single click.
func (p *darwinPlatform) MouseClick(pt Point, btn Button, count int) error {
	robotgo.Move(pt.X, pt.Y)
	name := buttonString(btn)
	if count <= 1 {
		return robotgo.Click(name, false)
	}
	return robotgo.MultiClick(name, count)
}

// MouseClickWithModifiers presses each modifier (translated for the
// current OS via translatePrimaryModifier — "cmd" stays "cmd" on
// darwin), performs the click at multiplicity count, then releases
// the modifiers in reverse order. The release is deferred so a panic
// mid-click still leaves the keyboard in a clean state — without that,
// a synthetic Cmd-click that wedged would keep Cmd physically pressed
// in the OS until the user moves the real mouse (BUG-21).
func (p *darwinPlatform) MouseClickWithModifiers(pt Point, btn Button, count int, mods []string) error {
	if len(mods) == 0 {
		return p.MouseClick(pt, btn, count)
	}
	pressed := make([]string, 0, len(mods))
	defer func() {
		// Release in reverse order so chord teardown matches buildup.
		for i := len(pressed) - 1; i >= 0; i-- {
			_ = robotgo.KeyToggle(pressed[i], "up")
		}
	}()
	for _, m := range mods {
		mt := translatePrimaryModifier(m, "darwin")
		if err := robotgo.KeyToggle(mt, "down"); err != nil {
			return fmt.Errorf("modifier KeyToggle down %q: %w", mt, err)
		}
		pressed = append(pressed, mt)
	}
	return p.MouseClick(pt, btn, count)
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
// releases. MoveSmooth's smooth speed (mouseSmoothLow/High, configurable
// via DD-3) makes the motion non-instant so apps that distinguish drag
// from teleport (e.g. canvas-based editors) treat it as a real drag.
//
// ctx (DD-2): bail before sending input if already cancelled, and
// release the button if cancelled between move and up. We can't
// interrupt the blocking MoveSmooth mid-step — would need to chunk
// the drag — but bracketing it with ctx checks is enough to prevent
// the worst case (cancelled call still completing the full sequence).
func (p *darwinPlatform) MouseDrag(ctx context.Context, from, to Point, btn Button) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	name := buttonString(btn)
	robotgo.Move(from.X, from.Y)
	if err := robotgo.Toggle(name, "down"); err != nil {
		return err
	}
	defer func() {
		// Always release — even if MoveSmooth panicked or ctx fired
		// during it, we can't leave the button physically held.
		_ = robotgo.Toggle(name, "up")
	}()
	low, high := mouseSmoothSpeed()
	robotgo.MoveSmooth(to.X, to.Y, low, high)
	return ctx.Err()
}

// Scroll moves to pt and emits dx horizontal / dy vertical wheel
// ticks. Per Anthropic's spec positive dy scrolls *down* — but
// robotgo.Scroll uses CGEvent's native convention where positive y
// scrolls *up* (and positive x scrolls *left*). scrollSigns flips both
// axes so the wire spec is honored regardless of OS natural-scroll
// setting (CGEvent ignores the user pref — these are raw wheel ticks).
func (p *darwinPlatform) Scroll(pt Point, dx, dy int) error {
	robotgo.Move(pt.X, pt.Y)
	rx, ry := scrollSigns(dx, dy)
	robotgo.Scroll(rx, ry)
	return nil
}

// ScrollWithModifiers brackets Scroll with hold/release of the named
// modifier keys — see MouseClickWithModifiers twin for the deferred-
// release rationale (BUG-21). Common idioms: ctrl+wheel = zoom in
// most apps, shift+wheel = horizontal scroll, alt+wheel = step-by-
// pixel in some image editors.
func (p *darwinPlatform) ScrollWithModifiers(pt Point, dx, dy int, mods []string) error {
	if len(mods) == 0 {
		return p.Scroll(pt, dx, dy)
	}
	pressed := make([]string, 0, len(mods))
	defer func() {
		for i := len(pressed) - 1; i >= 0; i-- {
			_ = robotgo.KeyToggle(pressed[i], "up")
		}
	}()
	for _, m := range mods {
		mt := translatePrimaryModifier(m, "darwin")
		if err := robotgo.KeyToggle(mt, "down"); err != nil {
			return fmt.Errorf("scroll-modifier KeyToggle down %q: %w", mt, err)
		}
		pressed = append(pressed, mt)
	}
	return p.Scroll(pt, dx, dy)
}

// buttonString lives in button_strings.go (no build tag) — DD-7
// dedupe of identical helpers across darwin / linux / windows.
