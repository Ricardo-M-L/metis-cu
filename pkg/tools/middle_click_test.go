package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

type fakeMiddleClickPlat struct {
	stubPlat
	clickErr error
}

func (p *fakeMiddleClickPlat) MouseClick(pt platform.Point, btn platform.Button, count int) error {
	return p.clickErr
}

func TestMiddleClick_OK(t *testing.T) {
	p := &fakeMiddleClickPlat{}
	params := map[string]any{"x": 100, "y": 200}
	res, err := handleMiddleClick(context.Background(), p, params)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
}

func TestMiddleClick_MissingX(t *testing.T) {
	p := &fakeMiddleClickPlat{}
	res, _ := handleMiddleClick(context.Background(), p, map[string]any{"y": 50})
	if !res.IsError {
		t.Fatal("expected IsError when x is missing")
	}
}

func TestMiddleClick_MissingY(t *testing.T) {
	p := &fakeMiddleClickPlat{}
	res, _ := handleMiddleClick(context.Background(), p, map[string]any{"x": 50})
	if !res.IsError {
		t.Fatal("expected IsError when y is missing")
	}
}

func TestMiddleClick_PlatformError(t *testing.T) {
	p := &fakeMiddleClickPlat{clickErr: errors.New("middle click failed")}
	params := map[string]any{"x": 10, "y": 20}
	res, _ := handleMiddleClick(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError when MouseClick fails")
	}
}
