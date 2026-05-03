package tools

import (
	"fmt"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

// requirePoint extracts a {x:int, y:int} sub-object from params under
// `key`. left_click_drag uses this for both `from` and `to`. Failures
// propagate as plain errors so the calling adapter wraps them in the
// usual "invalid <field>: ..." Result.
func requirePoint(params map[string]any, key string) (platform.Point, error) {
	v, ok := params[key]
	if !ok {
		return platform.Point{}, fmt.Errorf("missing required field: %s", key)
	}
	m, ok := v.(map[string]any)
	if !ok {
		return platform.Point{}, fmt.Errorf("%s: expected object, got %T", key, v)
	}
	x, err := requireInt(m, "x")
	if err != nil {
		return platform.Point{}, fmt.Errorf("%s.x: %w", key, err)
	}
	y, err := requireInt(m, "y")
	if err != nil {
		return platform.Point{}, fmt.Errorf("%s.y: %w", key, err)
	}
	return platform.Point{X: x, Y: y}, nil
}

// requireXY extracts the common (x, y) integer pair from a flat
// params object — the shape used by every mouse tool except
// left_click_drag.
func requireXY(params map[string]any) (platform.Point, error) {
	x, err := requireInt(params, "x")
	if err != nil {
		return platform.Point{}, err
	}
	y, err := requireInt(params, "y")
	if err != nil {
		return platform.Point{}, err
	}
	return platform.Point{X: x, Y: y}, nil
}

// pointSchema is the JSON Schema fragment for a {x:int, y:int} pair.
// Reused by left_click_drag's from / to declarations so both endpoints
// stay in sync.
func pointSchema(desc string) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"x": map[string]any{"type": "integer", "description": "X coordinate (logical pixels, top-left origin)."},
			"y": map[string]any{"type": "integer", "description": "Y coordinate (logical pixels, top-left origin)."},
		},
		"required":             []string{"x", "y"},
		"additionalProperties": false,
		"description":          desc,
	}
}

// xySchema returns the property map shared by every flat-(x,y) tool.
// Tools spread the returned map into their own `properties` block —
// using a helper keeps the schemas consistent (same descriptions,
// same minimum/integer constraints) without per-tool copy-paste.
func xySchema() map[string]any {
	return map[string]any{
		"x": map[string]any{
			"type":        "integer",
			"description": "X coordinate (logical pixels, top-left origin).",
		},
		"y": map[string]any{
			"type":        "integer",
			"description": "Y coordinate (logical pixels, top-left origin).",
		},
	}
}

// rejectModifiers reports whether the params carry a non-empty
// `modifiers` array. Phase 2-A doesn't wire modifier press/release
// (the platform interface doesn't accept modifiers on click and we're
// forbidden from adding robotgo to pkg/tools), so adapters that
// declare a `modifiers` field surface a clear "not supported yet"
// error rather than silently dropping the modifier.
func rejectModifiers(params map[string]any) (bool, string) {
	v, ok := params["modifiers"]
	if !ok {
		return false, ""
	}
	mods, err := asStringSlice(v)
	if err != nil {
		return true, fmt.Sprintf("invalid modifiers: %v", err)
	}
	if len(mods) == 0 {
		return false, ""
	}
	return true, fmt.Sprintf("modifiers %v not supported in this version (Phase 2-A); will be wired in a follow-up", mods)
}
