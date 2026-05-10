package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

type fakeRightClickPlat struct {
	stubPlat
	clickErr error
}

func (p *fakeRightClickPlat) MouseClick(pt platform.Point, btn platform.Button, count int) error {
	return p.clickErr
}

func TestRightClick_OK(t *testing.T) {
	p := &fakeRightClickPlat{}
	params := map[string]any{"x": 100, "y": 200}
	res, err := handleRightClick(context.Background(), p, params)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
}

func TestRightClick_MissingX(t *testing.T) {
	p := &fakeRightClickPlat{}
	res, _ := handleRightClick(context.Background(), p, map[string]any{"y": 50})
	if !res.IsError {
		t.Fatal("expected IsError when x is missing")
	}
}

func TestRightClick_MissingY(t *testing.T) {
	p := &fakeRightClickPlat{}
	res, _ := handleRightClick(context.Background(), p, map[string]any{"x": 50})
	if !res.IsError {
		t.Fatal("expected IsError when y is missing")
	}
}

func TestRightClick_PlatformError(t *testing.T) {
	p := &fakeRightClickPlat{clickErr: errors.New("right click failed")}
	params := map[string]any{"x": 10, "y": 20}
	res, _ := handleRightClick(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError when MouseClick fails")
	}
}
