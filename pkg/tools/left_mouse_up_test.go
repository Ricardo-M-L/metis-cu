package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

type fakeMouseUpPlat struct {
	stubPlat
	upErr error
}

func (p *fakeMouseUpPlat) MouseUp(pt platform.Point, btn platform.Button) error {
	return p.upErr
}

func TestLeftMouseUp_OK(t *testing.T) {
	p := &fakeMouseUpPlat{}
	params := map[string]any{"x": 100, "y": 200}
	res, err := handleLeftMouseUp(context.Background(), p, params)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
}

func TestLeftMouseUp_MissingX(t *testing.T) {
	p := &fakeMouseUpPlat{}
	res, _ := handleLeftMouseUp(context.Background(), p, map[string]any{"y": 50})
	if !res.IsError {
		t.Fatal("expected IsError when x is missing")
	}
}

func TestLeftMouseUp_MissingY(t *testing.T) {
	p := &fakeMouseUpPlat{}
	res, _ := handleLeftMouseUp(context.Background(), p, map[string]any{"x": 50})
	if !res.IsError {
		t.Fatal("expected IsError when y is missing")
	}
}

func TestLeftMouseUp_PlatformError(t *testing.T) {
	p := &fakeMouseUpPlat{upErr: errors.New("mouse up failed")}
	params := map[string]any{"x": 10, "y": 20}
	res, _ := handleLeftMouseUp(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError when MouseUp fails")
	}
}
