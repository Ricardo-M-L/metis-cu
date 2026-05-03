//go:build darwin

package platform

import (
	"os"
	"testing"
)

// TestParseKeyCombo exercises the combo parser. Pure unit test — no
// CGEvent posting involved — so it runs everywhere including CI.
func TestParseKeyCombo(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		wantKey  string
		wantMods []string
		wantErr  bool
	}{
		{name: "single key", input: "a", wantKey: "a"},
		{name: "uppercase normalized", input: "A", wantKey: "a"},
		{name: "one modifier", input: "cmd+a", wantKey: "a", wantMods: []string{"cmd"}},
		{name: "two modifiers", input: "cmd+shift+esc", wantKey: "esc", wantMods: []string{"cmd", "shift"}},
		{name: "spaces tolerated", input: " cmd + a ", wantKey: "a", wantMods: []string{"cmd"}},
		{name: "function key", input: "f11", wantKey: "f11"},
		{name: "empty", input: "", wantErr: true},
		{name: "whitespace only", input: "   ", wantErr: true},
		{name: "trailing plus", input: "cmd+", wantErr: true},
		{name: "leading plus", input: "+a", wantErr: true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gotKey, gotMods, err := parseKeyCombo(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseKeyCombo(%q) = (%q, %v, nil); want error", tc.input, gotKey, gotMods)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseKeyCombo(%q) unexpected error: %v", tc.input, err)
			}
			if gotKey != tc.wantKey {
				t.Errorf("key: got %q, want %q", gotKey, tc.wantKey)
			}
			if len(gotMods) != len(tc.wantMods) {
				t.Fatalf("mods length: got %v, want %v", gotMods, tc.wantMods)
			}
			for i, want := range tc.wantMods {
				got, ok := gotMods[i].(string)
				if !ok {
					t.Fatalf("mod[%d] not a string: %T", i, gotMods[i])
				}
				if got != want {
					t.Errorf("mod[%d]: got %q, want %q", i, got, want)
				}
			}
		})
	}
}

// TestType_EmptyString verifies Type accepts an empty string without
// error and without invoking the underlying robotgo call (which would
// be a no-op anyway). This runs everywhere — empty input never reaches
// the OS-level input pipeline.
func TestType_EmptyString(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := p.Type(""); err != nil {
		t.Fatalf("Type(\"\") returned error: %v", err)
	}
}

// TestKeyPress_Modifier exercises the real robotgo path with a
// harmless modifier-only "shift" press. Skipped on CI (no GUI session)
// and under -short — local manual verification covers the full key /
// chord matrix for press correctness. We only assert "no error, no
// panic" here: confirming the OS actually saw the keystroke would
// require a focused input target, which CI lacks.
func TestKeyPress_Modifier(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("skipping real KeyPress on CI (no GUI session, accessibility may be unset)")
	}
	if testing.Short() {
		t.Skip("skipping real KeyPress under -short")
	}
	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// "shift" alone is innocuous: pressing and releasing it doesn't
	// trigger any system action and shouldn't disturb any focused app.
	if err := p.KeyPress("shift"); err != nil {
		t.Fatalf("KeyPress(\"shift\"): %v", err)
	}
}

// TestKeyHold_DefaultDuration verifies the ms<=0 fallback path doesn't
// produce an error. Same skip rules as TestKeyPress_Modifier — this
// touches the real CGEvent pipeline.
func TestKeyHold_DefaultDuration(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("skipping real KeyHold on CI (no GUI session)")
	}
	if testing.Short() {
		t.Skip("skipping real KeyHold under -short")
	}
	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := p.KeyHold("shift", 0); err != nil {
		t.Fatalf("KeyHold(\"shift\", 0): %v", err)
	}
}
