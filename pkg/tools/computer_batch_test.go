package tools

import (
	"context"
	"errors"
	"image"
	"strings"
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

// recordPlat counts the platform calls computer_batch's sub-tools
// trigger so tests can assert the chain actually executed (not just
// that the result text looked right).
type recordPlat struct {
	stubPlat
	moves  int
	clicks int
}

func (p *recordPlat) MouseMove(platform.Point) error { p.moves++; return nil }
func (p *recordPlat) MouseClick(platform.Point, platform.Button, int) error {
	p.clicks++
	return nil
}
func (p *recordPlat) CursorPosition() (platform.Point, error) {
	return platform.Point{X: 1, Y: 2}, nil
}
func (p *recordPlat) Screenshot() (image.Image, error) {
	return image.NewRGBA(image.Rect(0, 0, 4, 4)), nil
}

// newBatchRegistry builds a minimal Registry around a fake platform.
// Init-time registrations populate every adapter, so the registry is
// ready to dispatch real tools (mouse_move, cursor_position, …).
func newBatchRegistry(plat platform.Platform) *Registry {
	return NewRegistry(plat)
}

func TestComputerBatch_RunsChain(t *testing.T) {
	plat := &recordPlat{}
	reg := newBatchRegistry(plat)
	params := map[string]any{
		"steps": []any{
			map[string]any{"tool": "mouse_move", "params": map[string]any{"x": float64(10), "y": float64(20)}},
			map[string]any{"tool": "left_click", "params": map[string]any{"x": float64(10), "y": float64(20)}},
			map[string]any{"tool": "cursor_position"},
		},
	}
	res, err := reg.Call(context.Background(), "computer_batch", params)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
	if plat.moves < 1 || plat.clicks < 1 {
		t.Errorf("expected mouse_move + left_click to fire; moves=%d clicks=%d", plat.moves, plat.clicks)
	}
	if !strings.Contains(res.Text, "step 0") || !strings.Contains(res.Text, "step 2") {
		t.Errorf("expected per-step summary, got %q", res.Text)
	}
}

func TestComputerBatch_AbortOnError(t *testing.T) {
	plat := &errClickPlat{}
	reg := newBatchRegistry(plat)
	params := map[string]any{
		"steps": []any{
			map[string]any{"tool": "mouse_move", "params": map[string]any{"x": float64(0), "y": float64(0)}},
			map[string]any{"tool": "left_click", "params": map[string]any{"x": float64(0), "y": float64(0)}},
			map[string]any{"tool": "mouse_move", "params": map[string]any{"x": float64(1), "y": float64(1)}},
		},
	}
	res, _ := reg.Call(context.Background(), "computer_batch", params)
	if !res.IsError {
		t.Fatal("expected IsError when a step fails")
	}
	if strings.Contains(res.Text, "step 2") {
		t.Errorf("third step should not run after error; got %q", res.Text)
	}
}

func TestComputerBatch_RejectsNested(t *testing.T) {
	reg := newBatchRegistry(&recordPlat{})
	params := map[string]any{
		"steps": []any{
			map[string]any{"tool": "computer_batch", "params": map[string]any{"steps": []any{}}},
		},
	}
	res, _ := reg.Call(context.Background(), "computer_batch", params)
	if !res.IsError {
		t.Fatal("expected IsError for nested computer_batch")
	}
	if !strings.Contains(res.Text, "nested") {
		t.Errorf("expected 'nested' in error, got %q", res.Text)
	}
}

func TestComputerBatch_OverCap(t *testing.T) {
	steps := make([]any, maxBatchSteps+1)
	for i := range steps {
		steps[i] = map[string]any{"tool": "cursor_position"}
	}
	reg := newBatchRegistry(&recordPlat{})
	res, _ := reg.Call(context.Background(), "computer_batch", map[string]any{"steps": steps})
	if !res.IsError {
		t.Fatal("expected IsError when steps exceeds cap")
	}
}

func TestComputerBatch_PropagatesImage(t *testing.T) {
	reg := newBatchRegistry(&recordPlat{})
	params := map[string]any{
		"steps": []any{
			map[string]any{"tool": "cursor_position"},
			map[string]any{"tool": "screenshot"},
		},
	}
	res, _ := reg.Call(context.Background(), "computer_batch", params)
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
	if res.Image == "" {
		t.Error("expected last screenshot's image to propagate to batch result")
	}
	if res.MIMEType != "image/png" {
		t.Errorf("unexpected mime: %q", res.MIMEType)
	}
}

// errClickPlat fails on left_click but not on mouse_move, so the
// abort-on-error path can be exercised mid-chain.
type errClickPlat struct {
	stubPlat
}

func (errClickPlat) MouseMove(platform.Point) error { return nil }
func (errClickPlat) MouseClick(platform.Point, platform.Button, int) error {
	return errors.New("synthetic click failure")
}
