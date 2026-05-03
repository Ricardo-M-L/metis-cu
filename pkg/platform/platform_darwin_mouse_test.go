//go:build darwin

package platform

import (
	"os"
	"testing"
)

// skipIfNoDesktop is the standard guard for tests that drive the real
// pointer / read the real cursor. CI runners and -short test runs do
// not have an active session, and refusing to skip there would make
// every CI build red on a real-world cursor jitter or zero-display
// host. The os.Getenv("CI") check matches GitHub Actions / GitLab CI
// conventions.
func skipIfNoDesktop(t *testing.T) {
	t.Helper()
	if os.Getenv("CI") != "" {
		t.Skip("skip on CI: no interactive session")
	}
	if testing.Short() {
		t.Skip("skip in -short mode: touches real cursor")
	}
}

// TestCursorPosition_NonNegative verifies the macOS cursor query path
// yields a sane Point. Top-left origin means both coordinates are
// always >= 0 on the primary display; multi-monitor secondary displays
// to the left would produce negatives, so we tolerate that case but
// require the call itself to succeed.
func TestCursorPosition_NonNegative(t *testing.T) {
	skipIfNoDesktop(t)
	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	pt, err := p.CursorPosition()
	if err != nil {
		t.Fatalf("CursorPosition: %v", err)
	}
	// Negative coords are legal with secondary displays placed to
	// the left of primary; we just make sure the call doesn't return
	// an obviously-broken sentinel.
	if pt.X == 0 && pt.Y == 0 {
		// {0,0} is a possible-but-suspicious value (cursor parked in
		// the corner). Don't fail — just log so a flaky CI run
		// surfaces the case.
		t.Logf("cursor at origin: %+v (unusual but legal)", pt)
	}
}

// TestMouseMove_Roundtrip moves the cursor to a known coordinate and
// verifies CursorPosition reflects it within a small slop (Retina
// scaling and pixel rounding can produce ±1 px drift). The original
// position is restored at the end so the test is non-disruptive when
// run interactively.
func TestMouseMove_Roundtrip(t *testing.T) {
	skipIfNoDesktop(t)
	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	original, err := p.CursorPosition()
	if err != nil {
		t.Fatalf("CursorPosition (pre): %v", err)
	}
	t.Cleanup(func() {
		// Best-effort restore — ignore errors so a faulty restore
		// doesn't mask the real assertion failure.
		_ = p.MouseMove(original)
	})

	target := Point{X: 100, Y: 100}
	if err := p.MouseMove(target); err != nil {
		t.Fatalf("MouseMove: %v", err)
	}
	got, err := p.CursorPosition()
	if err != nil {
		t.Fatalf("CursorPosition (post): %v", err)
	}
	const slop = 2
	if abs(got.X-target.X) > slop || abs(got.Y-target.Y) > slop {
		t.Errorf("MouseMove(%+v) → cursor at %+v (slop=%d)", target, got, slop)
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
