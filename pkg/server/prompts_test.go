package server

import (
	"runtime"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// TestE2E_ListsPrompts: the in-process client sees the curated prompt
// catalogue over the wire — exercises AddPrompt + WithPromptCapabilities
// + the MCP prompts/list path.
func TestE2E_ListsPrompts(t *testing.T) {
	c, ctx := startInProcessClient(t)
	resp, err := c.ListPrompts(ctx, mcp.ListPromptsRequest{})
	if err != nil {
		t.Fatalf("ListPrompts: %v", err)
	}
	if len(resp.Prompts) != len(allPromptSpecs()) {
		t.Fatalf("prompts count = %d, want %d", len(resp.Prompts), len(allPromptSpecs()))
	}
	seen := map[string]bool{}
	for _, p := range resp.Prompts {
		seen[p.Name] = true
		if p.Description == "" {
			t.Errorf("prompt %q has no description", p.Name)
		}
	}
	mustHave := []string{"computer_use_minimal", "tier_overview", "safe_browse"}
	for _, n := range mustHave {
		if !seen[n] {
			t.Errorf("missing prompt: %s", n)
		}
	}
}

// TestE2E_GetPrompt_OSAware: prompts/get on computer_use_minimal
// returns a body whose "primary modifier" hint matches the runtime
// OS — locks in the runtime.GOOS substitution path so a future
// refactor doesn't accidentally hardcode "cmd" everywhere.
func TestE2E_GetPrompt_OSAware(t *testing.T) {
	c, ctx := startInProcessClient(t)
	req := mcp.GetPromptRequest{}
	req.Params.Name = "computer_use_minimal"
	resp, err := c.GetPrompt(ctx, req)
	if err != nil {
		t.Fatalf("GetPrompt: %v", err)
	}
	if len(resp.Messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(resp.Messages))
	}
	tc, ok := resp.Messages[0].Content.(mcp.TextContent)
	if !ok {
		t.Fatalf("content type = %T, want mcp.TextContent", resp.Messages[0].Content)
	}
	body := tc.Text
	wantPrimary := "cmd"
	if runtime.GOOS != "darwin" {
		wantPrimary = "ctrl"
	}
	if !strings.Contains(body, wantPrimary) {
		t.Errorf("body missing primary modifier %q for GOOS=%s; body:\n%s", wantPrimary, runtime.GOOS, body)
	}
	// Always-present hints — nothing OS-specific about these. Match
	// the actual capitalization in the prompt body ("Tier gate", not
	// lowercase "tier").
	for _, must := range []string{"screen_size", "screenshot", "Tier"} {
		if !strings.Contains(body, must) {
			t.Errorf("body missing keyword %q; body:\n%s", must, body)
		}
	}
}

// TestE2E_GetPrompt_TierOverview is non-OS-specific so it doubles as
// a sanity check that prompts without GOOS branches still work.
func TestE2E_GetPrompt_TierOverview(t *testing.T) {
	c, ctx := startInProcessClient(t)
	req := mcp.GetPromptRequest{}
	req.Params.Name = "tier_overview"
	resp, err := c.GetPrompt(ctx, req)
	if err != nil {
		t.Fatalf("GetPrompt: %v", err)
	}
	tc := resp.Messages[0].Content.(mcp.TextContent)
	for _, must := range []string{"read", "click", "full", "request_access"} {
		if !strings.Contains(tc.Text, must) {
			t.Errorf("tier_overview body missing %q; body:\n%s", must, tc.Text)
		}
	}
}
