//go:build !darwin && !linux && !windows

package platform

import (
	"context"
	"image"
)

// Stub fallback for unsupported OSes (freebsd / openbsd / dragonfly /
// solaris / etc.). darwin / linux / windows have full implementations
// in their own platform_<goos>_*.go files. New() succeeds so the binary
// still boots and reports "not implemented on this platform" per call,
// keeping CI / cross-builds happy on platforms we haven't certified.

type stubPlatform struct{}

// New returns the platform implementation for the current GOOS.
func New() (Platform, error) { return &stubPlatform{}, nil }

func (p *stubPlatform) Close() error { return nil }

// vision
func (p *stubPlatform) Screenshot() (image.Image, error) { return nil, ErrNotImplemented }
func (p *stubPlatform) CursorPosition() (Point, error)   { return Point{}, ErrNotImplemented }
func (p *stubPlatform) DisplayCount() (int, error)       { return 0, ErrNotImplemented }
func (p *stubPlatform) SwitchDisplay(idx int) error      { return ErrNotImplemented }
func (p *stubPlatform) DisplayBounds(idx int) (image.Rectangle, error) {
	return image.Rectangle{}, ErrNotImplemented
}

// mouse
func (p *stubPlatform) MouseMove(pt Point) error                         { return ErrNotImplemented }
func (p *stubPlatform) MouseClick(pt Point, btn Button, count int) error { return ErrNotImplemented }
func (p *stubPlatform) MouseClickWithModifiers(pt Point, btn Button, count int, mods []string) error {
	return ErrNotImplemented
}
func (p *stubPlatform) MouseDown(pt Point, btn Button) error { return ErrNotImplemented }
func (p *stubPlatform) MouseUp(pt Point, btn Button) error   { return ErrNotImplemented }
func (p *stubPlatform) MouseDrag(ctx context.Context, from, to Point, btn Button) error {
	return ErrNotImplemented
}
func (p *stubPlatform) Scroll(pt Point, dx, dy int) error { return ErrNotImplemented }
func (p *stubPlatform) ScrollWithModifiers(pt Point, dx, dy int, mods []string) error {
	return ErrNotImplemented
}

// keyboard
func (p *stubPlatform) KeyPress(combo string) error { return ErrNotImplemented }
func (p *stubPlatform) KeyHold(ctx context.Context, combo string, ms int) error {
	return ErrNotImplemented
}
func (p *stubPlatform) Type(ctx context.Context, text string) error { return ErrNotImplemented }

// clipboard
func (p *stubPlatform) ClipboardRead() (string, error)             { return "", ErrNotImplemented }
func (p *stubPlatform) ClipboardWrite(text string) error           { return ErrNotImplemented }
func (p *stubPlatform) ClipboardSnapshot() ClipboardSnapshot       { return ClipboardSnapshot{Empty: true} }
func (p *stubPlatform) ClipboardRestore(s ClipboardSnapshot) error { return ErrNotImplemented }

// application
func (p *stubPlatform) OpenApplication(ctx context.Context, name string) error {
	return ErrNotImplemented
}
func (p *stubPlatform) GrantedApplications() ([]string, error) { return nil, ErrNotImplemented }
func (p *stubPlatform) RequestAccess(apps []string, tier AccessTier) (map[string]AccessTier, error) {
	return nil, ErrNotImplemented
}
func (p *stubPlatform) Tier(name string) AccessTier { return TierFull }
func (p *stubPlatform) FrontmostApp() (string, AccessTier, error) {
	return "", TierFull, ErrNotImplemented
}
func (p *stubPlatform) Confirm(message string) (bool, error) {
	return false, ErrNotImplemented
}

func (p *stubPlatform) OCR(_ image.Image) ([]OCRResult, error) {
	return nil, ErrNotImplemented
}
