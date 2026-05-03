package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

// maxWaitMs caps server-side sleeps so a runaway / hallucinated wait
// can't wedge the MCP session for hours. 60s is generous for any
// "wait for the page to load" pattern; longer waits should be split
// into a polling loop client-side.
const maxWaitMs = 60_000

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name: "wait",
			Description: "Sleep `ms` milliseconds server-side. Cheaper than client-side " +
				"polling because no MCP round-trips happen during the wait. Capped at " +
				"60s — for longer pauses, call `wait` repeatedly. Cancellable: if the " +
				"MCP context is cancelled mid-wait, the call returns early with an error.",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"ms": map[string]any{
						"type":        "integer",
						"minimum":     0,
						"maximum":     maxWaitMs,
						"description": "Milliseconds to sleep. 0 returns immediately. Capped at 60000.",
					},
				},
				"required":             []string{"ms"},
				"additionalProperties": false,
			},
			Handler: handleWait,
		})
	})
}

func handleWait(ctx context.Context, _ platform.Platform, params map[string]any) (*Result, error) {
	ms, err := requireInt(params, "ms")
	if err != nil {
		return &Result{Text: fmt.Sprintf("invalid ms: %v", err), IsError: true}, nil
	}
	if ms < 0 {
		return &Result{Text: fmt.Sprintf("invalid ms: %d (must be >= 0)", ms), IsError: true}, nil
	}
	if ms > maxWaitMs {
		return &Result{Text: fmt.Sprintf("ms %d exceeds max %d (call wait repeatedly for longer pauses)",
			ms, maxWaitMs), IsError: true}, nil
	}
	if ms == 0 {
		return &Result{Text: "waited 0ms"}, nil
	}
	timer := time.NewTimer(time.Duration(ms) * time.Millisecond)
	defer timer.Stop()
	select {
	case <-timer.C:
		return &Result{Text: fmt.Sprintf("waited %dms", ms)}, nil
	case <-ctx.Done():
		return &Result{Text: fmt.Sprintf("wait cancelled after partial sleep: %v", ctx.Err()), IsError: true}, nil
	}
}
