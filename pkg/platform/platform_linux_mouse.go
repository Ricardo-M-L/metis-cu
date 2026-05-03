//go:build linux

package platform

// Linux pointer events. robotgo wraps libxtst (XTEST extension) on X11
// for synthetic input — same call shape as darwin so the body of every
// method is a near-copy of platform_darwin_mouse.go. We intentionally
// duplicate rather than embed a shared type so future Wayland-specific
// branches stay co-located with their build tag.

import (
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

func (p *linuxPlatform) MouseClick(pt Point, btn Button, count int) error {
	robotgo.Move(pt.X, pt.Y)
	name := buttonStringLinux(btn)
	switch {
	case count <= 1:
		return robotgo.Click(name, false)
	case count == 2:
		return robotgo.Click(name, true)
	default:
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

func (p *linuxPlatform) MouseDown(pt Point, btn Button) error {
	robotgo.Move(pt.X, pt.Y)
	return robotgo.Toggle(buttonStringLinux(btn), "down")
}

func (p *linuxPlatform) MouseUp(pt Point, btn Button) error {
	robotgo.Move(pt.X, pt.Y)
	return robotgo.Toggle(buttonStringLinux(btn), "up")
}

func (p *linuxPlatform) MouseDrag(from, to Point, btn Button) error {
	name := buttonStringLinux(btn)
	robotgo.Move(from.X, from.Y)
	if err := robotgo.Toggle(name, "down"); err != nil {
		return err
	}
	robotgo.MoveSmooth(to.X, to.Y, 1.0, 1.0)
	return robotgo.Toggle(name, "up")
}

func (p *linuxPlatform) Scroll(pt Point, dx, dy int) error {
	robotgo.Move(pt.X, pt.Y)
	robotgo.Scroll(dx, dy)
	return nil
}

// buttonStringLinux mirrors the darwin helper but keeps a separate
// definition so build-tag scoping stays clean (each OS file is self-
// contained in its own build).
func buttonStringLinux(b Button) string {
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
