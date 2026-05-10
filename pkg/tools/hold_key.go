package tools

import (
	"context"
	"fmt"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

// holdKeyMaxMs is the legacy alias for the package default — actual
// runtime cap comes from holdKeyMaxMsFor(ctx) which reads the
// Registry's DD-3 override or falls back to DefaultHoldKeyMaxMs.
// Kept as a const so the schema description compiles without
// duplicating the literal.
const holdKeyMaxMs = DefaultHoldKeyMaxMs

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name: "hold_key",
			Description: "Press a key combo, hold it for `ms` milliseconds, " +
				"then release. Useful for momentary modifiers (e.g. " +
				"`{combo:\"shift\", ms:500}`) or held arrow keys for " +
				"continuous scroll/select. Combo grammar matches `key`. " +
				"Tier `click` apps reject this. ms is capped at 10000 " +
				"(10s) to prevent wedged modifiers.",
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
						"description": "Hold duration in milliseconds. Must be 1..10000.",
						"minimum":     1,
						"maximum":     holdKeyMaxMs,
					},
				},
				"required":             []string{"combo", "ms"},
				"additionalProperties": false,
			},
			Handler: handleHoldKey,
		})
	})
}

func handleHoldKey(ctx context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
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
	cap := holdKeyMaxMsFor(ctx)
	if ms > cap {
		return &Result{Text: fmt.Sprintf("hold_key: ms must be <= %d, got %d (BUG-11 cap: prevents a hallucinated giant ms wedging modifiers)", cap, ms), IsError: true}, nil
	}
	if denied, deny := gateOrDeny(plat, "hold_key"); deny {
		return denied, nil
	}
	if err := plat.KeyHold(ctx, combo, ms); err != nil {
		return &Result{Text: fmt.Sprintf("hold_key(%q, %dms): %v", combo, ms, err), IsError: true}, nil
	}
	return &Result{Text: fmt.Sprintf("held: %s for %dms", combo, ms)}, nil
}
