package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

type fakeMouseMovePlat struct {
	stubPlat
	moveErr error
}

func (p *fakeMouseMovePlat) MouseMove(pt platform.Point) error { return p.moveErr }

func TestMouseMove_OK(t *testing.T) {
	p := &fakeMouseMovePlat{}
	params := map[string]any{"x": 100, "y": 200}
	res, err := handleMouseMove(context.Background(), p, params)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
}

func TestMouseMove_MissingX(t *testing.T) {
	p := &fakeMouseMovePlat{}
	res, _ := handleMouseMove(context.Background(), p, map[string]any{"y": 50})
	if !res.IsError {
		t.Fatal("expected IsError when x is missing")
	}
}

func TestMouseMove_MissingY(t *testing.T) {
	p := &fakeMouseMovePlat{}
	res, _ := handleMouseMove(context.Background(), p, map[string]any{"x": 50})
	if !res.IsError {
		t.Fatal("expected IsError when y is missing")
	}
}

func TestMouseMove_PlatformError(t *testing.T) {
	p := &fakeMouseMovePlat{moveErr: errors.New("move failed")}
	params := map[string]any{"x": 10, "y": 20}
	res, _ := handleMouseMove(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError when MouseMove fails")
	}
}
