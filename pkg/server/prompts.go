package server

// MCP prompts/list + prompts/get capability — Tier-1 borrow from
// self-operating-computer's curated SYSTEM_PROMPT_* family
// (`operate/models/prompts.py:11-207`) generalised so any MCP client
// can adopt them. metis-cu is "model-agnostic, no loop" — but
// shipping a known-good starting prompt as an MCP capability lets
// clients (Claude Desktop, Cline, Cursor, etc.) discover one through
// the protocol rather than copy-pasting from a README.
//
// All prompts are static (no PromptArguments). OS-specific bits are
// resolved at GetPrompt time via runtime.GOOS so a Mac model gets
// "cmd+space" while a Windows model gets "win+s".

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// promptSpec is the local shape we register against the MCP server —
// pairs the protocol-facing Prompt with the closure that builds its
// PromptMessage at fetch time. Splitting out the build keeps the spec
// list scannable while letting each prompt do its own OS-aware
// substitution.
type promptSpec struct {
	prompt mcp.Prompt
	build  func(ctx context.Context) string
}

// registerPrompts wires every shipped prompt onto srv. Called from
// build() once at boot, alongside the tool registrations. Returns the
// count so the caller can log it during startup.
func registerPrompts(srv *mcpserver.MCPServer) int {
	specs := allPromptSpecs()
	for _, s := range specs {
		s := s // capture for closure
		srv.AddPrompt(s.prompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			body := s.build(ctx)
			return &mcp.GetPromptResult{
				Description: s.prompt.Description,
				Messages: []mcp.PromptMessage{
					mcp.NewPromptMessage(mcp.RoleUser, mcp.NewTextContent(body)),
				},
			}, nil
		})
	}
	return len(specs)
}

// allPromptSpecs returns the curated set. Adding a new prompt is one
// entry here — keep them tight (a screenful each) so a model that
// loads the prompt isn't immediately distracted by lore.
func allPromptSpecs() []promptSpec {
	return []promptSpec{
		{
			prompt: mcp.Prompt{
				Name:        "computer_use_minimal",
				Description: "Compact system prompt for an MCP client driving metis-cu. Includes the call-screen_size-first convention, OS-aware modifier hints, and the tier-gate caveat.",
			},
			build: buildComputerUseMinimal,
		},
		{
			prompt: mcp.Prompt{
				Name:        "tier_overview",
				Description: "One-paragraph explainer of the read/click/full tier system so the model knows why a click on Safari may be denied. Drop into your system prompt verbatim.",
			},
			build: buildTierOverview,
		},
		{
			prompt: mcp.Prompt{
				Name:        "safe_browse",
				Description: "Narrower starter prompt for read-mostly browsing tasks. Steers the model toward screenshot + cursor_position + scroll over typing/clicking, since browsers are TierRead by default.",
			},
			build: buildSafeBrowse,
		},
	}
}

// osHints returns OS-specific tuples so per-prompt templates don't
// each re-invent the cmd/ctrl/win plumbing.
type osHints struct {
	primary    string // "cmd" on darwin, "ctrl" elsewhere
	app        string // "command+space" on darwin, "win" on windows, "super" on linux
	openFiles  string // human-friendly app launcher hint
	closeAlert string
}

func currentOSHints() osHints {
	switch runtime.GOOS {
	case "darwin":
		return osHints{
			primary:    "cmd",
			app:        "cmd+space",
			openFiles:  "Spotlight (cmd+space) or open_application",
			closeAlert: "esc or cmd+. ",
		}
	case "windows":
		return osHints{
			primary:    "ctrl",
			app:        "win or win+s",
			openFiles:  "Start menu (Win key) or open_application",
			closeAlert: "esc or alt+f4",
		}
	default: // linux + other
		return osHints{
			primary:    "ctrl",
			app:        "super or alt+f2",
			openFiles:  "your launcher (super) or open_application",
			closeAlert: "esc",
		}
	}
}

func buildComputerUseMinimal(_ context.Context) string {
	h := currentOSHints()
	return strings.TrimSpace(fmt.Sprintf(`You are driving a host computer through the metis-cu MCP server. Tools are split into vision (screenshot, screen_size, cursor_position, zoom), mouse, keyboard, clipboard, and application categories.

Operating principles:

1. **Call %[1]s first.** Never guess display dimensions — request the canvas explicitly so coordinates land where you intend.
2. **Look before you act.** Take a %[2]s, reason about what you see, then call exactly one mouse / keyboard tool. Don't chain blind actions.
3. **Coordinates are LOGICAL pixels of the active display.** If the screenshot was downsampled (image dim != display_width_px), scale up before clicking.
4. **Modifier shortcut: use %[3]s as the primary modifier on this OS** (the wire form "cmd" is auto-translated to "ctrl" on Linux/Windows for portability).
5. **Tier gate.** Browsers are TierRead (no input), terminals/IDEs are TierClick (clicks only). If the frontmost-app gate denies a call, switch focus to a TierFull app or call request_access.
6. **Open apps via %[4]s.** Don't try to remember absolute file paths.
7. **When stuck:** call cursor_position to verify pointer state, or screen_size to confirm display geometry. If a button doesn't respond, pause and re-screenshot — animations may still be running.
8. **Stop when done.** Return a concise summary text, not another tool call.`,
		"`screen_size`",
		"`screenshot`",
		h.primary,
		h.openFiles,
	))
}

func buildTierOverview(_ context.Context) string {
	return strings.TrimSpace(`metis-cu enforces a per-app TIER gate before every input action:

- **read** — visible only. The model can screenshot but cannot click, type, or scroll. Default for browsers (Safari, Chrome, Firefox, Edge, Arc, Brave).
- **click** — left-click only. No typing or right-click. Default for terminals and IDEs (Terminal, iTerm2, VS Code, Cursor, IntelliJ, GoLand, Xcode, …) so a stray "type" command can't run a destructive shell.
- **full** — everything. Default for everything else.

When a call is denied the result text contains both the app name and the current tier so the model can self-correct. Override per-app via the request_access tool — that pops a NATIVE OS confirmation dialog the user must accept; the MCP client cannot self-grant.`)
}

func buildSafeBrowse(_ context.Context) string {
	h := currentOSHints()
	return strings.TrimSpace(fmt.Sprintf(`You are inspecting web pages through metis-cu. The frontmost app will be a browser, which is TierRead by default — clicks/typing will be denied.

Workflow:
1. screen_size, then screenshot to confirm the active page.
2. To navigate: scroll (read-only on TierRead) or zoom into a region.
3. To follow a link or fill a form: first call request_access with the browser name; the user gets a native confirm dialog. After approval, switch to TierFull tools.
4. Modifier convention: use %s for the primary modifier (auto-translated cross-OS).
5. Hover-only checks: use cursor_position + screenshot rather than risking a click that might be denied.

Don't try to bypass the tier gate. If the user hasn't granted access, surface the deny message verbatim so they can decide.`, h.primary))
}
