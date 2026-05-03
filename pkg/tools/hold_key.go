package tools

import (
	"context"
	"fmt"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name: "hold_key",
			Description: "Press a key combo, hold it for `ms` milliseconds, " +
				"then release. Useful for momentary modifiers (e.g. " +
				"`{combo:\"shift\", ms:500}`) or held arrow keys for " +
				"continuous scroll/select. Combo grammar matches `key`. " +
				"Tier `click` apps reject this.",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"combo": map[string]any{
						"type":        "string",
						"description": "Key combo, '+'-joined. Same grammar as the `key` tool.",
						"minLength":   1,
					},
					"ms": map[string]any{
						"type":        "integer",
						"description": "Hold duration in milliseconds. Must be >= 1.",
						"minimum":     1,
					},
				},
				"required":             []any{"combo", "ms"},
				"additionalProperties": false,
			},
			Handler: handleHoldKey,
		})
	})
}

func handleHoldKey(_ context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	combo, err := requireString(params, "combo")
	if err != nil {
		return &Result{Text: fmt.Sprintf("hold_key: %v", err), IsError: true}, nil
	}
	ms, err := requireInt(params, "ms")
	if err != nil {
		return &Result{Text: fmt.Sprintf("hold_key: %v", err), IsError: true}, nil
	}
	if ms < 1 {
		return &Result{Text: fmt.Sprintf("hold_key: ms must be >= 1, got %d", ms), IsError: true}, nil
	}
	if err := plat.KeyHold(combo, ms); err != nil {
		return &Result{Text: fmt.Sprintf("hold_key(%q, %dms): %v", combo, ms, err), IsError: true}, nil
	}
	return &Result{Text: fmt.Sprintf("held: %s for %dms", combo, ms)}, nil
}
