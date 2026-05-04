package tools

// JSON Schema helpers shared across tool adapters. Kept separate from
// params.go (which deals with runtime value coercion) so it's clear
// which file owns which side of the boundary: schema.go advertises
// the contract, params.go enforces it on incoming calls.

// noArgsSchema returns a JSON Schema for tools that take no real
// arguments. We expose a single REQUIRED `_` string property rather
// than a truly empty `properties: {}` block as a workaround for
// tool-calling models (notably MiniMax-M2.7 served through MiniMax's
// own Anthropic-format gateway at https://api.minimaxi.com/anthropic)
// whose gateway serializer emits invalid JSON when the model's tool
// call has empty `arguments`. Empirically, tools with required args
// round-trip fine; tools with no required args produce HTTP 400
// `invalid function arguments json string` even though the model
// "wants" to call them.
//
// Marking `_` as required forces the model to emit a non-empty
// arguments object (e.g. `{"_":""}`), which the gateway's serializer
// handles correctly. Handlers ignore the field — its only job is to
// nudge the gateway off the broken empty-args code path.
//
// Cost: one extra required string in tools/list output. Benefit:
// works on every Anthropic-compatible gateway regardless of how the
// upstream model serializes empty function calls.
func noArgsSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"_": map[string]any{
				"type":        "string",
				"description": "Pass any non-empty string (e.g. \"noop\"). Required workaround for gateways (notably MiniMax) that emit invalid JSON for truly empty function arguments. The value itself is ignored by the handler — the field's only job is to force the model out of the empty-args serialization path.",
			},
		},
		"required":             []string{"_"},
		"additionalProperties": false,
	}
}
