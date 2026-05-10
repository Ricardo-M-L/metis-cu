//go:build windows

package platform

// Windows keyboard events. robotgo wraps SendInput. parseKeyCombo is
// shared via keycombo.go (no build tag).

import (
	"context"
	"fmt"
	"time"

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

// KeyHold: see darwin twin for the ctx-cancellation rationale (DD-2).
func (p *windowsPlatform) KeyHold(ctx context.Context, combo string, ms int) error {
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
func (p *windowsPlatform) Type(ctx context.Context, text string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if text == "" {
		return nil
	}
	robotgo.TypeStr(text)
	return nil
}
