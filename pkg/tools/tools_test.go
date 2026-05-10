package tools

import (
	"context"
	"image"
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

func TestResult_IsError(t *testing.T) {
	res := &Result{Text: "error message", IsError: true}
	if !res.IsError {
		t.Error("expected IsError=true")
	}
}

func TestResult_ImageAndMIMEType(t *testing.T) {
	res := &Result{
		Text:     "captured PNG",
		Image:    "base64data",
		MIMEType: "image/png",
	}
	if res.Image != "base64data" || res.MIMEType != "image/png" {
		t.Errorf("got Image=%q MIMEType=%q, want base64data/image/png", res.Image, res.MIMEType)
	}
}

// stubPlatForTools mirrors stubPlat from gate_test.go but is local to this file
// so it doesn't create cross-file dependency on the test scope.
type stubPlatForTools struct{}

func (stubPlatForTools) Close() error                     { return platform.ErrNotImplemented }
func (stubPlatForTools) Screenshot() (image.Image, error) { return nil, platform.ErrNotImplemented }
func (stubPlatForTools) CursorPosition() (platform.Point, error) {
	return platform.Point{}, platform.ErrNotImplemented
}
func (stubPlatForTools) DisplayCount() (int, error) { return 0, platform.ErrNotImplemented }
func (stubPlatForTools) SwitchDisplay(int) error    { return platform.ErrNotImplemented }
func (stubPlatForTools) DisplayBounds(int) (image.Rectangle, error) {
	return image.Rectangle{}, platform.ErrNotImplemented
}
func (stubPlatForTools) MouseMove(platform.Point) error { return platform.ErrNotImplemented }
func (stubPlatForTools) MouseClick(platform.Point, platform.Button, int) error {
	return platform.ErrNotImplemented
}
func (stubPlatForTools) MouseClickWithModifiers(platform.Point, platform.Button, int, []string) error {
	return platform.ErrNotImplemented
}
func (stubPlatForTools) MouseDown(platform.Point, platform.Button) error {
	return platform.ErrNotImplemented
}
func (stubPlatForTools) MouseUp(platform.Point, platform.Button) error {
	return platform.ErrNotImplemented
}
func (stubPlatForTools) MouseDrag(context.Context, platform.Point, platform.Point, platform.Button) error {
	return platform.ErrNotImplemented
}
func (stubPlatForTools) Scroll(platform.Point, int, int) error { return platform.ErrNotImplemented }
func (stubPlatForTools) ScrollWithModifiers(platform.Point, int, int, []string) error {
	return platform.ErrNotImplemented
}
func (stubPlatForTools) KeyPress(string) error { return platform.ErrNotImplemented }
func (stubPlatForTools) KeyHold(context.Context, string, int) error {
	return platform.ErrNotImplemented
}
func (stubPlatForTools) Type(context.Context, string) error { return platform.ErrNotImplemented }
func (stubPlatForTools) ClipboardRead() (string, error)     { return "", platform.ErrNotImplemented }
func (stubPlatForTools) ClipboardWrite(string) error        { return platform.ErrNotImplemented }
func (stubPlatForTools) ClipboardSnapshot() platform.ClipboardSnapshot {
	return platform.ClipboardSnapshot{Empty: true}
}
func (stubPlatForTools) ClipboardRestore(platform.ClipboardSnapshot) error { return nil }
func (stubPlatForTools) OpenApplication(context.Context, string) error {
	return platform.ErrNotImplemented
}
func (stubPlatForTools) GrantedApplications() ([]string, error) {
	return nil, platform.ErrNotImplemented
}
func (stubPlatForTools) RequestAccess([]string, platform.AccessTier) (map[string]platform.AccessTier, error) {
	return nil, platform.ErrNotImplemented
}
func (stubPlatForTools) Tier(string) platform.AccessTier { return platform.TierFull }
func (stubPlatForTools) FrontmostApp() (string, platform.AccessTier, error) {
	return "", platform.TierFull, platform.ErrNotImplemented
}
func (stubPlatForTools) Confirm(string) (bool, error) { return false, platform.ErrNotImplemented }
func (stubPlatForTools) OCR(image.Image) ([]platform.OCRResult, error) {
	return nil, platform.ErrNotImplemented
}

func TestNewRegistry(t *testing.T) {
	var p stubPlatForTools
	r := NewRegistry(p)
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
	// Should have 27 tool names registered
	if len(r.specs) != 27 {
		t.Errorf("expected 27 specs, got %d", len(r.specs))
	}
}

func TestRegistrySpecs_Sorted(t *testing.T) {
	var p stubPlatForTools
	r := NewRegistry(p)
	specs := r.Specs()
	for i := 1; i < len(specs); i++ {
		if specs[i].Name < specs[i-1].Name {
			t.Errorf("Specs not sorted: %s before %s", specs[i-1].Name, specs[i].Name)
		}
	}
}

func TestRegistryNames_Sorted(t *testing.T) {
	var p stubPlatForTools
	r := NewRegistry(p)
	names := r.Names()
	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			t.Errorf("Names not sorted: %s before %s", names[i-1], names[i])
		}
	}
}

func TestRegistryCall_UnknownTool(t *testing.T) {
	var p stubPlatForTools
	r := NewRegistry(p)
	res, _ := r.Call(context.Background(), "nonexistent_tool", nil)
	if !res.IsError {
		t.Error("unknown tool should return IsError=true")
	}
}

func TestRegistryCall_NotImplemented(t *testing.T) {
	var p stubPlatForTools
	r := NewRegistry(p)
	// screenshot is implemented - verify it doesn't return "not implemented" text
	res, err := r.Call(context.Background(), "screenshot", nil)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.Text == "tool screenshot is not implemented yet" {
		t.Error("screenshot should be implemented")
	}
}

func TestRegistryCall_NilParams(t *testing.T) {
	var p stubPlatForTools
	r := NewRegistry(p)
	res, err := r.Call(context.Background(), "screenshot", nil)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	_ = res
}

func TestRegistryCall_ContextPlumbed(t *testing.T) {
	var p stubPlatForTools
	r := NewRegistry(p)
	for i := 0; i < 3; i++ {
		r.Call(context.Background(), "screenshot", nil)
	}
}
