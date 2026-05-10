package server

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadConfig_NoFile: missing config returns defaults verbatim, no
// error. The file is optional — most users never create it.
func TestLoadConfig_NoFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	want := DefaultConfig()
	if got != want {
		t.Errorf("LoadConfig with no file = %+v, want defaults %+v", got, want)
	}
}

// TestLoadConfig_OverridesScreenshot: a valid TOML with [screenshot]
// fields applies cleanly.
func TestLoadConfig_OverridesScreenshot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
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
	t.Setenv("HOME", home)
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
	t.Setenv("HOME", home)
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
	t.Setenv("HOME", home)
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
	t.Setenv("HOME", home)
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
	t.Setenv("HOME", home)
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
