package tools

import (
	"context"
	"fmt"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name:        "write_clipboard",
			Description: "Replace the system clipboard with the provided text.",
			Schema: map[string]any{
				"type":     "object",
				"required": []string{"text"},
				"properties": map[string]any{
					"text": map[string]any{
						"type":        "string",
						"description": "UTF-8 text to place on the clipboard. Replaces any prior content.",
					},
				},
				"additionalProperties": false,
			},
			Handler: handleWriteClipboard,
		})
	})
}

func handleWriteClipboard(_ context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	text, err := requireString(params, "text")
	if err != nil {
		return &Result{Text: fmt.Sprintf("write_clipboard: %v", err), IsError: true}, nil
	}
	// Size cap (read_clipboard already caps reads): a model/macro can't push
	// an arbitrarily large payload through the clipboard backend / memory.
	const maxClipboardWrite = 256 * 1024
	if len(text) > maxClipboardWrite {
		return &Result{Text: fmt.Sprintf("write_clipboard: text too large (%d bytes, max %d)", len(text), maxClipboardWrite), IsError: true}, nil
	}
	if denied, deny := gateOrDeny(plat, "write_clipboard"); deny {
		return denied, nil
	}
	if err := plat.ClipboardWrite(text); err != nil {
		return &Result{Text: fmt.Sprintf("write_clipboard: %v", err), IsError: true}, nil
	}
	// BUG-16: report rune count alongside byte count so multi-byte UTF-8
	// (Chinese, Japanese, emoji) doesn't read as a much larger payload
	// than it actually is. The `type` tool already reports rune count
	// for the same reason — keep them consistent.
	chars := len([]rune(text))
	bytes := len(text)
	if chars == bytes {
		return &Result{Text: fmt.Sprintf("wrote %d bytes to clipboard", bytes)}, nil
	}
	return &Result{Text: fmt.Sprintf("wrote %d chars (%d bytes) to clipboard", chars, bytes)}, nil
}
