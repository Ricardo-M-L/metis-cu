package tools

import (
	"context"
	"errors"
	"image"
	"strings"
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

// TestTierAllows is the table-driven heart of the gate: verifies the
// Read < Click < Full ordering matches the tier doc string in
// platform.go. Adding a new tier means extending this table.
func TestTierAllows(t *testing.T) {
	cases := []struct {
		have, required platform.AccessTier
		want           bool
	}{
		// read can satisfy only read
		{platform.TierRead, platform.TierRead, true},
		{platform.TierRead, platform.TierClick, false},
		{platform.TierRead, platform.TierFull, false},
		// click satisfies read + click
		{platform.TierClick, platform.TierRead, true},
		{platform.TierClick, platform.TierClick, true},
		{platform.TierClick, platform.TierFull, false},
		// full satisfies everything
		{platform.TierFull, platform.TierRead, true},
		{platform.TierFull, platform.TierClick, true},
		{platform.TierFull, platform.TierFull, true},
	}
	for _, tc := range cases {
		got := tierAllows(tc.have, tc.required)
		if got != tc.want {
			t.Errorf("tierAllows(have=%q, required=%q) = %v, want %v",
				tc.have, tc.required, got, tc.want)
		}
	}
}

// TestEnforceTier_DenyOnLookupFailure ensures the gate fails closed
// when FrontmostApp errors out (e.g. osascript denied, no GUI session).
// Adapters rely on this so a wedged frontmost-app lookup can't be
// silently bypassed.
func TestEnforceTier_DenyOnLookupFailure(t *testing.T) {
	p := &lookupErrPlat{err: errors.New("System Events not running")}
	msg, ok := EnforceTier(p, platform.TierClick)
	if ok {
		t.Fatal("EnforceTier should deny when FrontmostApp errors")
	}
	if !strings.Contains(msg, "frontmost app lookup failed") {
		t.Errorf("expected lookup-failure message, got %q", msg)
	}
}

// TestEnforceTier_DenyOnInsufficientTier verifies the standard deny
// path when the frontmost app's tier is below the required level.
func TestEnforceTier_DenyOnInsufficientTier(t *testing.T) {
	p := &fixedTierPlat{name: "Safari", tier: platform.TierRead}
	msg, ok := EnforceTier(p, platform.TierFull)
	if ok {
		t.Fatal("EnforceTier should deny TierRead app vs TierFull required")
	}
	if !strings.Contains(msg, "Safari") {
		t.Errorf("expected app name in deny message, got %q", msg)
	}
	if !strings.Contains(msg, "read") {
		t.Errorf("expected current tier in deny message, got %q", msg)
	}
}

// TestEnforceTier_AllowOnSufficientTier verifies the happy path.
func TestEnforceTier_AllowOnSufficientTier(t *testing.T) {
	p := &fixedTierPlat{name: "Calculator", tier: platform.TierFull}
	msg, ok := EnforceTier(p, platform.TierClick)
	if !ok {
		t.Fatalf("EnforceTier should allow TierFull app vs TierClick required; got %q", msg)
	}
	if msg != "" {
		t.Errorf("expected empty deny message when allowed, got %q", msg)
	}
}

// lookupErrPlat is a fake Platform whose FrontmostApp returns the
// configured error. All other methods return ErrNotImplemented so the
// gate test surface stays tightly scoped.
type lookupErrPlat struct {
	stubPlat
	err error
}

func (p *lookupErrPlat) FrontmostApp() (string, platform.AccessTier, error) {
	return "", platform.TierFull, p.err
}

// fixedTierPlat returns a fixed (name, tier) pair from FrontmostApp.
// Used to exercise the EnforceTier comparison branch without poking
// the real osascript path.
type fixedTierPlat struct {
	stubPlat
	name string
	tier platform.AccessTier
}

func (p *fixedTierPlat) FrontmostApp() (string, platform.AccessTier, error) {
	return p.name, p.tier, nil
}

// stubPlat is a default-deny Platform implementation used as the
// embedded base for tighter fakes. The 22-method interface would be
// noisy to inline in every test fake; embedding stubPlat keeps the
// fakes a single overridden method long.
//
// FrontmostApp is the one exception: it returns a TierFull stub-app
// rather than ErrNotImplemented, so handler tests that don't care
// about gating (the vast majority) auto-pass through gateOrDeny
// without each fake having to override FrontmostApp. Tests that DO
// want to exercise gate behavior override FrontmostApp explicitly
// (see fixedTierPlat / lookupErrPlat above and the gate-integration
// tests).
type stubPlat struct{}

func (stubPlat) Close() error                     { return platform.ErrNotImplemented }
func (stubPlat) Screenshot() (image.Image, error) { return nil, platform.ErrNotImplemented }
func (stubPlat) CursorPosition() (platform.Point, error) {
	return platform.Point{}, platform.ErrNotImplemented
}
func (stubPlat) DisplayCount() (int, error) { return 0, platform.ErrNotImplemented }
func (stubPlat) SwitchDisplay(int) error    { return platform.ErrNotImplemented }
func (stubPlat) DisplayBounds(int) (image.Rectangle, error) {
	return image.Rectangle{}, platform.ErrNotImplemented
}
func (stubPlat) MouseMove(platform.Point) error { return platform.ErrNotImplemented }
func (stubPlat) MouseClick(platform.Point, platform.Button, int) error {
	return platform.ErrNotImplemented
}
func (stubPlat) MouseClickWithModifiers(platform.Point, platform.Button, int, []string) error {
	return platform.ErrNotImplemented
}
func (stubPlat) MouseDown(platform.Point, platform.Button) error { return platform.ErrNotImplemented }
func (stubPlat) MouseUp(platform.Point, platform.Button) error   { return platform.ErrNotImplemented }
func (stubPlat) MouseDrag(context.Context, platform.Point, platform.Point, platform.Button) error {
	return platform.ErrNotImplemented
}
func (stubPlat) Scroll(platform.Point, int, int) error { return platform.ErrNotImplemented }
func (stubPlat) ScrollWithModifiers(platform.Point, int, int, []string) error {
	return platform.ErrNotImplemented
}
func (stubPlat) KeyPress(string) error                      { return platform.ErrNotImplemented }
func (stubPlat) KeyHold(context.Context, string, int) error { return platform.ErrNotImplemented }
func (stubPlat) Type(context.Context, string) error         { return platform.ErrNotImplemented }
func (stubPlat) ClipboardRead() (string, error)             { return "", platform.ErrNotImplemented }
func (stubPlat) ClipboardWrite(string) error                { return platform.ErrNotImplemented }
func (stubPlat) ClipboardSnapshot() platform.ClipboardSnapshot {
	return platform.ClipboardSnapshot{Empty: true}
}
func (stubPlat) ClipboardRestore(platform.ClipboardSnapshot) error { return nil }
func (stubPlat) OpenApplication(context.Context, string) error {
	return platform.ErrNotImplemented
}
func (stubPlat) GrantedApplications() ([]string, error) { return nil, platform.ErrNotImplemented }
func (stubPlat) RequestAccess([]string, platform.AccessTier) (map[string]platform.AccessTier, error) {
	return nil, platform.ErrNotImplemented
}
func (stubPlat) Tier(string) platform.AccessTier { return platform.TierFull }
func (stubPlat) FrontmostApp() (string, platform.AccessTier, error) {
	// Default-allow: pretend frontmost is a TierFull app so handler
	// tests pass through gateOrDeny unless they explicitly override.
	return "stub-app", platform.TierFull, nil
}
func (stubPlat) Confirm(string) (bool, error) {
	// Default-deny: tests that need a yes must override with a fake
	// returning true. This matches the production OS-dialog default
	// (the dialog defaults to Deny).
	return false, nil
}
