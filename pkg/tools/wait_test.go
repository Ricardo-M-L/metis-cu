package tools

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestWait_Zero(t *testing.T) {
	res, err := handleWait(context.Background(), stubPlat{}, map[string]any{"ms": float64(0)})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
	if !strings.Contains(res.Text, "0ms") {
		t.Errorf("unexpected text: %q", res.Text)
	}
}

func TestWait_Sleeps(t *testing.T) {
	start := time.Now()
	res, _ := handleWait(context.Background(), stubPlat{}, map[string]any{"ms": float64(40)})
	elapsed := time.Since(start)
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
	if elapsed < 30*time.Millisecond {
		t.Errorf("wait returned in %v, expected ~40ms", elapsed)
	}
}

func TestWait_OverCap(t *testing.T) {
	res, _ := handleWait(context.Background(), stubPlat{}, map[string]any{"ms": float64(maxWaitMs + 1)})
	if !res.IsError {
		t.Fatal("expected IsError when ms exceeds cap")
	}
}

func TestWait_Negative(t *testing.T) {
	res, _ := handleWait(context.Background(), stubPlat{}, map[string]any{"ms": float64(-5)})
	if !res.IsError {
		t.Fatal("expected IsError for negative ms")
	}
}

func TestWait_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	res, _ := handleWait(ctx, stubPlat{}, map[string]any{"ms": float64(2000)})
	elapsed := time.Since(start)
	if !res.IsError {
		t.Fatal("expected IsError on context cancel")
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("wait did not honor cancel; took %v", elapsed)
	}
}
