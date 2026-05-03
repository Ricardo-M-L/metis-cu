package tools

import (
	"context"
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

type cursorPlat struct {
	stubPlat
	pt  platform.Point
	err error
}

func (p *cursorPlat) CursorPosition() (platform.Point, error) {
	return p.pt, p.err
}

func TestCursorPosition_OK(t *testing.T) {
	res, err := handleCursorPosition(context.Background(), &cursorPlat{pt: platform.Point{X: 320, Y: 240}}, nil)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
	if res.Text != `{"x":320,"y":240}` {
		t.Errorf("unexpected text: %q", res.Text)
	}
}

func TestCursorPosition_PlatformError(t *testing.T) {
	res, err := handleCursorPosition(context.Background(), &cursorPlat{err: platform.ErrNotImplemented}, nil)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError when platform fails")
	}
}
