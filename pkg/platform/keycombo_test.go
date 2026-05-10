package platform

// Cross-platform tests for parseKeyComboFor — the host's GOOS doesn't
// matter because we drive the goos param explicitly. This is the only
// path to verify the modifier translation across darwin / linux /
// windows from a single test machine.
//
// The darwin-tagged platform_darwin_keyboard_test.go covers the
// macOS-specific defaults via parseKeyCombo (no goos arg); the cases
// here intentionally exercise *every* OS so a future regression on
// any platform is caught regardless of where CI runs.

import (
	"testing"
)

func TestParseKeyComboFor_TranslatesCmdOffDarwin(t *testing.T) {
	cases := []struct {
		name      string
		goos      string
		input     string
		wantKey   string
		wantMods  []string
		wantError bool
	}{
		// macOS: cmd stays cmd
		{"darwin-cmd-a", "darwin", "cmd+a", "a", []string{"cmd"}, false},
		{"darwin-cmd-shift-z", "darwin", "cmd+shift+z", "z", []string{"cmd", "shift"}, false},

		// Linux: cmd → ctrl
		{"linux-cmd-a", "linux", "cmd+a", "a", []string{"ctrl"}, false},
		{"linux-cmd-shift-z", "linux", "cmd+shift+z", "z", []string{"ctrl", "shift"}, false},

		// Windows: cmd → ctrl
		{"windows-cmd-c", "windows", "cmd+c", "c", []string{"ctrl"}, false},
		{"windows-cmd-alt-tab", "windows", "cmd+alt+tab", "tab", []string{"ctrl", "alt"}, false},

		// Literal "ctrl" passes through everywhere — never gets
		// rewritten to "cmd" on darwin (would break Ctrl-C in IDEs etc.)
		{"darwin-ctrl-c", "darwin", "ctrl+c", "c", []string{"ctrl"}, false},
		{"linux-ctrl-c", "linux", "ctrl+c", "c", []string{"ctrl"}, false},
		{"windows-ctrl-c", "windows", "ctrl+c", "c", []string{"ctrl"}, false},

		// Other modifiers untouched on every platform.
		{"darwin-alt-shift-arrow", "darwin", "alt+shift+left", "left", []string{"alt", "shift"}, false},
		{"linux-alt-shift-arrow", "linux", "alt+shift+left", "left", []string{"alt", "shift"}, false},

		// Single-key combos have no mods to translate.
		{"linux-esc", "linux", "esc", "esc", nil, false},

		// Errors short-circuit before translation.
		{"linux-empty", "linux", "", "", nil, true},
		{"linux-trailing-plus", "linux", "cmd+", "", nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotKey, gotMods, err := parseKeyComboFor(tc.input, tc.goos)
			if tc.wantError {
				if err == nil {
					t.Fatalf("parseKeyComboFor(%q, %q) = (%q, %v, nil); want error",
						tc.input, tc.goos, gotKey, gotMods)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseKeyComboFor(%q, %q) unexpected error: %v",
					tc.input, tc.goos, err)
			}
			if gotKey != tc.wantKey {
				t.Errorf("key: got %q, want %q", gotKey, tc.wantKey)
			}
			if len(gotMods) != len(tc.wantMods) {
				t.Fatalf("mods length mismatch: got %v, want %v", gotMods, tc.wantMods)
			}
			for i, want := range tc.wantMods {
				if got, _ := gotMods[i].(string); got != want {
					t.Errorf("mod[%d]: got %v, want %q", i, gotMods[i], want)
				}
			}
		})
	}
}

// TestTranslatePrimaryModifier exercises the helper directly so any
// future addition (e.g. translating "win" → "cmd" on darwin) is added
// alongside its test row.
func TestTranslatePrimaryModifier(t *testing.T) {
	cases := []struct {
		mod, goos, want string
	}{
		{"cmd", "darwin", "cmd"},
		{"cmd", "linux", "ctrl"},
		{"cmd", "windows", "ctrl"},
		{"cmd", "freebsd", "ctrl"}, // unknown OS → fall through to ctrl
		{"ctrl", "darwin", "ctrl"},
		{"ctrl", "linux", "ctrl"},
		{"ctrl", "windows", "ctrl"},
		{"alt", "linux", "alt"},
		{"shift", "windows", "shift"},
		{"control", "windows", "control"},
	}
	for _, tc := range cases {
		got := translatePrimaryModifier(tc.mod, tc.goos)
		if got != tc.want {
			t.Errorf("translatePrimaryModifier(%q, %q) = %q, want %q",
				tc.mod, tc.goos, got, tc.want)
		}
	}
}
