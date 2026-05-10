package platform

// Button-name mapping shared across darwin / linux / windows. Pulled
// out of the per-OS mouse files (DD-7) because the function body was
// identical in all three; keeping it here means a future Button enum
// addition only needs editing one switch.

// buttonString maps the typed Button enum onto robotgo's stringly-
// typed button names. Unknown values default to "left" — the safest
// fallback, since left-click is a no-op on most surfaces if the
// caller meant something else, whereas right-click triggers context
// menus that are hard to dismiss programmatically.
func buttonString(b Button) string {
	switch b {
	case ButtonRight:
		return "right"
	case ButtonMiddle:
		return "center"
	case ButtonLeft:
		return "left"
	default:
		return "left"
	}
}
