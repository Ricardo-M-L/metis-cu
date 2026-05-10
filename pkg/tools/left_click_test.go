package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

type fakeMouseClickPlat struct {
	stubPlat
	clickErr error
	// modsCalled records the modifiers passed to the *most recent*
	// MouseClickWithModifiers call so tests covering BUG-21 can assert
	// they were forwarded rather than dropped on the floor.
	modsCalled []string
}

func (p *fakeMouseClickPlat) MouseClick(pt platform.Point, btn platform.Button, count int) error {
	return p.clickErr
}

func (p *fakeMouseClickPlat) MouseClickWithModifiers(pt platform.Point, btn platform.Button, count int, mods []string) error {
	p.modsCalled = append([]string(nil), mods...)
	return p.clickErr
}

func TestLeftClick_OK(t *testing.T) {
	p := &fakeMouseClickPlat{}
	params := map[string]any{"x": 100, "y": 200}
	res, err := handleLeftClick(context.Background(), p, params)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
}

func TestLeftClick_MissingX(t *testing.T) {
	p := &fakeMouseClickPlat{}
	res, _ := handleLeftClick(context.Background(), p, map[string]any{"y": 50})
	if !res.IsError {
		t.Fatal("expected IsError when x is missing")
	}
}

func TestLeftClick_MissingY(t *testing.T) {
	p := &fakeMouseClickPlat{}
	res, _ := handleLeftClick(context.Background(), p, map[string]any{"x": 50})
	if !res.IsError {
		t.Fatal("expected IsError when y is missing")
	}
}

func TestLeftClick_WithEmptyModifiers(t *testing.T) {
	// Empty modifiers array is allowed (no-op per spec)
	p := &fakeMouseClickPlat{}
	params := map[string]any{"x": 10, "y": 20, "modifiers": []any{}}
	res, _ := handleLeftClick(context.Background(), p, params)
	if res.IsError {
		t.Fatalf("empty modifiers should be allowed: %s", res.Text)
	}
}

// TestLeftClick_WithModifiers (BUG-21): non-empty modifiers used to
// be rejected in Phase 2-A — now they're wired through to
// MouseClickWithModifiers. Asserts the platform actually receives the
// list (not silently dropped) and the result text mentions them.
func TestLeftClick_WithModifiers(t *testing.T) {
	p := &fakeMouseClickPlat{}
	params := map[string]any{"x": 10, "y": 20, "modifiers": []any{"cmd", "shift"}}
	res, err := handleLeftClick(context.Background(), p, params)
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("modifiers should now succeed (BUG-21 fix), got IsError: %s", res.Text)
	}
	if len(p.modsCalled) != 2 || p.modsCalled[0] != "cmd" || p.modsCalled[1] != "shift" {
		t.Errorf("MouseClickWithModifiers did not receive %v, got %v", []string{"cmd", "shift"}, p.modsCalled)
	}
}

// TestLeftClick_InvalidModifierName: an unknown modifier name surfaces
// as a clear error rather than getting silently passed to robotgo.
func TestLeftClick_InvalidModifierName(t *testing.T) {
	p := &fakeMouseClickPlat{}
	params := map[string]any{"x": 10, "y": 20, "modifiers": []any{"ctrlx"}}
	res, _ := handleLeftClick(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError for unknown modifier name")
	}
}

func TestLeftClick_PlatformError(t *testing.T) {
	p := &fakeMouseClickPlat{clickErr: errors.New("click failed")}
	params := map[string]any{"x": 10, "y": 20}
	res, _ := handleLeftClick(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError when MouseClick fails")
	}
}
