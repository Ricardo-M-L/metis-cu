//go:build linux

package platform

// Linux keyboard events. Same robotgo path as darwin — XTEST handles
// synthetic key events, and TypeStr decomposes UTF-8 into per-rune
// X-side keysym lookups. parseKeyCombo is shared with darwin via the
// platform_keycombo.go file (no build tag).

import (
	"context"
	"fmt"
	"time"

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

// KeyHold: see darwin twin for the ctx-cancellation + deferred-release
// rationale (DD-2).
func (p *linuxPlatform) KeyHold(ctx context.Context, combo string, ms int) error {
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
	var releaseErr error
	defer func() {
		upArgs := append([]any{"up"}, mods...)
		if err := robotgo.KeyToggle(key, upArgs...); err != nil && releaseErr == nil {
			releaseErr = fmt.Errorf("KeyToggle up %q: %w", combo, err)
		}
	}()
	timer := time.NewTimer(time.Duration(ms) * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return releaseErr
	}
}

// Type: see darwin twin for the ctx-at-entry rationale (DD-2).
func (p *linuxPlatform) Type(ctx context.Context, text string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if text == "" {
		return nil
	}
	robotgo.TypeStr(text)
	return nil
}
