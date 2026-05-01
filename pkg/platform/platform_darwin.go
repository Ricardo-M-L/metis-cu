//go:build darwin

package platform

import "image"

// macOS implementation — Sprint 2 fills these in via:
//   - Screenshot       : screencapture (CGImage / kbinani/screenshot)
//   - Mouse / Keyboard : CGEventCreate + CGEventPost (robotgo bindings)
//   - Clipboard        : NSPasteboard (golang.design/x/clipboard)
//   - Open / Frontmost : NSWorkspace + LSCopyApplicationURLsForBundleIdentifier
//   - Access prompt    : AXIsProcessTrustedWithOptions (Accessibility) +
//                        CGRequestScreenCaptureAccess
//
// All are stubs returning ErrNotImplemented for now so the rest of the
// repo compiles and the API surface is fixed.

type darwinPlatform struct{}

// New returns the platform implementation for the current GOOS.
func New() (Platform, error) { return &darwinPlatform{}, nil }

func (p *darwinPlatform) Close() error { return nil }

// vision
func (p *darwinPlatform) Screenshot() (image.Image, error) { return nil, ErrNotImplemented }
func (p *darwinPlatform) CursorPosition() (Point, error)   { return Point{}, ErrNotImplemented }
func (p *darwinPlatform) DisplayCount() (int, error)       { return 0, ErrNotImplemented }
func (p *darwinPlatform) SwitchDisplay(idx int) error      { return ErrNotImplemented }

// mouse
func (p *darwinPlatform) MouseMove(pt Point) error                       { return ErrNotImplemented }
func (p *darwinPlatform) MouseClick(pt Point, btn Button, count int) error { return ErrNotImplemented }
func (p *darwinPlatform) MouseDown(pt Point, btn Button) error           { return ErrNotImplemented }
func (p *darwinPlatform) MouseUp(pt Point, btn Button) error             { return ErrNotImplemented }
func (p *darwinPlatform) MouseDrag(from, to Point, btn Button) error     { return ErrNotImplemented }
func (p *darwinPlatform) Scroll(pt Point, dx, dy int) error              { return ErrNotImplemented }

// keyboard
func (p *darwinPlatform) KeyPress(combo string) error            { return ErrNotImplemented }
func (p *darwinPlatform) KeyHold(combo string, ms int) error     { return ErrNotImplemented }
func (p *darwinPlatform) Type(text string) error                 { return ErrNotImplemented }

// clipboard
func (p *darwinPlatform) ClipboardRead() (string, error)         { return "", ErrNotImplemented }
func (p *darwinPlatform) ClipboardWrite(text string) error       { return ErrNotImplemented }

// application
func (p *darwinPlatform) OpenApplication(name string) error                                { return ErrNotImplemented }
func (p *darwinPlatform) GrantedApplications() ([]string, error)                           { return nil, ErrNotImplemented }
func (p *darwinPlatform) RequestAccess(apps []string) (map[string]AccessTier, error)       { return nil, ErrNotImplemented }
func (p *darwinPlatform) FrontmostApp() (string, AccessTier, error)                        { return "", TierFull, ErrNotImplemented }
