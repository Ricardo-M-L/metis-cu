package platform

// scrollSigns translates the wire-format scroll axes (Anthropic CU
// spec: positive dy = scroll DOWN, positive dx = scroll RIGHT) into
// the convention robotgo / the underlying native scroll APIs use
// (positive y = scroll UP, positive x = scroll LEFT — verified
// against robotgo's own ScrollDir implementation:
//
//	if d == "down"  { Scroll(0, -x) }   // down  → negative y
//	if d == "right" { Scroll(-x, 0) }   // right → negative x
//
// Without this flip, requesting "scroll down by 3" actually scrolled
// the page UP. The mismatch was uniform across darwin/linux/windows
// because all three sit on top of robotgo's wrapper. Pulled into a
// shared helper (no build tag) so it can be unit-tested without a
// real input device and so a single source of truth survives any
// future axis-convention sweep.
func scrollSigns(dx, dy int) (int, int) {
	return -dx, -dy
}
