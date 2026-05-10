package tools

import (
	"image"
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

func TestAsInt(t *testing.T) {
	cases := []struct {
		input   any
		want    int
		wantErr bool
	}{
		{int(42), 42, false},
		{int32(42), 42, false},
		{int64(42), 42, false},
		{float64(42), 42, false},
		{float32(42), 42, false},
		{"42", 42, false},
		{float64(42.5), 0, true},
		{"not-a-number", 0, true},
		{true, 0, true},
	}
	for _, tc := range cases {
		got, err := asInt(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("asInt(%T(%v)) expected error, got %d", tc.input, tc.input, got)
			}
		} else {
			if err != nil {
				t.Errorf("asInt(%T(%v)) unexpected error: %v", tc.input, tc.input, err)
			}
			if got != tc.want {
				t.Errorf("asInt(%T(%v)) = %d, want %d", tc.input, tc.input, got, tc.want)
			}
		}
	}
}

func TestAsString(t *testing.T) {
	got, err := asString("hello")
	if err != nil || got != "hello" {
		t.Errorf("asString(\"hello\") = %q, %v; want \"hello\", nil", got, err)
	}
	_, err = asString(123)
	if err == nil {
		t.Error("asString(123) expected error")
	}
}

func TestAsStringSlice(t *testing.T) {
	got, err := asStringSlice([]string{"a", "b"})
	if err != nil || len(got) != 2 || got[0] != "a" {
		t.Errorf("asStringSlice([]string{\"a\",\"b\"}) = %v, %v; want [a,b], nil", got, err)
	}
	got, err = asStringSlice([]any{"x", "y"})
	if err != nil || len(got) != 2 || got[1] != "y" {
		t.Errorf("asStringSlice([]any) = %v, %v; want [x,y], nil", got, err)
	}
	_, err = asStringSlice([]any{"ok", 123})
	if err == nil {
		t.Error("asStringSlice mixed types expected error")
	}
	_, err = asStringSlice("not-a-slice")
	if err == nil {
		t.Error("asStringSlice string expected error")
	}
}

func TestRequireInt(t *testing.T) {
	params := map[string]any{"count": 5}
	got, err := requireInt(params, "count")
	if err != nil || got != 5 {
		t.Errorf("requireInt(params, \"count\") = %d, %v; want 5, nil", got, err)
	}
	_, err = requireInt(params, "missing")
	if err == nil {
		t.Error("requireInt missing key expected error")
	}
}

func TestRequireString(t *testing.T) {
	params := map[string]any{"name": "test"}
	got, err := requireString(params, "name")
	if err != nil || got != "test" {
		t.Errorf("requireString(params, \"name\") = %q, %v; want \"test\", nil", got, err)
	}
	_, err = requireString(params, "missing")
	if err == nil {
		t.Error("requireString missing key expected error")
	}
}

func TestOptionalInt(t *testing.T) {
	params := map[string]any{"count": 7}
	got, err := optionalInt(params, "count", 99)
	if err != nil || got != 7 {
		t.Errorf("optionalInt(count=7) = %d, %v; want 7, nil", got, err)
	}
	got, err = optionalInt(params, "missing", 99)
	if err != nil || got != 99 {
		t.Errorf("optionalInt(missing) = %d, %v; want 99, nil", got, err)
	}
}

func TestOptionalString(t *testing.T) {
	params := map[string]any{"name": "hello"}
	got, err := optionalString(params, "name", "default")
	if err != nil || got != "hello" {
		t.Errorf("optionalString(name=hello) = %q, %v; want \"hello\", nil", got, err)
	}
	got, err = optionalString(params, "missing", "default")
	if err != nil || got != "default" {
		t.Errorf("optionalString(missing) = %q, %v; want \"default\", nil", got, err)
	}
}

func TestAsFloat(t *testing.T) {
	cases := []struct {
		input   any
		want    float64
		wantErr bool
	}{
		{float64(3.14), 3.14, false},
		{int(5), 5.0, false},
		{int64(5), 5.0, false},
		{"3.14", 3.14, false},
		{"invalid", 0, true},
		{true, 0, true},
	}
	for _, tc := range cases {
		got, err := asFloat(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("asFloat(%T(%v)) expected error, got %f", tc.input, tc.input, got)
			}
		} else {
			if err != nil {
				t.Errorf("asFloat(%T(%v)) unexpected error: %v", tc.input, tc.input, err)
			}
			if got != tc.want {
				t.Errorf("asFloat(%T(%v)) = %f, want %f", tc.input, tc.input, got, tc.want)
			}
		}
	}
}

func TestAsFloat_Float32(t *testing.T) {
	// Float32 has precision issues when compared directly to float64
	got, err := asFloat(float32(3.14))
	if err != nil {
		t.Errorf("asFloat(float32) unexpected error: %v", err)
	}
	// Use a tolerance check since float32(3.14) != float64(3.14) exactly
	if got < 3.13 || got > 3.15 {
		t.Errorf("asFloat(float32(3.14)) = %f, want ~3.14", got)
	}
}

func TestRequireFloat(t *testing.T) {
	params := map[string]any{"factor": 2.5}
	got, err := requireFloat(params, "factor")
	if err != nil || got != 2.5 {
		t.Errorf("requireFloat(factor=2.5) = %f, %v; want 2.5, nil", got, err)
	}
	_, err = requireFloat(params, "missing")
	if err == nil {
		t.Error("requireFloat missing key expected error")
	}
}

// stubPlatForParams provides a stub platform.Platform for tests that need
// to reference the type but don't actually call methods.
type stubPlatForParams struct{}

func (stubPlatForParams) Close() error                     { return platform.ErrNotImplemented }
func (stubPlatForParams) Screenshot() (image.Image, error) { return nil, platform.ErrNotImplemented }
func (stubPlatForParams) CursorPosition() (platform.Point, error) {
	return platform.Point{}, platform.ErrNotImplemented
}
func (stubPlatForParams) DisplayCount() (int, error) { return 0, platform.ErrNotImplemented }
func (stubPlatForParams) SwitchDisplay(int) error    { return platform.ErrNotImplemented }
func (stubPlatForParams) DisplayBounds(int) (image.Rectangle, error) {
	return image.Rectangle{}, platform.ErrNotImplemented
}
func (stubPlatForParams) MouseMove(platform.Point) error { return platform.ErrNotImplemented }
func (stubPlatForParams) MouseClick(platform.Point, platform.Button, int) error {
	return platform.ErrNotImplemented
}
func (stubPlatForParams) MouseDown(platform.Point, platform.Button) error {
	return platform.ErrNotImplemented
}
func (stubPlatForParams) MouseUp(platform.Point, platform.Button) error {
	return platform.ErrNotImplemented
}
func (stubPlatForParams) MouseDrag(platform.Point, platform.Point, platform.Button) error {
	return platform.ErrNotImplemented
}
func (stubPlatForParams) Scroll(platform.Point, int, int) error { return platform.ErrNotImplemented }
func (stubPlatForParams) ScrollWithModifiers(platform.Point, int, int, []string) error {
	return platform.ErrNotImplemented
}
func (stubPlatForParams) KeyPress(string) error          { return platform.ErrNotImplemented }
func (stubPlatForParams) KeyHold(string, int) error      { return platform.ErrNotImplemented }
func (stubPlatForParams) Type(string) error              { return platform.ErrNotImplemented }
func (stubPlatForParams) ClipboardRead() (string, error) { return "", platform.ErrNotImplemented }
func (stubPlatForParams) ClipboardWrite(string) error    { return platform.ErrNotImplemented }
func (stubPlatForParams) OpenApplication(string) error   { return platform.ErrNotImplemented }
func (stubPlatForParams) GrantedApplications() ([]string, error) {
	return nil, platform.ErrNotImplemented
}
func (stubPlatForParams) RequestAccess([]string) (map[string]platform.AccessTier, error) {
	return nil, platform.ErrNotImplemented
}
func (stubPlatForParams) FrontmostApp() (string, platform.AccessTier, error) {
	return "", platform.TierFull, platform.ErrNotImplemented
}
