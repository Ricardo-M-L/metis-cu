//go:build windows

package platform

// Windows pointer events. robotgo wraps mouse_event / SendInput on
// Windows. Same call shape as darwin / linux.

import (
	"context"
	"fmt"
	"runtime"

	"github.com/go-vgo/robotgo"
)

func (p *windowsPlatform) CursorPosition() (Point, error) {
	x, y := robotgo.Location()
	return Point{X: x, Y: y}, nil
}

func (p *windowsPlatform) MouseMove(pt Point) error {
	robotgo.Move(pt.X, pt.Y)
	return nil
}

// MouseClick: parity with darwin (BUG-14). robotgo.MultiClick on
// Windows routes through SendInput with the click multiplicity
// encoded in the message stream, which Notepad / VS Code / Edge
// inspect when deciding whether to treat back-to-back clicks as a
// triple. Single-click stays on the plain Click path to avoid the
// extra MouseSleep MultiClick adds.
func (p *windowsPlatform) MouseClick(pt Point, btn Button, count int) error {
	robotgo.Move(pt.X, pt.Y)
	name := buttonString(btn)
	if count <= 1 {
		return robotgo.Click(name, false)
	}
	return robotgo.MultiClick(name, count)
}

// MouseClickWithModifiers: see darwin twin for the deferred-release
// rationale (BUG-21). On Windows "cmd" is translated to "ctrl".
func (p *windowsPlatform) MouseClickWithModifiers(pt Point, btn Button, count int, mods []string) error {
	if len(mods) == 0 {
		return p.MouseClick(pt, btn, count)
	}
	pressed := make([]string, 0, len(mods))
	defer func() {
		for i := len(pressed) - 1; i >= 0; i-- {
			_ = robotgo.KeyToggle(pressed[i], "up")
		}
	}()
	for _, m := range mods {
		mt := translatePrimaryModifier(m, runtime.GOOS)
		if err := robotgo.KeyToggle(mt, "down"); err != nil {
			return fmt.Errorf("modifier KeyToggle down %q: %w", mt, err)
		}
		pressed = append(pressed, mt)
	}
	return p.MouseClick(pt, btn, count)
}

func (p *windowsPlatform) MouseDown(pt Point, btn Button) error {
	robotgo.Move(pt.X, pt.Y)
	return robotgo.Toggle(buttonString(btn), "down")
}

func (p *windowsPlatform) MouseUp(pt Point, btn Button) error {
	robotgo.Move(pt.X, pt.Y)
	return robotgo.Toggle(buttonString(btn), "up")
}

// MouseDrag: see darwin twin for ctx + smooth-speed rationale (DD-2/DD-3).
func (p *windowsPlatform) MouseDrag(ctx context.Context, from, to Point, btn Button) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	name := buttonString(btn)
	robotgo.Move(from.X, from.Y)
	if err := robotgo.Toggle(name, "down"); err != nil {
		return err
	}
	defer func() { _ = robotgo.Toggle(name, "up") }()
	low, high := mouseSmoothSpeed()
	robotgo.MoveSmooth(to.X, to.Y, low, high)
	return ctx.Err()
}

// Scroll: see scroll_signs.go — robotgo's underlying SendInput wheel
// path uses positive y = scroll UP / positive x = scroll LEFT,
// opposite to the wire spec.
func (p *windowsPlatform) Scroll(pt Point, dx, dy int) error {
	robotgo.Move(pt.X, pt.Y)
	rx, ry := scrollSigns(dx, dy)
	robotgo.Scroll(rx, ry)
	return nil
}

// Per-OS buttonString helpers were merged into the shared
// button_strings.go (DD-7).
