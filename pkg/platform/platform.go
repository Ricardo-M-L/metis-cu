// Package platform abstracts OS-specific GUI control behind one
// interface. Every tool in pkg/tools dispatches through Platform — no
// OS imports leak into tool handlers.
//
// Per-platform implementations live in platform_<goos>.go and are
// selected by build constraints. Stubs return ErrNotImplemented so the
// scaffolding compiles on every host even before Sprint 2/3/4 land
// the real macOS / Linux / Windows code.
package platform

import (
	"errors"
	"image"
)

// ErrNotImplemented is returned by stubs while real platform code is
// pending. Tool handlers surface it verbatim so the MCP client sees
// "not implemented on this platform" instead of a silent zero-value.
var ErrNotImplemented = errors.New("not implemented on this platform")

// Point is screen-relative, origin top-left, in CSS-style logical
// pixels (so DPI scaling on Retina + Hi-DPI Linux is handled by the
// platform layer, not callers). Anthropic's spec uses the same shape.
type Point struct {
	X, Y int
}

// Button is the mouse button the call refers to. Order matches X11
// conventions and Anthropic's tool params.
type Button int

const (
	ButtonLeft Button = iota + 1
	ButtonMiddle
	ButtonRight
)

// Modifier names mirror what `key` / `hold_key` accept on the wire.
// Lowercase keeps parsing trivial; combos use "+" join (e.g.
// "cmd+shift+a").
type Modifier string

const (
	ModCmd   Modifier = "cmd"
	ModCtrl  Modifier = "ctrl"
	ModAlt   Modifier = "alt"
	ModShift Modifier = "shift"
)

// AccessTier mirrors Claude Code's frontmost-app gating: browsers are
// "read" (visible but no input), terminals/IDEs are "click" (left-click
// only), everything else is "full". The gate is enforced in pkg/tools
// before dispatch — this enum is just the wire vocabulary.
type AccessTier string

const (
	TierRead  AccessTier = "read"
	TierClick AccessTier = "click"
	TierFull  AccessTier = "full"
)

// Platform is everything tools need. Methods stay coarse — one tool ≈
// one method — so adding a new platform doesn't require reverse-
// engineering an aggregation layer.
type Platform interface {
	Close() error

	// vision
	Screenshot() (image.Image, error)
	CursorPosition() (Point, error)
	DisplayCount() (int, error)
	SwitchDisplay(idx int) error

	// mouse
	MouseMove(p Point) error
	MouseClick(p Point, btn Button, count int) error // count: 1 single, 2 double, 3 triple
	MouseDown(p Point, btn Button) error
	MouseUp(p Point, btn Button) error
	MouseDrag(from, to Point, btn Button) error
	Scroll(p Point, dx, dy int) error

	// keyboard
	KeyPress(combo string) error // "cmd+a", "esc", "F11"
	KeyHold(combo string, ms int) error
	Type(text string) error

	// clipboard
	ClipboardRead() (string, error)
	ClipboardWrite(text string) error

	// application
	OpenApplication(name string) error
	GrantedApplications() ([]string, error)
	RequestAccess(apps []string) (map[string]AccessTier, error)

	// frontmost-app gate (used by tools to enforce tier)
	FrontmostApp() (string, AccessTier, error)
}
