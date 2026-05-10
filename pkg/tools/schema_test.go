package tools

import "testing"

func TestNoArgsSchema(t *testing.T) {
	s := noArgsSchema()
	if s["type"] != "object" {
		t.Errorf("expected type=object, got %v", s["type"])
	}
	props := s["properties"].(map[string]any)
	if _, ok := props["_"]; !ok {
		t.Error("expected _ property")
	}
	required := s["required"].([]string)
	if len(required) != 1 || required[0] != "_" {
		t.Errorf("expected [_] required, got %v", required)
	}
}
