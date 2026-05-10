package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

type fakeMouseDownPlat struct {
	stubPlat
	downErr error
}

func (p *fakeMouseDownPlat) MouseDown(pt platform.Point, btn platform.Button) error {
	return p.downErr
}

func TestLeftMouseDown_OK(t *testing.T) {
	p := &fakeMouseDownPlat{}
	params := map[string]any{"x": 100, "y": 200}
	res, err := handleLeftMouseDown(context.Background(), p, params)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
}

func TestLeftMouseDown_MissingX(t *testing.T) {
	p := &fakeMouseDownPlat{}
	res, _ := handleLeftMouseDown(context.Background(), p, map[string]any{"y": 50})
	if !res.IsError {
		t.Fatal("expected IsError when x is missing")
	}
}

func TestLeftMouseDown_MissingY(t *testing.T) {
	p := &fakeMouseDownPlat{}
	res, _ := handleLeftMouseDown(context.Background(), p, map[string]any{"x": 50})
	if !res.IsError {
		t.Fatal("expected IsError when y is missing")
	}
}

func TestLeftMouseDown_PlatformError(t *testing.T) {
	p := &fakeMouseDownPlat{downErr: errors.New("mouse down failed")}
	params := map[string]any{"x": 10, "y": 20}
	res, _ := handleLeftMouseDown(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError when MouseDown fails")
	}
}
