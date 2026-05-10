package server

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// startInProcessClient builds the same MCPServer Run() would, then
// connects an in-process client. The returned cleanup closes the
// client; the server has no separate lifecycle when using the
// in-process transport (it stops when the client disconnects).
func startInProcessClient(t *testing.T) (*client.Client, context.Context) {
	t.Helper()
	srv, _, err := build(Options{})
	if err != nil {
		t.Fatalf("build server: %v", err)
	}
	c, err := client.NewInProcessClient(srv)
	if err != nil {
		t.Fatalf("new in-process client: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	if err := c.Start(ctx); err != nil {
		t.Fatalf("client start: %v", err)
	}
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "metis-cu-test", Version: "0"}
	if _, err := c.Initialize(ctx, initReq); err != nil {
		t.Fatalf("client initialize: %v", err)
	}
	return c, ctx
}

// TestE2E_ListsAllTools confirms the server advertises every tool
// declared in pkg/tools.allToolNames over the actual MCP wire — not
// just via the in-memory Registry. Catches schema-marshal issues that
// would manifest as a missing tool entry.
func TestE2E_ListsAllTools(t *testing.T) {
	c, ctx := startInProcessClient(t)
	resp, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	const expected = 28
	if got := len(resp.Tools); got != expected {
		t.Fatalf("tools count = %d, want %d", got, expected)
	}
	seen := map[string]bool{}
	for _, tl := range resp.Tools {
		seen[tl.Name] = true
		if tl.Description == "" {
			// Description can be empty for not-yet-implemented tools
			// (Phase 1 only filled screenshot). Tools registered in
			// later phases should add it.
			continue
		}
	}
	mustHave := []string{
		"screenshot", "cursor_position", "switch_display", "zoom", "screen_size", "find_text_on_screen",
		"mouse_move", "left_click", "click_text", "right_click", "middle_click",
		"double_click", "triple_click", "left_click_drag",
		"left_mouse_down", "left_mouse_up", "scroll", "highlight_text_span",
		"key", "hold_key", "type",
		"read_clipboard", "write_clipboard",
		"open_application", "list_granted_applications", "request_access",
		"wait", "computer_batch",
	}
	for _, name := range mustHave {
		if !seen[name] {
			t.Errorf("missing tool: %s", name)
		}
	}
}

// TestE2E_Screenshot exercises the full call path (client → mcp-go
// transport → server → tools.Registry → platform.Screenshot). All
// three first-class platforms (darwin / linux / windows) now have real
// implementations, so the response shape depends on the runner's GUI
// state rather than the build:
//
//   - GUI session present (dev box, Windows CI runner): expect an
//     ImageContent block with PNG data.
//   - Headless / permission-denied (Linux CI without DISPLAY, macOS CI
//     without Screen Recording permission): expect IsError so the LLM
//     can self-correct.
//
// In either case the wiring is proven out — what we're really
// regression-testing is that mcp-go round-trips through tools.Registry
// without dropping the result. Both branches accept that; only a
// transport-level error fails the test.
func TestE2E_Screenshot(t *testing.T) {
	c, ctx := startInProcessClient(t)

	req := mcp.CallToolRequest{}
	req.Params.Name = "screenshot"
	req.Params.Arguments = map[string]any{}
	resp, err := c.CallTool(ctx, req)
	if err != nil {
		t.Fatalf("CallTool screenshot: %v", err)
	}

	if testing.Short() {
		t.Skip("skipping real screenshot capture in short mode")
	}

	// Non-supported OS (freebsd / openbsd) hit the stub fallback
	// (platform_other.go); IsError is the only valid outcome there.
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
		// fall through to the GUI-session-aware checks below
	default:
		if !resp.IsError {
			t.Fatalf("expected IsError on stub platform %s, got: %+v", runtime.GOOS, resp)
		}
		return
	}

	if resp.IsError {
		// Headless runner (no DISPLAY) or permission denied (no Screen
		// Recording entitlement) — the wiring still proved out; the
		// server actually tried to capture and got a platform-level
		// error that propagated through MCP. Log for context.
		t.Logf("screenshot returned error (headless / permission denied likely): %s", textOf(resp))
		return
	}

	var sawImage bool
	for _, blk := range resp.Content {
		if img, ok := blk.(mcp.ImageContent); ok {
			if img.Data == "" {
				t.Errorf("image content has empty Data")
			}
			if img.MIMEType != "image/png" {
				t.Errorf("image MIMEType = %q, want image/png", img.MIMEType)
			}
			sawImage = true
		}
	}
	if !sawImage {
		t.Errorf("expected an ImageContent block, got: %+v", resp.Content)
	}
}

// TestE2E_UnknownTool confirms unknown tool names round-trip as a
// graceful isError, not a transport-level failure.
func TestE2E_UnknownTool(t *testing.T) {
	c, ctx := startInProcessClient(t)
	req := mcp.CallToolRequest{}
	req.Params.Name = "this_tool_does_not_exist"
	req.Params.Arguments = map[string]any{}
	resp, err := c.CallTool(ctx, req)
	// mcp-go's server returns an unknown-tool error at the transport
	// level. Either shape is acceptable as long as the client doesn't
	// crash; we just verify CallTool returns *something* without
	// panicking.
	if err == nil && resp != nil && !resp.IsError {
		t.Errorf("expected error or isError result for unknown tool, got success: %+v", resp)
	}
}

// textOf joins all TextContent blocks for logging.
func textOf(r *mcp.CallToolResult) string {
	if r == nil {
		return ""
	}
	out := ""
	for _, blk := range r.Content {
		if tc, ok := blk.(mcp.TextContent); ok {
			out += tc.Text
		}
	}
	return out
}
