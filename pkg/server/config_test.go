package server

import (
	"os"
	"path/filepath"
	"testing"
)

// setHome is the cross-platform home redirect for tests. os.UserHomeDir
// reads $HOME on darwin/linux but $USERPROFILE on windows; without the
// double-set the windows CI runner ignores the t.TempDir() and reads
// the real user's ~/.metis-cu/config.toml — failing every override
// assertion with "got default values". Setting both keeps the test
// portable.
func setHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
}

// TestLoadConfig_NoFile: missing config returns defaults verbatim, no
// error. The file is optional — most users never create it.
func TestLoadConfig_NoFile(t *testing.T) {
	setHome(t, t.TempDir())
	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	want := DefaultConfig()
	if got != want {
		t.Errorf("LoadConfig with no file = %+v, want defaults %+v", got, want)
	}
}

// TestLoadConfig_ScreenshotFormat: format=jpeg + quality override
// flow through the loader; an unknown format string falls back to png
// and an out-of-range quality clamps to default 85.
func TestLoadConfig_ScreenshotFormat(t *testing.T) {
	cases := []struct {
		name        string
		body        string
		wantFormat  string
		wantQuality int
	}{
		{
			"jpeg-with-quality",
			"[screenshot]\nformat = \"jpeg\"\nquality = 70\n",
			"jpeg", 70,
		},
		{
			"unknown-format-fallback",
			"[screenshot]\nformat = \"webp\"\n",
			"png", 85,
		},
		{
			"jpeg-bad-quality-clamps",
			"[screenshot]\nformat = \"jpeg\"\nquality = 9000\n",
			"jpeg", 85,
		},
		{
			"png-explicit",
			"[screenshot]\nformat = \"png\"\n",
			"png", 85,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			setHome(t, home)
			writeConfig(t, home, tc.body)
			got, err := LoadConfig()
			if err != nil {
				t.Fatalf("LoadConfig: %v", err)
			}
			if got.Screenshot.Format != tc.wantFormat {
				t.Errorf("Format = %q, want %q", got.Screenshot.Format, tc.wantFormat)
			}
			if got.Screenshot.Quality != tc.wantQuality {
				t.Errorf("Quality = %d, want %d", got.Screenshot.Quality, tc.wantQuality)
			}
		})
	}
}

// TestLoadConfig_OverridesScreenshot: a valid TOML with [screenshot]
// fields applies cleanly.
func TestLoadConfig_OverridesScreenshot(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	writeConfig(t, home, `
[screenshot]
max_width = 1024
max_height = 768
`)
	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if got.Screenshot.MaxWidth != 1024 || got.Screenshot.MaxHeight != 768 {
		t.Errorf("LoadConfig override = %+v, want 1024x768", got.Screenshot)
	}
}

// TestLoadConfig_PartialStanza: only one of the two screenshot keys
// set — the other stays at default rather than being zeroed out.
func TestLoadConfig_PartialStanza(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	writeConfig(t, home, `
[screenshot]
max_width = 1600
`)
	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if got.Screenshot.MaxWidth != 1600 {
		t.Errorf("MaxWidth = %d, want 1600", got.Screenshot.MaxWidth)
	}
	if got.Screenshot.MaxHeight != 800 {
		t.Errorf("MaxHeight = %d, want default 800", got.Screenshot.MaxHeight)
	}
}

// TestLoadConfig_NegativeValues: hostile values fall back to defaults
// (a 0×0 screenshot would be a crash trigger upstream).
func TestLoadConfig_NegativeValues(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	writeConfig(t, home, `
[screenshot]
max_width = -1
max_height = 0
`)
	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if got.Screenshot.MaxWidth != 1280 || got.Screenshot.MaxHeight != 800 {
		t.Errorf("LoadConfig negative-fallback = %+v, want defaults 1280x800", got.Screenshot)
	}
}

// TestLoadConfig_KeyboardOverride: [keyboard] section is honoured.
func TestLoadConfig_KeyboardOverride(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	writeConfig(t, home, `
[keyboard]
type_paste_threshold = 200
`)
	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if got.Keyboard.TypePasteThreshold != 200 {
		t.Errorf("TypePasteThreshold = %d, want 200", got.Keyboard.TypePasteThreshold)
	}
}

// TestLoadConfig_KeyboardZeroFallback: zero/negative falls back to default.
func TestLoadConfig_KeyboardZeroFallback(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	writeConfig(t, home, `
[keyboard]
type_paste_threshold = 0
`)
	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if got.Keyboard.TypePasteThreshold != 80 {
		t.Errorf("TypePasteThreshold = %d, want default 80", got.Keyboard.TypePasteThreshold)
	}
}

// TestLoadConfig_MalformedTOML: parse error returns defaults plus the
// underlying error so the caller can log if it cares.
func TestLoadConfig_MalformedTOML(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	writeConfig(t, home, `not = valid = toml`)
	got, err := LoadConfig()
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
	want := DefaultConfig()
	if got != want {
		t.Errorf("LoadConfig with bad TOML = %+v, want defaults %+v", got, want)
	}
}

func writeConfig(t *testing.T, home, body string) {
	t.Helper()
	dir := filepath.Join(home, ".metis-cu")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
