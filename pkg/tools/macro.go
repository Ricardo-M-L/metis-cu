package tools

// macro.go — persistent tool-sequence macros. Stores ordered lists
// of cu-tool calls under ~/.metis-cu/macros/<name>.json so common
// flows ("open chrome to logged-in github", "navigate to settings >
// privacy", etc) can be replayed without per-step LLM round-trips.
//
// Design choice — explicit save, not auto-record. The MCP wire is
// stateless per call (each tools/call is independent), so a true
// "start recording → operate normally → stop recording" feature
// would need protocol-level hooks we don't have. Instead the LLM
// (or a human via the file) writes the macro file once, then any
// future session replays it with one tool call. Lower implementation
// cost, similar end-user value.
//
// Macro JSON shape:
//
//   { "name": "login-github",
//     "steps": [
//       {"tool": "browser_dom_outline", "args": {}},
//       {"tool": "browser_type", "args": {"selector":"#login_field","text":"alice"}},
//       {"tool": "browser_type", "args": {"selector":"#password","text":"<env:GH_PW>"}},
//       {"tool": "browser_click", "args": {"selector":"input[name=commit]"}}
//     ]
//   }
//
// Steps execute serially; first error aborts and the result text
// reports which step failed. Recursive macros (macro_play inside a
// macro) are rejected to prevent infinite loops.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

const (
	macroDirEnv   = "METIS_CU_MACRO_DIR"
	macroFileExt  = ".json"
	macroMaxSteps = 64
)

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name:        "macro_save",
			Description: "Persist a named sequence of cu tool calls to disk. Use AFTER you've worked out a flow interactively so subsequent sessions can replay it with one tool call instead of N. Path: ~/.metis-cu/macros/<name>.json (override dir via METIS_CU_MACRO_DIR). Recursive `macro_play` inside `steps` is rejected to prevent infinite loops.",
			Schema: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"name", "steps"},
				"properties": map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "Macro filename stem (no extension). Lowercase letters, digits, hyphens — anything else gets rejected so the file name is shell-safe.",
						"pattern":     "^[a-z0-9-]+$",
						"minLength":   1,
					},
					"steps": map[string]any{
						"type":        "array",
						"description": "Ordered list of {tool, args} entries to replay. Max 64 steps.",
						"minItems":    1,
						"maxItems":    macroMaxSteps,
						"items": map[string]any{
							"type":     "object",
							"required": []string{"tool"},
							"properties": map[string]any{
								"tool": map[string]any{"type": "string"},
								"args": map[string]any{"type": "object"},
							},
						},
					},
				},
			},
			Handler: r.handleMacroSave,
		})

		r.register(Spec{
			Name:        "macro_play",
			Description: "Replay a previously-saved macro by name. Executes each step serially against the same Platform / Registry the live session uses; aborts on the first step error and reports which step number failed. Use `macro_list` to discover available macros.",
			Schema: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"name"},
				"properties": map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "Macro name (the filename stem you saved under). See macro_list for available macros.",
						"pattern":     "^[a-z0-9-]+$",
						"minLength":   1,
					},
				},
			},
			Handler: r.handleMacroPlay,
		})

		r.register(Spec{
			Name:        "macro_list",
			Description: "List every saved macro under ~/.metis-cu/macros/. Returns name + step count + timestamp so the model can pick the right one before calling macro_play.",
			Schema: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					"_": map[string]any{
						"type":        "string",
						"description": "Pass any non-empty string (e.g. \"noop\"). MiniMax-quirk workaround for empty-args tool_use (see other no-arg cu tools).",
					},
				},
				"required": []string{"_"},
			},
			Handler: handleMacroList,
		})
	})
}

// macroSpec is the on-disk JSON shape.
type macroSpec struct {
	Name      string      `json:"name"`
	SavedAt   time.Time   `json:"saved_at"`
	Steps     []macroStep `json:"steps"`
	SourceTag string      `json:"source_tag,omitempty"` // optional caller-provided origin marker
}

type macroStep struct {
	Tool string         `json:"tool"`
	Args map[string]any `json:"args,omitempty"`
}

// macroDir returns the directory macros live in. Honors
// METIS_CU_MACRO_DIR for testing; defaults to ~/.metis-cu/macros.
func macroDir() (string, error) {
	if v := os.Getenv(macroDirEnv); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".metis-cu", "macros"), nil
}

func macroPath(name string) (string, error) {
	dir, err := macroDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name+macroFileExt), nil
}

// handleMacroSave validates + writes the macro file. Rejects
// recursive macro_play inside steps so a misconfigured macro can't
// trigger an infinite chain.
func (r *Registry) handleMacroSave(_ context.Context, _ platform.Platform, params map[string]any) (*Result, error) {
	name := paramStringOr(params, "name", "")
	if name == "" || !macroNameOK(name) {
		return &Result{Text: "macro_save: `name` is required and must match [a-z0-9-]+", IsError: true}, nil
	}
	rawSteps, ok := params["steps"].([]any)
	if !ok || len(rawSteps) == 0 {
		return &Result{Text: "macro_save: `steps` is required (non-empty array of {tool, args})", IsError: true}, nil
	}
	if len(rawSteps) > macroMaxSteps {
		return &Result{Text: fmt.Sprintf("macro_save: too many steps (%d > %d cap)", len(rawSteps), macroMaxSteps), IsError: true}, nil
	}
	steps := make([]macroStep, 0, len(rawSteps))
	for i, s := range rawSteps {
		m, ok := s.(map[string]any)
		if !ok {
			return &Result{Text: fmt.Sprintf("macro_save: step %d is not an object", i+1), IsError: true}, nil
		}
		tool, _ := m["tool"].(string)
		if tool == "" {
			return &Result{Text: fmt.Sprintf("macro_save: step %d missing `tool`", i+1), IsError: true}, nil
		}
		if tool == "macro_play" {
			return &Result{Text: fmt.Sprintf("macro_save: step %d uses `macro_play` — recursive macros are rejected", i+1), IsError: true}, nil
		}
		args, _ := m["args"].(map[string]any)
		steps = append(steps, macroStep{Tool: tool, Args: args})
	}

	spec := macroSpec{
		Name:    name,
		SavedAt: time.Now(),
		Steps:   steps,
	}
	dir, err := macroDir()
	if err != nil {
		return &Result{Text: fmt.Sprintf("macro_save: %v", err), IsError: true}, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return &Result{Text: fmt.Sprintf("macro_save: mkdir: %v", err), IsError: true}, nil
	}
	path, _ := macroPath(name)
	body, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return &Result{Text: fmt.Sprintf("macro_save: encode: %v", err), IsError: true}, nil
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return &Result{Text: fmt.Sprintf("macro_save: write %s: %v", path, err), IsError: true}, nil
	}
	return &Result{Text: fmt.Sprintf("saved macro %q with %d step(s) → %s", name, len(steps), path)}, nil
}

// handleMacroPlay reads + executes the macro file. Resolves each
// step's tool via the Registry's own Call path so all the same
// gating / argument coercion applies.
func (r *Registry) handleMacroPlay(ctx context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	name := paramStringOr(params, "name", "")
	if name == "" || !macroNameOK(name) {
		return &Result{Text: "macro_play: `name` is required and must match [a-z0-9-]+", IsError: true}, nil
	}
	path, err := macroPath(name)
	if err != nil {
		return &Result{Text: fmt.Sprintf("macro_play: %v", err), IsError: true}, nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return &Result{Text: fmt.Sprintf("macro_play: read %s: %v", path, err), IsError: true}, nil
	}
	var spec macroSpec
	if err := json.Unmarshal(body, &spec); err != nil {
		return &Result{Text: fmt.Sprintf("macro_play: decode %s: %v", path, err), IsError: true}, nil
	}
	if len(spec.Steps) == 0 {
		return &Result{Text: fmt.Sprintf("macro_play: %q has no steps", name), IsError: true}, nil
	}

	var summary strings.Builder
	fmt.Fprintf(&summary, "macro %q (%d steps):\n", name, len(spec.Steps))
	for i, step := range spec.Steps {
		// Recursion guard — even though save rejects macro_play in
		// steps, an older file might pre-date that check.
		if step.Tool == "macro_play" {
			fmt.Fprintf(&summary, "  step %d (%s): REJECTED — recursive macro_play not allowed\n", i+1, step.Tool)
			return &Result{Text: summary.String(), IsError: true}, nil
		}
		args := step.Args
		if args == nil {
			args = map[string]any{}
		}
		_ = plat // Registry.Call uses its own internal plat reference
		res, err := r.Call(ctx, step.Tool, args)
		if err != nil {
			fmt.Fprintf(&summary, "  step %d (%s): ERROR %v\n", i+1, step.Tool, err)
			return &Result{Text: summary.String(), IsError: true}, nil
		}
		if res.IsError {
			fmt.Fprintf(&summary, "  step %d (%s): IsError — %s\n", i+1, step.Tool, truncForMacroSummary(res.Text, 120))
			return &Result{Text: summary.String(), IsError: true}, nil
		}
		fmt.Fprintf(&summary, "  step %d (%s): ok — %s\n", i+1, step.Tool, truncForMacroSummary(res.Text, 80))
	}
	fmt.Fprintf(&summary, "all %d steps completed.", len(spec.Steps))
	return &Result{Text: summary.String()}, nil
}

func handleMacroList(_ context.Context, _ platform.Platform, _ map[string]any) (*Result, error) {
	dir, err := macroDir()
	if err != nil {
		return &Result{Text: fmt.Sprintf("macro_list: %v", err), IsError: true}, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return &Result{Text: "macro_list: no macros saved yet (use macro_save to create one)"}, nil
		}
		return &Result{Text: fmt.Sprintf("macro_list: %v", err), IsError: true}, nil
	}
	type row struct {
		Name    string    `json:"name"`
		Steps   int       `json:"steps"`
		SavedAt time.Time `json:"saved_at"`
	}
	var rows []row
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), macroFileExt) {
			continue
		}
		path := filepath.Join(dir, e.Name())
		body, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var spec macroSpec
		if json.Unmarshal(body, &spec) != nil {
			continue
		}
		rows = append(rows, row{Name: spec.Name, Steps: len(spec.Steps), SavedAt: spec.SavedAt})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	out, _ := json.Marshal(map[string]any{"count": len(rows), "macros": rows})
	return &Result{Text: string(out)}, nil
}

// macroNameOK enforces the [a-z0-9-]+ shell-safe constraint without
// pulling in regexp for one call.
func macroNameOK(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '-' {
			return false
		}
	}
	return true
}

// truncForMacroSummary trims long tool-result text for the per-step
// summary line so a multi-screen response stays on one line in the
// final macro result.
func truncForMacroSummary(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
