// Package tools registers the 24 computer-use tools and routes calls
// to the Platform layer. Tool names + parameter shapes deliberately
// mirror Anthropic's `mcp__computer-use__*` namespace so prompts,
// traces, and eval datasets carry over.
//
// Each tool is a small adapter:
//
//	1. parse JSON params from the MCP client
//	2. enforce frontmost-app tier (read / click / full) where applicable
//	3. call the Platform method
//	4. shape the result back into MCP content blocks
//
// Tier enforcement is centralized in `gate.go` so every mouse / keyboard
// tool inherits it for free.
package tools

import (
	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

// Spec is the metadata MCP server uses to advertise the tool: name,
// description, parameter JSON-Schema. The schema source-of-truth is
// docs/api-spec.md — keep them in sync.
type Spec struct {
	Name        string
	Description string
	Schema      map[string]any
}

// Registry holds the 24 tools plus the platform back-end. The MCP
// server iterates Specs() at handshake time to advertise the tool list,
// then routes Call(name, params) per invocation.
type Registry struct {
	plat  platform.Platform
	specs map[string]Spec
}

// NewRegistry wires the platform implementation chosen by build-tag and
// declares all 24 specs. Filling in the Schema map is Sprint 2's job —
// for now the Name lists are the contract.
func NewRegistry(plat platform.Platform) *Registry {
	r := &Registry{plat: plat, specs: map[string]Spec{}}
	for _, name := range allToolNames {
		r.specs[name] = Spec{Name: name}
	}
	return r
}

// Names returns the 24 tool names, alphabetized — handy for the MCP
// `tools/list` response and for debugging.
func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.specs))
	for n := range r.specs {
		out = append(out, n)
	}
	// stable order — caller can sort if it wants display order
	return out
}

// allToolNames is the canonical list, mirroring Anthropic's
// `mcp__computer-use__*` exactly. Order groups by category so a reader
// scanning the source sees the API surface at a glance.
var allToolNames = []string{
	// vision
	"screenshot",
	"cursor_position",
	"switch_display",
	"zoom",

	// mouse
	"mouse_move",
	"left_click",
	"right_click",
	"middle_click",
	"double_click",
	"triple_click",
	"left_click_drag",
	"left_mouse_down",
	"left_mouse_up",
	"scroll",

	// keyboard
	"key",
	"hold_key",
	"type",

	// clipboard
	"read_clipboard",
	"write_clipboard",

	// application
	"open_application",
	"list_granted_applications",
	"request_access",

	// control / batch
	"wait",
	"computer_batch",
}
