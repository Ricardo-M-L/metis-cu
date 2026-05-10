package tools

import (
	"context"
	"errors"
	"testing"
)

type fakeHoldKeyPlat struct {
	stubPlat
	holdErr error
}

func (p *fakeHoldKeyPlat) KeyHold(_ context.Context, combo string, ms int) error {
	return p.holdErr
}

func TestHoldKey_OK(t *testing.T) {
	p := &fakeHoldKeyPlat{}
	params := map[string]any{"combo": "shift", "ms": 500}
	res, err := handleHoldKey(context.Background(), p, params)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
}

func TestHoldKey_MissingCombo(t *testing.T) {
	p := &fakeHoldKeyPlat{}
	res, _ := handleHoldKey(context.Background(), p, map[string]any{"ms": 100})
	if !res.IsError {
		t.Fatal("expected IsError when combo is missing")
	}
}

func TestHoldKey_MissingMs(t *testing.T) {
	p := &fakeHoldKeyPlat{}
	res, _ := handleHoldKey(context.Background(), p, map[string]any{"combo": "ctrl"})
	if !res.IsError {
		t.Fatal("expected IsError when ms is missing")
	}
}

func TestHoldKey_ZeroMs(t *testing.T) {
	p := &fakeHoldKeyPlat{}
	params := map[string]any{"combo": "a", "ms": 0}
	res, _ := handleHoldKey(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError when ms < 1")
	}
}

func TestHoldKey_NegativeMs(t *testing.T) {
	p := &fakeHoldKeyPlat{}
	params := map[string]any{"combo": "b", "ms": -1}
	res, _ := handleHoldKey(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError when ms < 1")
	}
}

func TestHoldKey_PlatformError(t *testing.T) {
	p := &fakeHoldKeyPlat{holdErr: errors.New("key hold failed")}
	params := map[string]any{"combo": "cmd", "ms": 300}
	res, _ := handleHoldKey(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError when KeyHold fails")
	}
}

// TestHoldKey_RejectsOverMax: a hallucinated giant ms must be rejected
// before reaching the platform — otherwise a 24-hour hold would wedge
// modifiers and block the MCP request for the same duration (BUG-11).
func TestHoldKey_RejectsOverMax(t *testing.T) {
	p := &fakeHoldKeyPlat{}
	params := map[string]any{"combo": "shift", "ms": 86400000} // 1 day
	res, _ := handleHoldKey(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError for ms over cap")
	}
}

// TestHoldKey_AcceptsAtMax: the boundary ms = holdKeyMaxMs is allowed
// (cap is inclusive — caller asking for exactly the cap shouldn't be
// rejected for an off-by-one).
func TestHoldKey_AcceptsAtMax(t *testing.T) {
	p := &fakeHoldKeyPlat{}
	params := map[string]any{"combo": "ctrl", "ms": holdKeyMaxMs}
	res, _ := handleHoldKey(context.Background(), p, params)
	if res.IsError {
		t.Fatalf("expected ms=%d to be accepted, got IsError: %s", holdKeyMaxMs, res.Text)
	}
}
