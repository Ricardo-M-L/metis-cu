package tools

// Adapter for the `type` MCP tool. The file is named type.go (not
// type_tool.go) for consistency with the other adapters; `type` is a
// Go keyword but only ever appears as a string literal here, so the
// name is unambiguous.
//
// All exported / package-level identifiers prefix the tool name with
// `handle` (e.g. handleTypeText) so they don't collide with the
// keyword in any context.

import (
	"context"
	"fmt"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name: "type",
			Description: "Type a UTF-8 string into the focused input. " +
				"Use after `left_click` to focus the target. Frontmost-" +
				"app tier `click` apps (terminals, IDEs) reject this; " +
				"only `full`-tier apps accept synthetic typing. Special " +
				"keys (Enter, Tab, Esc, modifiers) go through `key` / " +
				"`hold_key` instead — `type` is for text content only.",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"text": map[string]any{
						"type":        "string",
						"description": "UTF-8 text to type. Empty string is a no-op.",
					},
				},
				"required":             []any{"text"},
				"additionalProperties": false,
			},
			Handler: handleTypeText,
		})
	})
}

func handleTypeText(_ context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	text, err := requireString(params, "text")
	if err != nil {
		return &Result{Text: fmt.Sprintf("type: %v", err), IsError: true}, nil
	}
	if err := plat.Type(text); err != nil {
		return &Result{Text: fmt.Sprintf("type: %v", err), IsError: true}, nil
	}
	return &Result{Text: fmt.Sprintf("typed: %d chars", len([]rune(text)))}, nil
}
