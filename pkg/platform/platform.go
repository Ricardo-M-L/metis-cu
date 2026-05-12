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
	"context"
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

// ClipboardSnapshot is the opaque payload returned by
// ClipboardSnapshot and consumed by ClipboardRestore. Currently
// captures only text — non-text formats (image, file URL) round-trip
// as Empty=true so Restore is a no-op rather than risking corruption.
// Future: add Image []byte / Files []string fields when the underlying
// clipboard library exposes them.
type ClipboardSnapshot struct {
	Text  string
	Empty bool // true when the source clipboard had no readable text
}

// WindowInfo describes one on-screen window — the per-OS enumerators
// (CGWindowListCopyWindowInfo on darwin, wmctrl/X11 on linux, Win32
// EnumWindows on windows) return a list of these. Bounds are in
// LOGICAL pixels in the virtual desktop coordinate space, matching
// the Screenshot / MouseMove convention.
//
// Title may be empty for chromeless / utility windows. App is the
// process display name — same shape FrontmostApp returns. Bounds
// reported as image.Rectangle for symmetry with DisplayBounds.
type WindowInfo struct {
	App    string          `json:"app"`
	Title  string          `json:"title"`
	Bounds image.Rectangle `json:"bounds"`
}

// OCRResult is one recognised text region from a Platform.OCR call.
// Bounds are pixel coordinates in the SAME space as the input image
// (caller-side scaling to logical px is the caller's job — typically
// no-op since metis-cu screenshots are already in logical px).
// Confidence ranges 0.0..1.0 where the engine reports it; 0 means
// "not provided by this engine" rather than "definitely wrong".
type OCRResult struct {
	Text       string
	Bounds     image.Rectangle
	Confidence float64
}

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
	// DisplayBounds returns the LOGICAL pixel rectangle of display
	// `idx` (0-based), origin at the global multi-monitor virtual
	// coordinate space. Used by the `screen_size` tool to tell the
	// model the canvas it should be emitting clicks into — matches
	// Anthropic's `display_width_px` / `display_height_px` contract.
	// Out-of-range idx returns ErrNotImplemented (or a wrapped errno
	// per OS); never panics.
	DisplayBounds(idx int) (image.Rectangle, error)

	// mouse
	MouseMove(p Point) error
	MouseClick(p Point, btn Button, count int) error // count: 1 single, 2 double, 3 triple
	// MouseClickWithModifiers presses each modifier (cmd/ctrl/alt/shift),
	// performs the click at the given multiplicity, then releases the
	// modifiers — atomic at the platform layer so a panic mid-click can
	// still release modifiers (BUG-21). Empty mods slice equals plain
	// MouseClick. Cross-platform "cmd" → "ctrl" translation lives in
	// keycombo.go's translatePrimaryModifier so an LLM trained on
	// macOS-style "cmd-click" works on Linux/Windows too.
	MouseClickWithModifiers(p Point, btn Button, count int, mods []string) error
	MouseDown(p Point, btn Button) error
	MouseUp(p Point, btn Button) error
	// MouseDrag is bounded by ctx so a cancellation surfaces a
	// ctx.Err — currently a "best-effort" check at entry/exit since
	// robotgo.MoveSmooth is a blocking C call we can't interrupt
	// mid-step. Future: chunk the drag into ctx-checked sub-moves.
	MouseDrag(ctx context.Context, from, to Point, btn Button) error
	Scroll(p Point, dx, dy int) error
	// ScrollWithModifiers is Scroll bracketed by holding the named
	// modifier keys (e.g. "ctrl" for ctrl+wheel = zoom in most apps,
	// "shift" for horizontal scroll, "cmd" for app-specific). Empty
	// mods = plain Scroll. Modifier release is deferred so a panic
	// mid-scroll still leaves the keyboard in a clean state, mirroring
	// the MouseClickWithModifiers safety contract (BUG-21 pattern).
	// Cross-platform "cmd" → "ctrl" translation lives in keycombo.go's
	// translatePrimaryModifier so an LLM trained on macOS-style
	// "cmd-scroll" works on Linux/Windows too.
	ScrollWithModifiers(p Point, dx, dy int, mods []string) error

	// keyboard
	KeyPress(combo string) error // "cmd+a", "esc", "F11"
	// KeyHold presses combo, waits for ms milliseconds OR ctx
	// cancellation, then releases. Cancellation guarantees the
	// release fires (DD-2) so a SIGTERM mid-hold doesn't leave
	// modifiers physically pressed in the OS.
	KeyHold(ctx context.Context, combo string, ms int) error
	// Type sends text. Bounded by ctx — bails with ctx.Err if
	// already cancelled at entry. The paste-fallback path used for
	// long strings (BUG-22) checks ctx between snapshot/write/paste/
	// restore steps so a cancel cleans up without stranding text on
	// the user's clipboard.
	Type(ctx context.Context, text string) error

	// clipboard
	ClipboardRead() (string, error)
	ClipboardWrite(text string) error
	// ClipboardSnapshot captures the current clipboard payload as
	// opaque bytes so a tool that needs to mutate the clipboard
	// (e.g. paste-via-type fallback in BUG-22) can restore the
	// user's prior content. The returned snapshot type is platform-
	// internal — pass it back verbatim to ClipboardRestore.
	ClipboardSnapshot() ClipboardSnapshot
	// ClipboardRestore writes the snapshot back. Returns nil if the
	// snapshot is empty / unsupported on this OS — restoration is
	// best-effort, since for example a non-text payload that we
	// can't introspect should simply be left as-is when the
	// alternative is corrupting it.
	ClipboardRestore(s ClipboardSnapshot) error

	// application
	// OpenApplication launches the named app. ctx caps the wait —
	// `open -a` (macOS) and `xdg-open` (Linux) can block 10s+ on
	// cold launches, so a cancel from upstream lets the MCP request
	// abort instead of waiting out the full subprocess timeout.
	OpenApplication(ctx context.Context, name string) error
	GrantedApplications() ([]string, error)
	// RequestAccess records each app at the given tier and persists
	// the result. tier defaults to TierFull when callers pass an
	// empty / unknown tier — preserves the historical behaviour of
	// the single-arg form before BUG-20 added per-app tier choice.
	RequestAccess(apps []string, tier AccessTier) (map[string]AccessTier, error)
	// Tier returns the tier the platform would assign to `name`.
	// Consults the persisted grants first, then the per-platform
	// defaults, and falls through to TierFull. Never errors — an
	// unknown app is a TierFull app.
	Tier(name string) AccessTier

	// OCR runs an OS-native (or shelled-out) text recognizer over img
	// and returns one OCRResult per recognised region. Empty slice
	// means "no text found"; ErrNotImplemented means the platform has
	// no OCR backend configured. Callers MUST tolerate the
	// ErrNotImplemented case — OCR is an opt-in capability today
	// (Linux: tesseract via PATH; darwin/windows: stubs pending).
	// The image is single-frame and assumed to be the active screenshot
	// in logical pixels. Larger inputs trade latency for recall; the
	// caller's job (not the platform's) to crop / downsample first.
	OCR(img image.Image) ([]OCRResult, error)

	// ListWindows enumerates every visible top-level window across
	// every process. Pragmatic subset of full accessibility-tree
	// exposure — gives the model "what's on screen" without needing
	// a11y permissions or the per-OS accessibility bus.
	// ErrNotImplemented on platforms where the enumerator isn't wired
	// (currently windows; mac uses osascript, linux uses wmctrl).
	ListWindows() ([]WindowInfo, error)

	// frontmost-app gate (used by tools to enforce tier)
	FrontmostApp() (string, AccessTier, error)

	// Confirm pops a synchronous OS-native confirmation dialog and
	// blocks until the user responds. Returns true when the user
	// approves, false when they deny / cancel / let the dialog time
	// out. Errors are reserved for hard failures (no dialog mechanism
	// available, dispatcher unreachable). Used by request_access so an
	// MCP client can't silently self-grant; future tools that need a
	// human decision (e.g. "send file to remote") should reuse this.
	Confirm(message string) (bool, error)
}
