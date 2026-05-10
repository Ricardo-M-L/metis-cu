package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

type fakeTripleClickPlat struct {
	stubPlat
	clickErr error
}

func (p *fakeTripleClickPlat) MouseClick(pt platform.Point, btn platform.Button, count int) error {
	return p.clickErr
}

func TestTripleClick_OK(t *testing.T) {
	p := &fakeTripleClickPlat{}
	params := map[string]any{"x": 100, "y": 200}
	res, err := handleTripleClick(context.Background(), p, params)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
}

func TestTripleClick_MissingX(t *testing.T) {
	p := &fakeTripleClickPlat{}
	res, _ := handleTripleClick(context.Background(), p, map[string]any{"y": 50})
	if !res.IsError {
		t.Fatal("expected IsError when x is missing")
	}
}

func TestTripleClick_MissingY(t *testing.T) {
	p := &fakeTripleClickPlat{}
	res, _ := handleTripleClick(context.Background(), p, map[string]any{"x": 50})
	if !res.IsError {
		t.Fatal("expected IsError when y is missing")
	}
}

func TestTripleClick_PlatformError(t *testing.T) {
	p := &fakeTripleClickPlat{clickErr: errors.New("triple click failed")}
	params := map[string]any{"x": 10, "y": 20}
	res, _ := handleTripleClick(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError when MouseClick fails")
	}
}
