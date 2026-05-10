package tools

// Adapter for the `type` MCP tool. The file is named type.go (not
// type_tool.go) for consistency with the other adapters; `type` is a
// Go keyword but only ever appears as a string literal here, so the
// name is unambiguous.
//
// All exported / package-level identifiers prefix the tool name with
// `handle` (e.g. handleTypeText) so they don't collide with the
// keyword in any context.
//
// Long-text path (BUG-22): robotgo.TypeStr drops characters under
// load on macOS — a 200-char string can lose 5-10 chars and the model
// believes the type succeeded. For text >= typePasteThreshold runes
// we instead snapshot the clipboard, write the text, send Cmd/Ctrl-V
// (translated per-OS), then restore the prior clipboard via
// ClipboardSnapshot/Restore (DD-1). Faster + reliable.

import (
	"context"
	"fmt"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

// DefaultTypePasteThreshold is the rune-count above which `type`
// switches from synthetic per-key events to clipboard-paste. 80 is
// roughly the point where TypeStr starts dropping characters on a
// busy macOS host; below that the per-key path is fine and avoids
// touching the clipboard at all. Override via config.toml's
// [keyboard] type_paste_threshold key.
const DefaultTypePasteThreshold = 80

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name: "type",
			Description: "Type a UTF-8 string into the focused input. " +
				"Use after `left_click` to focus the target. Frontmost-" +
				"app tier `click` apps (terminals, IDEs) reject this; " +
				"only `full`-tier apps accept synthetic typing. Special " +
				"keys (Enter, Tab, Esc, modifiers) go through `key` / " +
				"`hold_key` instead — `type` is for text content only. " +
				"Long strings (>= 80 runes) are pasted via the clipboard " +
				"to avoid robotgo's char-drop under load; the user's " +
				"prior clipboard is restored automatically.",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"text": map[string]any{
						"type":        "string",
						"description": "UTF-8 text to type. Empty string is a no-op.",
					},
				},
				"required":             []string{"text"},
				"additionalProperties": false,
			},
			Handler: handleTypeText,
		})
	})
}

func handleTypeText(ctx context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	text, err := requireString(params, "text")
	if err != nil {
		return &Result{Text: fmt.Sprintf("type: %v", err), IsError: true}, nil
	}
	if denied, deny := gateOrDeny(plat, "type"); deny {
		return denied, nil
	}
	runes := []rune(text)
	threshold := typePasteThreshold(ctx)
	if len(runes) >= threshold && threshold > 0 {
		if err := typeViaPaste(ctx, plat, text); err != nil {
			return &Result{Text: fmt.Sprintf("type (paste path): %v", err), IsError: true}, nil
		}
		return &Result{Text: fmt.Sprintf("typed: %d chars (via paste, clipboard restored)", len(runes))}, nil
	}
	if err := plat.Type(ctx, text); err != nil {
		return &Result{Text: fmt.Sprintf("type: %v", err), IsError: true}, nil
	}
	return &Result{Text: fmt.Sprintf("typed: %d chars", len(runes))}, nil
}

// typeViaPaste implements the BUG-22 + DD-1 paste path: snapshot →
// write text → Cmd/Ctrl-V → restore. The release of the prior
// clipboard is deferred so a panic mid-paste doesn't strand the
// model's text on the user's clipboard. ctx (DD-2) is checked between
// each step so a cancel during paste cleans up cleanly.
func typeViaPaste(ctx context.Context, plat platform.Platform, text string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	snap := plat.ClipboardSnapshot()
	defer func() { _ = plat.ClipboardRestore(snap) }()
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := plat.ClipboardWrite(text); err != nil {
		return fmt.Errorf("clipboard write: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// "cmd+v" — keycombo.go translates "cmd" → "ctrl" on Linux/Windows
	// so the same combo string works everywhere.
	if err := plat.KeyPress("cmd+v"); err != nil {
		return fmt.Errorf("paste keypress: %w", err)
	}
	return nil
}

// typePasteThreshold pulls the threshold from the active Registry's
// config (DD-3 wires this through config.toml). Falls back to the
// package default when the registry isn't on the context (direct
// handler tests) or when the configured value is non-positive.
func typePasteThreshold(ctx context.Context) int {
	if reg, ok := ctx.Value(registryKey{}).(*Registry); ok && reg != nil {
		if reg.TypePasteThreshold > 0 {
			return reg.TypePasteThreshold
		}
	}
	return DefaultTypePasteThreshold
}
