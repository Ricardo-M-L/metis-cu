package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

// batchTool is the per-step shape inside `computer_batch`'s `steps`
// array. Mirrors the {tool, params} object the model emits.
type batchTool struct {
	Tool   string
	Params map[string]any
}

// maxBatchSteps caps the chain length so a runaway batch can't lock the
// session for thousands of synthetic events. 32 covers any realistic
// macro (open app → click → type → screenshot etc.) with headroom.
const maxBatchSteps = 32

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name: "computer_batch",
			Description: "Run a sequence of computer-use steps in one round-trip. " +
				"Each step is `{tool, params}` (same shapes as standalone calls). " +
				"Aborts on the first error. The final result text is a per-step " +
				"summary; if the last successful step produced an image, that image " +
				"is returned. Recursive `computer_batch` steps are rejected. " +
				"Capped at 32 steps.",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"steps": map[string]any{
						"type":     "array",
						"minItems": 1,
						"maxItems": maxBatchSteps,
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"tool": map[string]any{
									"type":        "string",
									"description": "Tool name to invoke (e.g. `left_click`, `type`).",
								},
								"params": map[string]any{
									"type":                 "object",
									"description":          "Arguments object for the tool. Same shape as a standalone call.",
									"additionalProperties": true,
								},
							},
							"required":             []string{"tool"},
							"additionalProperties": false,
						},
						"description": "Ordered list of steps to execute. The batch aborts on the first error.",
					},
				},
				"required":             []string{"steps"},
				"additionalProperties": false,
			},
			Handler: handleComputerBatchAdapter(),
		})
	})
}

// handleComputerBatchAdapter returns the handler closed over a
// nil registry — the real registry is bound at dispatch time via the
// context-injected dispatcher (see Registry.Call). We can't reach the
// Registry from a free handler the normal way because Handler signatures
// only see the platform; instead, we pull the active registry off the
// context value installed by Registry.Call. See registryKey usage below.
func handleComputerBatchAdapter() Handler {
	return func(ctx context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
		reg, ok := ctx.Value(registryKey{}).(*Registry)
		if !ok || reg == nil {
			return &Result{Text: "computer_batch: registry not bound to context (server bug)", IsError: true}, nil
		}
		return runBatch(ctx, reg, params)
	}
}

func runBatch(ctx context.Context, reg *Registry, params map[string]any) (*Result, error) {
	rawSteps, ok := params["steps"]
	if !ok {
		return &Result{Text: "missing required field: steps", IsError: true}, nil
	}
	stepList, ok := rawSteps.([]any)
	if !ok {
		return &Result{Text: fmt.Sprintf("steps: expected array, got %T", rawSteps), IsError: true}, nil
	}
	if len(stepList) == 0 {
		return &Result{Text: "steps: must contain at least one step", IsError: true}, nil
	}
	if len(stepList) > maxBatchSteps {
		return &Result{Text: fmt.Sprintf("steps: %d exceeds max %d", len(stepList), maxBatchSteps), IsError: true}, nil
	}

	steps := make([]batchTool, 0, len(stepList))
	for i, raw := range stepList {
		obj, ok := raw.(map[string]any)
		if !ok {
			return &Result{Text: fmt.Sprintf("steps[%d]: expected object, got %T", i, raw), IsError: true}, nil
		}
		name, err := requireString(obj, "tool")
		if err != nil {
			return &Result{Text: fmt.Sprintf("steps[%d]: %v", i, err), IsError: true}, nil
		}
		if name == "computer_batch" {
			return &Result{Text: fmt.Sprintf("steps[%d]: nested computer_batch is not allowed", i), IsError: true}, nil
		}
		var args map[string]any
		if rawArgs, present := obj["params"]; present {
			args, ok = rawArgs.(map[string]any)
			if !ok {
				return &Result{Text: fmt.Sprintf("steps[%d].params: expected object, got %T", i, rawArgs), IsError: true}, nil
			}
		}
		steps = append(steps, batchTool{Tool: name, Params: args})
	}

	var summary strings.Builder
	var lastImage, lastMIME string
	for i, step := range steps {
		if err := ctx.Err(); err != nil {
			fmt.Fprintf(&summary, "step %d (%s): cancelled (%v)\n", i, step.Tool, err)
			return &Result{Text: strings.TrimRight(summary.String(), "\n"), IsError: true}, nil
		}
		res, err := reg.Call(ctx, step.Tool, step.Params)
		if err != nil {
			fmt.Fprintf(&summary, "step %d (%s): transport error: %v\n", i, step.Tool, err)
			return &Result{Text: strings.TrimRight(summary.String(), "\n"), IsError: true}, nil
		}
		if res == nil {
			fmt.Fprintf(&summary, "step %d (%s): nil result\n", i, step.Tool)
			return &Result{Text: strings.TrimRight(summary.String(), "\n"), IsError: true}, nil
		}
		fmt.Fprintf(&summary, "step %d (%s): %s\n", i, step.Tool, oneLine(res.Text))
		if res.IsError {
			return &Result{Text: strings.TrimRight(summary.String(), "\n"), IsError: true}, nil
		}
		if res.Image != "" {
			lastImage = res.Image
			lastMIME = res.MIMEType
		}
	}
	out := &Result{Text: strings.TrimRight(summary.String(), "\n")}
	if lastImage != "" {
		out.Image = lastImage
		out.MIMEType = lastMIME
	}
	return out, nil
}

// oneLine collapses multi-line tool output into a single line so the
// per-step summary stays readable. Most tools return one line already;
// `computer_batch` itself is rejected as nested, so the only multi-line
// candidate would be a future tool returning structured text.
func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ↵ ")
	if len(s) > 200 {
		s = s[:197] + "..."
	}
	return s
}
