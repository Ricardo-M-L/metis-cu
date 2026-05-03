//go:build windows

package platform

// Windows keyboard events. robotgo wraps SendInput. parseKeyCombo is
// shared via keycombo.go (no build tag).

import (
	"fmt"

	"github.com/go-vgo/robotgo"
)

const defaultKeyHoldMillisWindows = 100

func (p *windowsPlatform) KeyPress(combo string) error {
	key, mods, err := parseKeyCombo(combo)
	if err != nil {
		return err
	}
	if err := robotgo.KeyTap(key, mods...); err != nil {
		return fmt.Errorf("KeyTap(%q): %w", combo, err)
	}
	return nil
}

func (p *windowsPlatform) KeyHold(combo string, ms int) error {
	key, mods, err := parseKeyCombo(combo)
	if err != nil {
		return err
	}
	if ms <= 0 {
		ms = defaultKeyHoldMillisWindows
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

func (p *windowsPlatform) Type(text string) error {
	if text == "" {
		return nil
	}
	robotgo.TypeStr(text)
	return nil
}
