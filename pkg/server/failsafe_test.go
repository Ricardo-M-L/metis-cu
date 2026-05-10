package server

import (
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

// TestIsInAnyCorner: the geometry-only corner test is the part of
// failsafe.go that's worth unit-testing in CI. The watchdog goroutine
// itself ends in os.Exit(2) which would kill the test process.
func TestIsInAnyCorner(t *testing.T) {
	const (
		minX, minY = 0, 0
		maxX, maxY = 1919, 1079
		corner     = 5
	)
	cases := []struct {
		name string
		pt   platform.Point
		want bool
	}{
		{"top-left exact", platform.Point{X: 0, Y: 0}, true},
		{"top-left within tolerance", platform.Point{X: 3, Y: 4}, true},
		{"top-right exact", platform.Point{X: 1919, Y: 0}, true},
		{"top-right within tolerance", platform.Point{X: 1916, Y: 2}, true},
		{"bottom-left exact", platform.Point{X: 0, Y: 1079}, true},
		{"bottom-right exact", platform.Point{X: 1919, Y: 1079}, true},
		{"bottom-right within tolerance", platform.Point{X: 1915, Y: 1077}, true},
		{"center — should not trip", platform.Point{X: 960, Y: 540}, false},
		{"top edge mid — not corner", platform.Point{X: 960, Y: 0}, false},
		{"left edge mid — not corner", platform.Point{X: 0, Y: 540}, false},
		{"just past tolerance", platform.Point{X: 6, Y: 6}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isInAnyCorner(tc.pt, minX, minY, maxX, maxY, corner)
			if got != tc.want {
				t.Errorf("isInAnyCorner(%+v) = %v, want %v", tc.pt, got, tc.want)
			}
		})
	}
}

// TestStartFailsafeWatchdog_DisabledIsNoOp: enabled=false short-
// circuits before spawning the goroutine and returns a callable
// no-op cancel — verified by calling cancel and checking it doesn't
// panic. Belt-and-braces: confirms users who haven't opted in pay no
// runtime cost and don't need to worry about a hidden goroutine.
func TestStartFailsafeWatchdog_DisabledIsNoOp(t *testing.T) {
	cancel := startFailsafeWatchdog(nil, failsafeConfig{enabled: false})
	cancel() // must be safe to call
}
