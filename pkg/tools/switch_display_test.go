package tools

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type switchPlat struct {
	stubPlat
	called int
	err    error
}

func (p *switchPlat) SwitchDisplay(int) error {
	p.called++
	return p.err
}

func TestSwitchDisplay_OK(t *testing.T) {
	p := &switchPlat{}
	res, err := handleSwitchDisplay(context.Background(), p, map[string]any{"display": float64(1)})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
	if p.called != 1 {
		t.Fatalf("SwitchDisplay called %d times, want 1", p.called)
	}
	if !strings.Contains(res.Text, `"display":1`) {
		t.Errorf("expected display index in result, got %q", res.Text)
	}
}

func TestSwitchDisplay_MissingParam(t *testing.T) {
	res, _ := handleSwitchDisplay(context.Background(), &switchPlat{}, map[string]any{})
	if !res.IsError {
		t.Fatal("expected IsError when display field missing")
	}
}

func TestSwitchDisplay_PlatformError(t *testing.T) {
	p := &switchPlat{err: errors.New("display 9 out of range")}
	res, _ := handleSwitchDisplay(context.Background(), p, map[string]any{"display": float64(9)})
	if !res.IsError {
		t.Fatal("expected IsError when platform refuses")
	}
	if !strings.Contains(res.Text, "out of range") {
		t.Errorf("expected platform error text, got %q", res.Text)
	}
}
