//go:build linux

package platform

// Linux keyboard events. Same robotgo path as darwin — XTEST handles
// synthetic key events, and TypeStr decomposes UTF-8 into per-rune
// X-side keysym lookups. parseKeyCombo is shared with darwin via the
// platform_keycombo.go file (no build tag).

import (
	"fmt"

	"github.com/go-vgo/robotgo"
)

const defaultKeyHoldMillisLinux = 100

func (p *linuxPlatform) KeyPress(combo string) error {
	key, mods, err := parseKeyCombo(combo)
	if err != nil {
		return err
	}
	if err := robotgo.KeyTap(key, mods...); err != nil {
		return fmt.Errorf("KeyTap(%q): %w", combo, err)
	}
	return nil
}

func (p *linuxPlatform) KeyHold(combo string, ms int) error {
	key, mods, err := parseKeyCombo(combo)
	if err != nil {
		return err
	}
	if ms <= 0 {
		ms = defaultKeyHoldMillisLinux
	}
	downArgs := append([]any{"down"}, mods...)
	if err := robotgo.KeyToggle(key, downArgs...); err != nil {
		return fmt.Errorf("KeyToggle down %q: %w", combo, err)
	}
	robotgo.MilliSleep(ms)
	upArgs := append([]any{"up"}, mods...)
	if err := robotgo.KeyToggle(key, upArgs...); err != nil {
		return fmt.Errorf("KeyToggle up %q: %w", combo, err)
	}
	return nil
}

func (p *linuxPlatform) Type(text string) error {
	if text == "" {
		return nil
	}
	robotgo.TypeStr(text)
	return nil
}
