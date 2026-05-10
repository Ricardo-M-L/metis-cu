package platform

import "testing"

// TestScrollSigns walks the four cardinal directions plus zero. Each
// row encodes "the LLM asked for X" and asserts the call we make to
// robotgo. If a future robotgo upgrade flips its convention, this
// test is the canary — without it, real-device testing is the only
// way to catch the regression.
func TestScrollSigns(t *testing.T) {
	cases := []struct {
		name             string
		inDx, inDy       int
		wantDx, wantDy   int
		humanDescription string
	}{
		{"down", 0, 3, 0, -3, "wire dy=+3 (scroll down) → robotgo y=-3"},
		{"up", 0, -3, 0, 3, "wire dy=-3 (scroll up) → robotgo y=+3"},
		{"right", 5, 0, -5, 0, "wire dx=+5 (scroll right) → robotgo x=-5"},
		{"left", -5, 0, 5, 0, "wire dx=-5 (scroll left) → robotgo x=+5"},
		{"diagonal-down-right", 2, 4, -2, -4, "both axes flipped"},
		{"zero-zero", 0, 0, 0, 0, "no-op stays no-op"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotDx, gotDy := scrollSigns(tc.inDx, tc.inDy)
			if gotDx != tc.wantDx || gotDy != tc.wantDy {
				t.Errorf("scrollSigns(%d, %d) = (%d, %d), want (%d, %d) — %s",
					tc.inDx, tc.inDy, gotDx, gotDy, tc.wantDx, tc.wantDy, tc.humanDescription)
			}
		})
	}
}
