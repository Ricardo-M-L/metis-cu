// Package tools registers the 29 computer-use tools and routes calls
// to the Platform layer. Tool names + parameter shapes deliberately
// mirror Anthropic's `mcp__computer-use__*` namespace so prompts,
// traces, and eval datasets carry over.
//
// Each tool is a small adapter:
//
//  1. parse JSON params from the MCP client (CallToolRequest)
//  2. enforce frontmost-app tier (read / click / full) where applicable
//  3. call the Platform method
//  4. shape the result back into MCP content blocks (text / image)
//
// Tier enforcement is centralized in `gate.go` so every mouse / keyboard
// tool inherits it for free.
package tools

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

// Result is the platform-neutral shape an adapter returns. The server
// package converts it into mcp-go's CallToolResult; keeping mcp-go
// out of pkg/tools means platform tests can assert handler output
// without spinning up the MCP wire.
type Result struct {
	// Text is the human-readable summary. Always set unless purely
	// image (and even then, set a one-line "captured 1280x800 PNG"
	// string so MCP clients that ignore image blocks still see
	// progress).
	Text string

	// Image, when non-empty, is base64-encoded image data (no data:
	// URL prefix). Pair with MIMEType.
	Image    string
	MIMEType string

	// IsError marks tool-level failures (bad params, platform refused,
	// tier denied). MCP wraps it into the result's isError flag rather
	// than returning a transport-level error so the LLM can see and
	// self-correct.
	IsError bool
}

// Handler runs one tool invocation. params is the raw JSON of the
// `arguments` object (always an object per JSON Schema; may be `{}`).
// Implementations decode into a struct or use ad-hoc map access — see
// per-tool files for the convention.
type Handler func(ctx context.Context, plat platform.Platform, params map[string]any) (*Result, error)

// Spec is the metadata the MCP server uses to advertise the tool plus
// the function that runs it. The schema source-of-truth is
// docs/api-spec.md — keep them in sync. Schema is hand-written
// map[string]any (no builder DSL) following metis's convention.
type Spec struct {
	Name        string
	Description string
	Schema      map[string]any
	Handler     Handler
}

// Registry holds the 29 tools plus the platform back-end. The MCP
// server iterates Specs() at handshake time to advertise the tool list,
// then routes Call(name, params) per invocation.
//
// Registry is safe for concurrent reads; specs is populated once at
// NewRegistry-time and never mutated afterwards. ScreenshotMaxW/H are
// runtime-configurable via SetScreenshotLimits — pkg/server reads
// ~/.metis-cu/config.toml at boot and overrides the defaults so a
// Retina display doesn't flood the LLM with native-resolution PNGs.
type Registry struct {
	plat  platform.Platform
	specs map[string]Spec

	// Screenshot downsampling cap. Defaults to 1280×800 (Anthropic CU
	// reference value) at NewRegistry-time; pkg/server may override
	// from the user's config.toml.
	ScreenshotMaxW int
	ScreenshotMaxH int

	// ScreenshotFormat selects the wire encoder: "png" (lossless,
	// default) or "jpeg" (3-5× smaller payload at q=85). Anything
	// else falls back to png in screenshotFormat(ctx). ScreenshotJPEGQ
	// is honoured only when the format is jpeg, and clamps to [1,100].
	ScreenshotFormat string
	ScreenshotJPEGQ  int

	// TypePasteThreshold (BUG-22): rune-count above which `type`
	// switches from per-key events to clipboard-paste. <= 0 means
	// "use the package default" (DefaultTypePasteThreshold = 80).
	TypePasteThreshold int

	// HoldKeyMaxMs (BUG-11 + DD-3): upper bound on hold_key duration.
	// 0 falls back to the package DefaultHoldKeyMaxMs.
	HoldKeyMaxMs int

	// ClipboardMaxBytes (BUG-15 + DD-3): cap on read_clipboard payload.
	ClipboardMaxBytes int

	// BatchMaxSteps (DD-3): cap on computer_batch step count.
	BatchMaxSteps int

	// ZoomMaxFactor (BUG-18 + DD-3): cap on zoom factor.
	ZoomMaxFactor float64

	// ZoomMaxOutputPixels (BUG-18 + DD-3): cap on zoom output pixel count.
	ZoomMaxOutputPixels int

	// MouseSettleMs (Tier-1 borrow): post-action quiescence delay
	// applied at the end of every successful mouse / keyboard / scroll
	// / type handler. 0 = disabled (matches pre-Tier-1 behaviour).
	// Pulled by mouseSettleFor(ctx) in limits.go.
	MouseSettleMs int
}

// DefaultScreenshotMaxW / H are the baked-in caps for screenshot
// downsampling — sane defaults so the registry is usable without any
// config file. Anthropic's computer-use docs cite 1280×800 as the
// recommended target for the 24-tool spec.
const (
	DefaultScreenshotMaxW = 1280
	DefaultScreenshotMaxH = 800
)

// SetScreenshotLimits overrides the per-screenshot cap. Non-positive
// values are silently ignored so an empty / malformed config can't
// produce a 0×0 image. Called once at boot from pkg/server after
// LoadConfig — runtime mutation otherwise should be avoided since
// Registry doesn't synchronise these reads (handler-side reads happen
// off the same goroutine that constructed the Registry).
func (r *Registry) SetScreenshotLimits(maxW, maxH int) {
	if maxW > 0 {
		r.ScreenshotMaxW = maxW
	}
	if maxH > 0 {
		r.ScreenshotMaxH = maxH
	}
}

// SetMouseSettleMs overrides the post-action quiescence delay. Negative
// values are silently coerced to 0 (disabled). 0 means "don't sleep" —
// the pre-Tier-1 default. Used by every action handler at the end of
// its successful path via settleAndMaybeShot().
func (r *Registry) SetMouseSettleMs(ms int) {
	if ms < 0 {
		ms = 0
	}
	r.MouseSettleMs = ms
}

// SetScreenshotFormat overrides the wire encoding. Format must be
// "png" or "jpeg" (anything else is silently ignored, leaving the
// previous value — boot-time default is "png"). Quality applies only
// to jpeg and clamps to [1,100]; out-of-range values are ignored.
func (r *Registry) SetScreenshotFormat(format string, quality int) {
	switch format {
	case "png", "jpeg":
		r.ScreenshotFormat = format
	}
	if quality >= 1 && quality <= 100 {
		r.ScreenshotJPEGQ = quality
	}
}

// SetTypePasteThreshold overrides the rune-count above which `type`
// switches to the paste path (BUG-22). Non-positive values are
// silently ignored, leaving the package default in place.
func (r *Registry) SetTypePasteThreshold(threshold int) {
	if threshold > 0 {
		r.TypePasteThreshold = threshold
	}
}

// SetLimits is a bulk setter for the DD-3 limit knobs. Non-positive
// values per field leave the package default intact, so a partial
// config stanza never inadvertently zeroes out a cap (which would
// reject every call past the now-impossible threshold).
func (r *Registry) SetLimits(holdKeyMaxMs, clipboardMaxBytes, batchMaxSteps int, zoomMaxFactor float64, zoomMaxOutputPixels int) {
	if holdKeyMaxMs > 0 {
		r.HoldKeyMaxMs = holdKeyMaxMs
	}
	if clipboardMaxBytes > 0 {
		r.ClipboardMaxBytes = clipboardMaxBytes
	}
	if batchMaxSteps > 0 {
		r.BatchMaxSteps = batchMaxSteps
	}
	if zoomMaxFactor > 0 {
		r.ZoomMaxFactor = zoomMaxFactor
	}
	if zoomMaxOutputPixels > 0 {
		r.ZoomMaxOutputPixels = zoomMaxOutputPixels
	}
}

// NewRegistry wires the platform implementation chosen by build-tag and
// declares all 29 specs. Tools without a Handler yet return a
// "not implemented" Result rather than panicking, so the server still
// serves a complete tools/list response while sprints fill in coverage.
func NewRegistry(plat platform.Platform) *Registry {
	r := &Registry{
		plat:           plat,
		specs:          make(map[string]Spec, len(allToolNames)),
		ScreenshotMaxW: DefaultScreenshotMaxW,
		ScreenshotMaxH: DefaultScreenshotMaxH,
	}
	for _, name := range allToolNames {
		// Default schema for not-yet-implemented tools: an empty
		// object that accepts arbitrary fields. Real adapters
		// replace this with their actual schema via init().
		r.specs[name] = Spec{
			Name:   name,
			Schema: defaultSchema(),
		}
	}
	for _, register := range registrations {
		register(r)
	}
	return r
}

// defaultSchema is the placeholder schema for tools that have a name
// registered in allToolNames but no adapter file yet. It stays valid
// JSON Schema so MCP clients don't reject the advertisement at
// tools/list time.
func defaultSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"properties":           map[string]any{},
		"additionalProperties": true,
	}
}

// Specs returns the 29 specs sorted by name. The MCP server uses this
// at tools/list time.
func (r *Registry) Specs() []Spec {
	out := make([]Spec, 0, len(r.specs))
	for _, s := range r.specs {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Names returns the 24 tool names, alphabetized — handy for the MCP
// tools/list response and for debugging.
func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.specs))
	for n := range r.specs {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Platform returns the underlying platform back-end. Adapters in the
// same package use it; external callers should treat Registry as
// opaque.
func (r *Registry) Platform() platform.Platform { return r.plat }

// registryKey is the unexported context key used to plumb the active
// Registry into a handler. Only computer_batch needs this — it reads
// the registry off the context to dispatch sub-steps without taking a
// constructor-time dependency on Registry (which would make tests that
// build adapter handlers in isolation harder).
type registryKey struct{}

// Call dispatches to the registered Handler. Unknown tool names return
// an "unknown tool" Result with IsError=true; tools with no Handler yet
// return a "not implemented" Result. Both shapes are visible to the
// LLM so it can self-correct rather than silently wedging.
//
// The active *Registry is attached to ctx under registryKey so handlers
// that need to fan out to sibling tools (computer_batch) can recover
// it. This avoids making every Handler signature carry a *Registry.
func (r *Registry) Call(ctx context.Context, name string, params map[string]any) (*Result, error) {
	spec, ok := r.specs[name]
	if !ok {
		return &Result{Text: fmt.Sprintf("unknown tool: %s", name), IsError: true}, nil
	}
	if spec.Handler == nil {
		return &Result{Text: fmt.Sprintf("tool %s is not implemented yet", name), IsError: true}, nil
	}
	if params == nil {
		params = map[string]any{}
	}
	if ctx.Value(registryKey{}) != r {
		ctx = context.WithValue(ctx, registryKey{}, r)
	}
	return spec.Handler(ctx, r.plat, params)
}

// register replaces the spec for a tool. Adapters call this from their
// init() (via the registrations slice) so the per-tool file is the
// source of truth for description + schema + handler.
func (r *Registry) register(spec Spec) {
	if _, ok := r.specs[spec.Name]; !ok {
		// Catches typos in adapter files — every registered name must
		// be in allToolNames or the registry diverges from the spec.
		panic("tools: register called for unknown tool name: " + spec.Name)
	}
	r.specs[spec.Name] = spec
}

// registrations is the list of per-tool init hooks. Adapter files call
// addRegistration in their init() so import order doesn't matter.
var (
	registrationsMu sync.Mutex
	registrations   []func(*Registry)
)

// addRegistration is called from per-tool init() functions to declare
// their Spec. Using a slice + NewRegistry replay (rather than a global
// map mutated at init time) keeps the registry per-instance — useful
// for tests that build their own Registry with a mock platform.
func addRegistration(fn func(*Registry)) {
	registrationsMu.Lock()
	defer registrationsMu.Unlock()
	registrations = append(registrations, fn)
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
	"screen_size",
	"find_text_on_screen",

	// mouse
	"mouse_move",
	"left_click",
	"click_text",
	"right_click",
	"middle_click",
	"double_click",
	"triple_click",
	"left_click_drag",
	"highlight_text_span",
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
	"list_windows",
	"request_access",

	// control / batch
	"wait",
	"computer_batch",
}
