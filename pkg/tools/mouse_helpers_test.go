package tools

import (
	"testing"
)

func TestRequirePoint_OK(t *testing.T) {
	params := map[string]any{
		"from": map[string]any{"x": 100, "y": 200},
	}
	pt, err := requirePoint(params, "from")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pt.X != 100 || pt.Y != 200 {
		t.Errorf("got (%d, %d), want (100, 200)", pt.X, pt.Y)
	}
}

func TestRequirePoint_Missing(t *testing.T) {
	params := map[string]any{}
	_, err := requirePoint(params, "from")
	if err == nil {
		t.Fatal("expected error for missing point")
	}
}

func TestRequirePoint_NotAnObject(t *testing.T) {
	params := map[string]any{"from": "not-an-object"}
	_, err := requirePoint(params, "from")
	if err == nil {
		t.Fatal("expected error for non-object value")
	}
}

func TestRequirePoint_MissingX(t *testing.T) {
	params := map[string]any{
		"from": map[string]any{"y": 200},
	}
	_, err := requirePoint(params, "from")
	if err == nil {
		t.Fatal("expected error for missing x")
	}
}

func TestRequireXY_OK(t *testing.T) {
	params := map[string]any{"x": 50, "y": 75}
	pt, err := requireXY(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pt.X != 50 || pt.Y != 75 {
		t.Errorf("got (%d, %d), want (50, 75)", pt.X, pt.Y)
	}
}

func TestRequireXY_MissingX(t *testing.T) {
	params := map[string]any{"y": 75}
	_, err := requireXY(params)
	if err == nil {
		t.Fatal("expected error for missing x")
	}
}

func TestRequireXY_MissingY(t *testing.T) {
	params := map[string]any{"x": 50}
	_, err := requireXY(params)
	if err == nil {
		t.Fatal("expected error for missing y")
	}
}

// optionalModifiers replaces rejectModifiers (BUG-21). Tests cover:
// no key → nil, empty → nil, valid names → passthrough, unknown name
// → error, non-array type → error.
func TestOptionalModifiers_NoKey(t *testing.T) {
	mods, err := optionalModifiers(map[string]any{"x": 100})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if mods != nil {
		t.Errorf("expected nil for missing key, got %v", mods)
	}
}

func TestOptionalModifiers_EmptyArray(t *testing.T) {
	mods, err := optionalModifiers(map[string]any{"modifiers": []any{}})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if mods != nil {
		t.Errorf("expected nil for empty array, got %v", mods)
	}
}

func TestOptionalModifiers_ValidNames(t *testing.T) {
	mods, err := optionalModifiers(map[string]any{
		"modifiers": []any{"cmd", "shift", "alt", "ctrl"},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := []string{"cmd", "shift", "alt", "ctrl"}
	if len(mods) != len(want) {
		t.Fatalf("got %v, want %v", mods, want)
	}
	for i := range want {
		if mods[i] != want[i] {
			t.Errorf("mod[%d]: got %q, want %q", i, mods[i], want[i])
		}
	}
}

func TestOptionalModifiers_UnknownName(t *testing.T) {
	_, err := optionalModifiers(map[string]any{"modifiers": []any{"super"}})
	if err == nil {
		t.Fatal("expected error for unknown modifier name")
	}
}

func TestOptionalModifiers_InvalidType(t *testing.T) {
	_, err := optionalModifiers(map[string]any{"modifiers": "not-an-array"})
	if err == nil {
		t.Fatal("expected error for non-array modifiers")
	}
}

func TestPointSchema(t *testing.T) {
	s := pointSchema("test description")
	if s["type"] != "object" {
		t.Errorf("expected type=object, got %v", s["type"])
	}
	props := s["properties"].(map[string]any)
	if _, ok := props["x"]; !ok {
		t.Error("expected x property")
	}
	if _, ok := props["y"]; !ok {
		t.Error("expected y property")
	}
	required := s["required"].([]string)
	if len(required) != 2 || required[0] != "x" || required[1] != "y" {
		t.Errorf("expected [x, y] required, got %v", required)
	}
}

func TestXYSchema(t *testing.T) {
	s := xySchema()
	if s["x"] == nil || s["y"] == nil {
		t.Error("xySchema should return x and y properties")
	}
}

func TestRequireRect_OK(t *testing.T) {
	params := map[string]any{
		"region": map[string]any{"x": 10, "y": 20, "w": 100, "h": 200},
	}
	r, err := requireRect(params, "region")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.X != 10 || r.Y != 20 || r.W != 100 || r.H != 200 {
		t.Errorf("got (%d,%d,%d,%d), want (10,20,100,200)", r.X, r.Y, r.W, r.H)
	}
}

func TestRequireRect_Missing(t *testing.T) {
	_, err := requireRect(map[string]any{}, "region")
	if err == nil {
		t.Fatal("expected error for missing region")
	}
}

func TestRequireRect_NotAnObject(t *testing.T) {
	_, err := requireRect(map[string]any{"region": "not-an-object"}, "region")
	if err == nil {
		t.Fatal("expected error for non-object")
	}
}

func TestRequireRect_ZeroWidth(t *testing.T) {
	params := map[string]any{
		"region": map[string]any{"x": 10, "y": 20, "w": 0, "h": 200},
	}
	_, err := requireRect(params, "region")
	if err == nil {
		t.Fatal("expected error for zero width")
	}
}

func TestRequireRect_NegativeHeight(t *testing.T) {
	params := map[string]any{
		"region": map[string]any{"x": 10, "y": 20, "w": 100, "h": -1},
	}
	_, err := requireRect(params, "region")
	if err == nil {
		t.Fatal("expected error for negative height")
	}
}
