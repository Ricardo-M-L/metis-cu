//go:build linux

package platform

// Linux pointer events. robotgo wraps libxtst (XTEST extension) on X11
// for synthetic input — same call shape as darwin so the body of every
// method is a near-copy of platform_darwin_mouse.go. We intentionally
// duplicate rather than embed a shared type so future Wayland-specific
// branches stay co-located with their build tag.

import (
	"context"
	"fmt"
	"runtime"

	"github.com/go-vgo/robotgo"
)

func (p *linuxPlatform) CursorPosition() (Point, error) {
	x, y := robotgo.Location()
	return Point{X: x, Y: y}, nil
}

func (p *linuxPlatform) MouseMove(pt Point) error {
	robotgo.Move(pt.X, pt.Y)
	return nil
}

// MouseClick: parity with darwin (BUG-14). Three plain singles 50ms
// apart was unreliable for triple-click "select line" gestures across
// Linux desktops because the target widgets check XInput's
// detail.click_count field, not just timestamps. robotgo.MultiClick
// routes through the underlying XTEST helper which sets that field
// for us. Single-click stays on the plain Click path because
// MultiClick adds an unwanted MouseSleep even for count=1.
func (p *linuxPlatform) MouseClick(pt Point, btn Button, count int) error {
	robotgo.Move(pt.X, pt.Y)
	name := buttonString(btn)
	if count <= 1 {
		return robotgo.Click(name, false)
	}
	return robotgo.MultiClick(name, count)
}

// MouseClickWithModifiers: see darwin twin for the deferred-release
// rationale (BUG-21). On Linux "cmd" is translated to "ctrl" via
// translatePrimaryModifier so an LLM trained on macOS-style "cmd-click"
// gets the Ctrl-click semantics it expects.
func (p *linuxPlatform) MouseClickWithModifiers(pt Point, btn Button, count int, mods []string) error {
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

func (p *linuxPlatform) MouseDown(pt Point, btn Button) error {
	robotgo.Move(pt.X, pt.Y)
	return robotgo.Toggle(buttonString(btn), "down")
}

func (p *linuxPlatform) MouseUp(pt Point, btn Button) error {
	robotgo.Move(pt.X, pt.Y)
	return robotgo.Toggle(buttonString(btn), "up")
}

// MouseDrag: see darwin twin for ctx + smooth-speed rationale (DD-2/DD-3).
func (p *linuxPlatform) MouseDrag(ctx context.Context, from, to Point, btn Button) error {
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

// Scroll: see scroll_signs.go — robotgo's underlying XTEST wrapper
// uses positive y = scroll UP / positive x = scroll LEFT, opposite
// to the wire spec.
func (p *linuxPlatform) Scroll(pt Point, dx, dy int) error {
	robotgo.Move(pt.X, pt.Y)
	rx, ry := scrollSigns(dx, dy)
	robotgo.Scroll(rx, ry)
	return nil
}

// ScrollWithModifiers: see darwin twin for the deferred-release
// rationale (BUG-21). On Linux "cmd" → "ctrl" via translatePrimaryModifier.
func (p *linuxPlatform) ScrollWithModifiers(pt Point, dx, dy int, mods []string) error {
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
		mt := translatePrimaryModifier(m, runtime.GOOS)
		if err := robotgo.KeyToggle(mt, "down"); err != nil {
			return fmt.Errorf("scroll-modifier KeyToggle down %q: %w", mt, err)
		}
		pressed = append(pressed, mt)
	}
	return p.Scroll(pt, dx, dy)
}

// Per-OS buttonString helpers were merged into the shared
// button_strings.go (DD-7).
