package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

type fakeScrollPlat struct {
	stubPlat
	scrollErr error
}

func (p *fakeScrollPlat) Scroll(pt platform.Point, dx, dy int) error { return p.scrollErr }

func TestScroll_OK(t *testing.T) {
	p := &fakeScrollPlat{}
	params := map[string]any{"x": 100, "y": 200, "dx": 0, "dy": -3}
	res, err := handleScroll(context.Background(), p, params)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
}

func TestScroll_MissingX(t *testing.T) {
	p := &fakeScrollPlat{}
	res, _ := handleScroll(context.Background(), p, map[string]any{"y": 200, "dx": 0, "dy": 1})
	if !res.IsError {
		t.Fatal("expected IsError when x is missing")
	}
}

func TestScroll_MissingY(t *testing.T) {
	p := &fakeScrollPlat{}
	res, _ := handleScroll(context.Background(), p, map[string]any{"x": 100, "dx": 0, "dy": 1})
	if !res.IsError {
		t.Fatal("expected IsError when y is missing")
	}
}

func TestScroll_MissingDx(t *testing.T) {
	p := &fakeScrollPlat{}
	res, _ := handleScroll(context.Background(), p, map[string]any{"x": 100, "y": 200, "dy": 1})
	if !res.IsError {
		t.Fatal("expected IsError when dx is missing")
	}
}

func TestScroll_MissingDy(t *testing.T) {
	p := &fakeScrollPlat{}
	res, _ := handleScroll(context.Background(), p, map[string]any{"x": 100, "y": 200, "dx": 0})
	if !res.IsError {
		t.Fatal("expected IsError when dy is missing")
	}
}

func TestScroll_PlatformError(t *testing.T) {
	p := &fakeScrollPlat{scrollErr: errors.New("scroll failed")}
	params := map[string]any{"x": 10, "y": 20, "dx": 0, "dy": 5}
	res, _ := handleScroll(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError when Scroll fails")
	}
}
