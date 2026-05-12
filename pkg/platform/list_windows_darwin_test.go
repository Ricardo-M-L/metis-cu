//go:build darwin

package platform

import "testing"

func TestParseListWindowsTSV(t *testing.T) {
	in := "Safari\tApple\t0\t0\t1440\t900\n" +
		"Code\tmetis-cu — Phase I\t100\t50\t1600\t1050\n" +
		"\n" + // blank line — skipped
		"badline\twithout enough fields\n" + // too few fields — skipped
		"Terminal\tricardo@host\t200\t100\t800\t600"
	got := parseListWindowsTSV(in)
	if len(got) != 3 {
		t.Fatalf("expected 3 windows; got %d (%+v)", len(got), got)
	}
	if got[0].App != "Safari" || got[0].Title != "Apple" {
		t.Errorf("safari row: %+v", got[0])
	}
	if got[1].Bounds.Min.X != 100 || got[1].Bounds.Dx() != 1600 {
		t.Errorf("code row bounds: %+v", got[1])
	}
	if got[2].App != "Terminal" || got[2].Bounds.Dy() != 600 {
		t.Errorf("terminal row: %+v", got[2])
	}
}

func TestParseListWindowsTSV_EmptyInput(t *testing.T) {
	if got := parseListWindowsTSV(""); len(got) != 0 {
		t.Errorf("empty input should yield no windows; got %+v", got)
	}
	if got := parseListWindowsTSV("\n\n\n"); len(got) != 0 {
		t.Errorf("whitespace-only should yield no windows; got %+v", got)
	}
}
