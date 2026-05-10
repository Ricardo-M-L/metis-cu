package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

// fakePlat implements platform.Platform for double_click tests.
// It returns the configured click error, if any.
type fakePlat struct {
	stubPlat
	clickErr error
}

func (p *fakePlat) MouseClick(pt platform.Point, btn platform.Button, count int) error {
	return p.clickErr
}

func TestDoubleClick_OK(t *testing.T) {
	p := &fakePlat{}
	params := map[string]any{"x": 100, "y": 200}
	res, err := handleDoubleClick(context.Background(), p, params)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
}

func TestDoubleClick_MissingX(t *testing.T) {
	p := &fakePlat{}
	params := map[string]any{"y": 200}
	res, _ := handleDoubleClick(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError when x is missing")
	}
}

func TestDoubleClick_MissingY(t *testing.T) {
	p := &fakePlat{}
	params := map[string]any{"x": 100}
	res, _ := handleDoubleClick(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError when y is missing")
	}
}

func TestDoubleClick_PlatformError(t *testing.T) {
	p := &fakePlat{clickErr: errors.New("simulated failure")}
	params := map[string]any{"x": 50, "y": 60}
	res, _ := handleDoubleClick(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError when MouseClick fails")
	}
}
