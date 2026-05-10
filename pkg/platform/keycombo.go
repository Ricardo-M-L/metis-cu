package platform

// parseKeyCombo is shared across all OS keyboard implementations
// (darwin / linux / windows). The combo grammar is the same on every
// platform — '+'-joined, last segment is the key, preceding segments
// are modifiers — and the lowercase normalization keeps the lookup
// against robotgo's name table consistent.
//
// Returns []any (rather than []string) because robotgo's KeyTap /
// KeyToggle take variadic any so the slice can be spread directly.
//
// Modifier translation: an LLM trained on macOS conventions emits
// "cmd+a" to mean "select all" — but robotgo's "cmd" name maps to
// K_META, which on Windows is the Win key and on Linux is the Super
// key. Neither does what the model intends. translatePrimaryModifier
// rewrites "cmd" → "ctrl" off-darwin so the muscle-memory combo works
// everywhere. The literal "ctrl" / "control" / "alt" / "shift" pass
// through unchanged on all platforms.

import (
	"fmt"
	"runtime"
	"strings"
)

// parseKeyCombo is the public entrypoint — uses runtime.GOOS for the
// modifier translation. parseKeyComboFor lets tests drive the
// translation across all three OSes from a single host.
func parseKeyCombo(combo string) (key string, mods []any, err error) {
	return parseKeyComboFor(combo, runtime.GOOS)
}

func parseKeyComboFor(combo, goos string) (key string, mods []any, err error) {
	combo = strings.TrimSpace(combo)
	if combo == "" {
		return "", nil, fmt.Errorf("empty key combo")
	}
	parts := strings.Split(combo, "+")
	for i, part := range parts {
		parts[i] = strings.ToLower(strings.TrimSpace(part))
	}
	key = parts[len(parts)-1]
	if key == "" {
		return "", nil, fmt.Errorf("empty key in combo %q", combo)
	}
	if len(parts) == 1 {
		return key, nil, nil
	}
	mods = make([]any, 0, len(parts)-1)
	for _, m := range parts[:len(parts)-1] {
		if m == "" {
			return "", nil, fmt.Errorf("empty modifier in combo %q", combo)
		}
		mods = append(mods, translatePrimaryModifier(m, goos))
	}
	return key, mods, nil
}

// translatePrimaryModifier maps the wire-format "cmd" modifier onto
// the platform's actual primary modifier:
//
//	darwin  → "cmd"  (K_META = Command key, what the user wants)
//	linux   → "ctrl" (K_CONTROL — "cmd" robotgo-mapped to Super, wrong)
//	windows → "ctrl" (K_CONTROL — "cmd" robotgo-mapped to Win, wrong)
//
// Other modifiers ("ctrl", "alt", "shift", "control") pass through —
// they already mean the right thing on every platform.
func translatePrimaryModifier(mod, goos string) string {
	if mod != "cmd" {
		return mod
	}
	if goos == "darwin" {
		return "cmd"
	}
	return "ctrl"
}
