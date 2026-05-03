//go:build !darwin && !linux && !windows

package platform

import "image"

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

// mouse
func (p *stubPlatform) MouseMove(pt Point) error                         { return ErrNotImplemented }
func (p *stubPlatform) MouseClick(pt Point, btn Button, count int) error { return ErrNotImplemented }
func (p *stubPlatform) MouseDown(pt Point, btn Button) error             { return ErrNotImplemented }
func (p *stubPlatform) MouseUp(pt Point, btn Button) error               { return ErrNotImplemented }
func (p *stubPlatform) MouseDrag(from, to Point, btn Button) error       { return ErrNotImplemented }
func (p *stubPlatform) Scroll(pt Point, dx, dy int) error                { return ErrNotImplemented }

// keyboard
func (p *stubPlatform) KeyPress(combo string) error        { return ErrNotImplemented }
func (p *stubPlatform) KeyHold(combo string, ms int) error { return ErrNotImplemented }
func (p *stubPlatform) Type(text string) error             { return ErrNotImplemented }

// clipboard
func (p *stubPlatform) ClipboardRead() (string, error)   { return "", ErrNotImplemented }
func (p *stubPlatform) ClipboardWrite(text string) error { return ErrNotImplemented }

// application
func (p *stubPlatform) OpenApplication(name string) error      { return ErrNotImplemented }
func (p *stubPlatform) GrantedApplications() ([]string, error) { return nil, ErrNotImplemented }
func (p *stubPlatform) RequestAccess(apps []string) (map[string]AccessTier, error) {
	return nil, ErrNotImplemented
}
func (p *stubPlatform) FrontmostApp() (string, AccessTier, error) {
	return "", TierFull, ErrNotImplemented
}
