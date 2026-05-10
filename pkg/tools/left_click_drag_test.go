package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

type fakeDragPlat struct {
	stubPlat
	dragErr error
}

func (p *fakeDragPlat) MouseDrag(_ context.Context, from, to platform.Point, btn platform.Button) error {
	return p.dragErr
}

func TestLeftClickDrag_OK(t *testing.T) {
	p := &fakeDragPlat{}
	params := map[string]any{
		"from": map[string]any{"x": 100, "y": 200},
		"to":   map[string]any{"x": 300, "y": 400},
	}
	res, err := handleLeftClickDrag(context.Background(), p, params)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
}

func TestLeftClickDrag_MissingFrom(t *testing.T) {
	p := &fakeDragPlat{}
	params := map[string]any{
		"to": map[string]any{"x": 300, "y": 400},
	}
	res, _ := handleLeftClickDrag(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError when from is missing")
	}
}

func TestLeftClickDrag_MissingTo(t *testing.T) {
	p := &fakeDragPlat{}
	params := map[string]any{
		"from": map[string]any{"x": 100, "y": 200},
	}
	res, _ := handleLeftClickDrag(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError when to is missing")
	}
}

func TestLeftClickDrag_FromPointMissingX(t *testing.T) {
	p := &fakeDragPlat{}
	params := map[string]any{
		"from": map[string]any{"y": 200},
		"to":   map[string]any{"x": 300, "y": 400},
	}
	res, _ := handleLeftClickDrag(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError when from.x is missing")
	}
}

func TestLeftClickDrag_PlatformError(t *testing.T) {
	p := &fakeDragPlat{dragErr: errors.New("drag failed")}
	params := map[string]any{
		"from": map[string]any{"x": 10, "y": 20},
		"to":   map[string]any{"x": 100, "y": 200},
	}
	res, _ := handleLeftClickDrag(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError when MouseDrag fails")
	}
}
