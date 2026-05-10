//go:build darwin

package platform

// Keyboard events. Real implementation (Sprint 2-B): CGEvent posting
// via go-vgo/robotgo. KeyTap synthesizes a press+release for `key`,
// KeyToggle drives down/up explicitly so we can hold a chord for an
// arbitrary duration, and TypeStr (an alias for Type in v1.0.2) sends
// a UTF-8 string through the OS-level input pipeline.
//
// Combo parsing is intentionally trivial — split on "+", lowercase,
// last segment is the key, everything before it is a modifier. Robotgo
// owns the actual key-name → keycode mapping, so we don't enumerate
// special keys here; "esc", "enter", "f11", arrow names, etc. are all
// passed through verbatim.

import (
	"context"
	"fmt"
	"time"

	"github.com/go-vgo/robotgo"
)

// defaultKeyHoldMillis is the duration applied when KeyHold receives
// a non-positive ms argument. 100ms is short enough that an LLM that
// forgets to pass ms still gets "press and hold briefly" semantics
// rather than a no-op or an indefinite hang.
const defaultKeyHoldMillis = 100

// KeyPress synthesizes a single key-tap for combo. Combo is a "+"
// separated string where the final segment is the key and any
// preceding segments are modifiers — e.g. "cmd+shift+a", "esc",
// "ctrl+f5". Modifier names match what robotgo expects ("cmd",
// "shift", "ctrl", "alt"); we lowercase before passing through.
func (p *darwinPlatform) KeyPress(combo string) error {
	key, mods, err := parseKeyCombo(combo)
	if err != nil {
		return err
	}
	if err := robotgo.KeyTap(key, mods...); err != nil {
		return fmt.Errorf("KeyTap(%q): %w", combo, err)
	}
	return nil
}

// KeyHold presses combo, waits for ms milliseconds OR ctx
// cancellation, then releases. Non-positive ms falls back to
// defaultKeyHoldMillis. The release runs in a defer so even a panic
// during the wait still releases modifier keys.
//
// ctx propagation (DD-2) replaces the prior bare MilliSleep — a
// SIGTERM mid-hold or an MCP request-level cancel now triggers an
// immediate release instead of waiting out the full ms window with
// modifiers stuck physically down.
func (p *darwinPlatform) KeyHold(ctx context.Context, combo string, ms int) error {
	key, mods, err := parseKeyCombo(combo)
	if err != nil {
		return err
	}
	if ms <= 0 {
		ms = defaultKeyHoldMillis
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
		// Caller cancelled — release fires via defer; ctx error
		// propagates to the tool layer which surfaces it as IsError.
		return ctx.Err()
	case <-timer.C:
		return releaseErr
	}
}

// Type sends text through the OS input pipeline. UTF-8 is supported
// (robotgo decomposes per-rune internally). Robotgo's TypeStr returns
// no error in v1.0.2, so we always return nil — any platform-level
// failure surfaces as missing characters in the target field, which
// the caller can detect with a follow-up screenshot.
//
// ctx (DD-2): bail before sending input if already cancelled. TypeStr
// is a blocking C call we can't interrupt mid-string, so the
// granularity is whole-message. The paste fallback in pkg/tools/type.go
// checks ctx between snapshot/write/paste/restore steps for
// finer-grained cancellation on long strings.
func (p *darwinPlatform) Type(ctx context.Context, text string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if text == "" {
		return nil
	}
	robotgo.TypeStr(text)
	return nil
}

// parseKeyCombo lives in keycombo.go (no build tag) so all OS impls
// share it.
