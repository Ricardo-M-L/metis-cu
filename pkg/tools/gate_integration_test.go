package tools

// Integration tests for the per-handler gateOrDeny wiring. Unit tests
// in gate_test.go already cover EnforceTier / tierAllows in isolation;
// these tests confirm that an actual handler short-circuits when the
// frontmost-app tier is too low — which is the change that turns the
// gate from dead code (pre-Phase-3) into an enforced safety boundary.

import (
	"context"
	"strings"
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

// gatedFakePlat is a Platform whose FrontmostApp returns a fixed
// (name, tier) pair AND whose mouse/keyboard methods are no-op
// successes — so a deny can be cleanly attributed to the gate, not
// to a downstream platform error.
type gatedFakePlat struct {
	stubPlat
	name string
	tier platform.AccessTier
}

func (p *gatedFakePlat) FrontmostApp() (string, platform.AccessTier, error) {
	return p.name, p.tier, nil
}
func (p *gatedFakePlat) MouseClick(platform.Point, platform.Button, int) error { return nil }
func (p *gatedFakePlat) MouseClickWithModifiers(platform.Point, platform.Button, int, []string) error {
	return nil
}
func (p *gatedFakePlat) MouseMove(platform.Point) error                  { return nil }
func (p *gatedFakePlat) MouseDown(platform.Point, platform.Button) error { return nil }
func (p *gatedFakePlat) MouseUp(platform.Point, platform.Button) error   { return nil }
func (p *gatedFakePlat) MouseDrag(context.Context, platform.Point, platform.Point, platform.Button) error {
	return nil
}
func (p *gatedFakePlat) Scroll(platform.Point, int, int) error { return nil }
func (p *gatedFakePlat) ScrollWithModifiers(platform.Point, int, int, []string) error {
	return nil
}
func (p *gatedFakePlat) KeyPress(string) error                         { return nil }
func (p *gatedFakePlat) KeyHold(context.Context, string, int) error    { return nil }
func (p *gatedFakePlat) Type(context.Context, string) error            { return nil }
func (p *gatedFakePlat) ClipboardWrite(string) error                   { return nil }
func (p *gatedFakePlat) OpenApplication(context.Context, string) error { return nil }

// gatedHandlerCase pairs a handler with the params it expects and the
// minimum tier it should require — driven off requiredTier so the
// tests stay aligned with the source-of-truth table.
type gatedHandlerCase struct {
	name    string
	handler Handler
	params  map[string]any
}

var gatedHandlerCases = []gatedHandlerCase{
	{"left_click", handleLeftClick, map[string]any{"x": 10, "y": 20}},
	{"right_click", handleRightClick, map[string]any{"x": 10, "y": 20}},
	{"middle_click", handleMiddleClick, map[string]any{"x": 10, "y": 20}},
	{"double_click", handleDoubleClick, map[string]any{"x": 10, "y": 20}},
	{"triple_click", handleTripleClick, map[string]any{"x": 10, "y": 20}},
	{"left_mouse_down", handleLeftMouseDown, map[string]any{"x": 10, "y": 20}},
	{"left_mouse_up", handleLeftMouseUp, map[string]any{"x": 10, "y": 20}},
	{"mouse_move", handleMouseMove, map[string]any{"x": 10, "y": 20}},
	{"left_click_drag", handleLeftClickDrag, map[string]any{
		"from": map[string]any{"x": 1, "y": 2},
		"to":   map[string]any{"x": 3, "y": 4},
	}},
	{"scroll", handleScroll, map[string]any{"x": 10, "y": 20, "dx": 0, "dy": 1}},
	{"key", handleKey, map[string]any{"combo": "a"}},
	{"hold_key", handleHoldKey, map[string]any{"combo": "shift", "ms": 50}},
	{"type", handleTypeText, map[string]any{"text": "hi"}},
	{"write_clipboard", handleWriteClipboard, map[string]any{"text": "x"}},
	{"open_application", handleOpenApplication, map[string]any{"name": "Safari"}},
}

// TestGateIntegration_DeniesBelowTier walks every gated handler with a
// frontmost app whose tier is too low and asserts the handler returns
// IsError without calling the platform. Catches future regressions
// where someone adds a handler but forgets the gateOrDeny line.
func TestGateIntegration_DeniesBelowTier(t *testing.T) {
	for _, tc := range gatedHandlerCases {
		t.Run(tc.name, func(t *testing.T) {
			required, ok := requiredTier[tc.name]
			if !ok {
				t.Fatalf("tool %q missing from requiredTier table", tc.name)
			}
			// Pick a tier strictly below what's required so the gate
			// must reject. TierClick required → TierRead is below;
			// TierFull required → TierClick is below.
			var below platform.AccessTier
			switch required {
			case platform.TierClick:
				below = platform.TierRead
			case platform.TierFull:
				below = platform.TierClick
			default:
				t.Fatalf("unexpected required tier %q for %q", required, tc.name)
			}
			plat := &gatedFakePlat{name: "GatedApp", tier: below}
			res, err := tc.handler(context.Background(), plat, tc.params)
			if err != nil {
				t.Fatalf("transport error: %v", err)
			}
			if !res.IsError {
				t.Fatalf("expected IsError when tier is below required; got %q", res.Text)
			}
			if !strings.Contains(res.Text, "GatedApp") {
				t.Errorf("expected app name in deny message, got %q", res.Text)
			}
			if !strings.Contains(res.Text, string(below)) {
				t.Errorf("expected current tier %q in deny message, got %q", below, res.Text)
			}
		})
	}
}

// TestGateIntegration_AllowsAtOrAboveTier confirms the same handlers
// dispatch successfully when frontmost is TierFull.
func TestGateIntegration_AllowsAtOrAboveTier(t *testing.T) {
	for _, tc := range gatedHandlerCases {
		t.Run(tc.name, func(t *testing.T) {
			plat := &gatedFakePlat{name: "OkApp", tier: platform.TierFull}
			res, err := tc.handler(context.Background(), plat, tc.params)
			if err != nil {
				t.Fatalf("transport error: %v", err)
			}
			if res.IsError {
				t.Fatalf("expected success, got IsError: %q", res.Text)
			}
		})
	}
}
