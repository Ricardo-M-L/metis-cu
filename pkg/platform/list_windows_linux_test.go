//go:build linux

package platform

import "testing"

func TestParseWmctrlOutput(t *testing.T) {
	// Representative wmctrl -lpG output. Fields:
	// id workspace pid x y w h host class-or-title (rest)
	in := `0x05400003 -1 1234 0 0 1920 1080 hostname x-caja-desktop
0x06800040  0 5678 480 280 960 720 hostname Terminal - ricardo@host: ~
0x06800050  0 9012 100 100 800 600 hostname code-oss - metis-cu — Phase I
malformed line that should not parse
0x06800060  0 3456 0 0 100 100 hostname single-word-class`
	got := parseWmctrlOutput(in)
	if len(got) != 4 {
		t.Fatalf("expected 4 windows; got %d (%+v)", len(got), got)
	}
	// Title parses correctly when " - " separator is present.
	if got[1].App != "Terminal" || got[1].Title != "ricardo@host: ~" {
		t.Errorf("terminal row: app=%q title=%q", got[1].App, got[1].Title)
	}
	if got[2].App != "code-oss" || got[2].Title != "metis-cu — Phase I" {
		t.Errorf("code row: app=%q title=%q", got[2].App, got[2].Title)
	}
	// Bounds parsed from columns 3-6.
	if got[1].Bounds.Min.X != 480 || got[1].Bounds.Dy() != 720 {
		t.Errorf("terminal bounds: %+v", got[1].Bounds)
	}
}
