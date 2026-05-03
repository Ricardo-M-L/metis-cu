//go:build windows

package platform

// Windows pointer events. robotgo wraps mouse_event / SendInput on
// Windows. Same call shape as darwin / linux.

import (
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

func (p *windowsPlatform) MouseClick(pt Point, btn Button, count int) error {
	robotgo.Move(pt.X, pt.Y)
	name := buttonStringWindows(btn)
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

func (p *windowsPlatform) MouseDown(pt Point, btn Button) error {
	robotgo.Move(pt.X, pt.Y)
	return robotgo.Toggle(buttonStringWindows(btn), "down")
}

func (p *windowsPlatform) MouseUp(pt Point, btn Button) error {
	robotgo.Move(pt.X, pt.Y)
	return robotgo.Toggle(buttonStringWindows(btn), "up")
}

func (p *windowsPlatform) MouseDrag(from, to Point, btn Button) error {
	name := buttonStringWindows(btn)
	robotgo.Move(from.X, from.Y)
	if err := robotgo.Toggle(name, "down"); err != nil {
		return err
	}
	robotgo.MoveSmooth(to.X, to.Y, 1.0, 1.0)
	return robotgo.Toggle(name, "up")
}

func (p *windowsPlatform) Scroll(pt Point, dx, dy int) error {
	robotgo.Move(pt.X, pt.Y)
	robotgo.Scroll(dx, dy)
	return nil
}

func buttonStringWindows(b Button) string {
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
