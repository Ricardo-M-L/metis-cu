// Package server hosts the stdio MCP server that exposes the 24
// computer-use tools to any MCP client (metis, Claude Code, Cline, etc.).
//
// The server is a thin adapter: it owns no GUI logic itself. Each tool's
// implementation is in pkg/tools/*.go and dispatches to a Platform
// interface (pkg/platform) that has per-OS realizations.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
	"github.com/Ricardo-M-L/metis-cu/pkg/tools"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// Version is wired in from main at build time so tests can construct
// servers without depending on the main package.
var Version = "0.0.0-test"

// Options controls runtime behavior. Kept separate from MCP-protocol
// internals so future flags (debug log path, sandbox mode, frontmost-app
// tier overrides) don't leak into the wire layer.
type Options struct {
	// Debug toggles RPC frame logging to ~/.metis-cu/debug.log. Useful
	// when chasing "tool reported wrong coords" type issues. Currently
	// a no-op pending sprint-3 instrumentation.
	Debug bool
}

// Run starts the MCP server on stdio and blocks until the client
// disconnects (which on stdio means EOF on stdin). The platform
// implementation is selected at compile time via build tags in
// pkg/platform/platform_<goos>.go.
//
// Defers plat.Close() so resources held by the platform layer
// (kbinani/screenshot's X11 display handle on Linux, robotgo's
// CGEvent source on macOS, clipboard library run loops) get
// released cleanly on EOF. Without this, restarting the server in
// the same process — common in tests and embedded use — could
// reuse stale handles and produce confusing "no display" errors.
//
// If the user opted into [failsafe] enabled = true, Run also starts
// the corner-exit watchdog goroutine; cancelled on EOF so it doesn't
// outlive the parent process.
func Run(opts Options) error {
	srv, reg, err := build(opts)
	if err != nil {
		return err
	}
	defer func() { _ = reg.Platform().Close() }()
	cfg, _ := LoadConfig()
	stopFailsafe := startFailsafeWatchdog(reg.Platform(), failsafeConfig{
		enabled:  cfg.Failsafe.Enabled,
		pollMs:   cfg.Failsafe.PollMs,
		holdMs:   cfg.Failsafe.HoldMs,
		cornerPx: cfg.Failsafe.CornerPx,
	})
	defer stopFailsafe()
	return mcpserver.ServeStdio(srv)
}

// build constructs the *MCPServer with all 25 tools registered and a
// matching Registry. Splitting it from Run lets tests spin up an
// in-process client against the same server without touching stdio.
func build(opts Options) (*mcpserver.MCPServer, *tools.Registry, error) {
	plat, err := platform.New()
	if err != nil {
		return nil, nil, fmt.Errorf("platform init: %w", err)
	}

	reg := tools.NewRegistry(plat)

	// Honor user overrides from ~/.metis-cu/config.toml (if present).
	// LoadConfig falls back to defaults on any read/parse failure so a
	// boot never fails on a hostile config — the MCP server prefers a
	// working default to a startup error the user can't see.
	cfg, _ := LoadConfig()
	reg.SetScreenshotLimits(cfg.Screenshot.MaxWidth, cfg.Screenshot.MaxHeight)
	reg.SetScreenshotFormat(cfg.Screenshot.Format, cfg.Screenshot.Quality)
	reg.SetTypePasteThreshold(cfg.Keyboard.TypePasteThreshold)
	reg.SetLimits(
		cfg.Keyboard.HoldMaxMs,
		cfg.Limits.ClipboardMaxBytes,
		cfg.Limits.BatchMaxSteps,
		cfg.Limits.ZoomMaxFactor,
		cfg.Limits.ZoomMaxOutputPixels,
	)
	platform.SetMouseSmooth(cfg.Mouse.SmoothLow, cfg.Mouse.SmoothHigh)
	reg.SetMouseSettleMs(cfg.Mouse.SettleMs)
	platform.SetFrontmostProbeTimeout(time.Duration(cfg.Gate.FrontmostTimeoutMs) * time.Millisecond)
	platform.SetFrontmostCacheTTL(time.Duration(cfg.Gate.FrontmostCacheTtlMs) * time.Millisecond)
	// Host-terminal tier override. Env wins over config so metis can
	// flip this on per-spawn without writing to the user's TOML; an
	// empty / unrecognised tier disables the override entirely.
	hostTier := os.Getenv("METIS_CU_HOST_TERMINAL_TIER")
	if hostTier == "" {
		hostTier = cfg.Gate.HostTerminalTier
	}
	platform.SetHostTerminalOverride(platform.AccessTier(hostTier))

	srv := mcpserver.NewMCPServer(
		"metis-cu",
		Version,
		mcpserver.WithToolCapabilities(false),
		mcpserver.WithPromptCapabilities(false),
	)
	registerPrompts(srv)

	for _, spec := range reg.Specs() {
		spec := spec // capture for closure
		schemaBytes, err := json.Marshal(spec.Schema)
		if err != nil {
			return nil, nil, fmt.Errorf("marshal schema for %s: %w", spec.Name, err)
		}
		// Construct mcp.Tool directly (instead of mcp.NewTool) so we
		// can supply RawInputSchema without conflicting with the
		// default object InputSchema that NewTool would otherwise
		// install. See mcp/tools.go:Tool.MarshalJSON for the conflict
		// check.
		tool := mcp.Tool{
			Name:           spec.Name,
			Description:    spec.Description,
			RawInputSchema: schemaBytes,
		}
		srv.AddTool(tool, makeHandler(reg, spec.Name))
	}

	_ = opts.Debug // reserved for future RPC frame logging
	return srv, reg, nil
}

// makeHandler binds one tool name to a closure that pulls arguments
// off the CallToolRequest, dispatches via reg.Call, and shapes the
// tools.Result into mcp-go's CallToolResult (text-only or text+image).
func makeHandler(reg *tools.Registry, name string) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		res, err := reg.Call(ctx, name, args)
		if err != nil {
			// Transport-level error — wire as an isError result so the
			// LLM sees and self-corrects rather than the request hard-
			// failing at the protocol layer.
			return mcp.NewToolResultError(err.Error()), nil
		}
		if res == nil {
			return mcp.NewToolResultError("tool returned nil result"), nil
		}
		return resultToMCP(res), nil
	}
}

// resultToMCP packs a tools.Result into mcp-go's CallToolResult. Image
// results carry both the descriptive Text and the base64 Image (so MCP
// clients that ignore image blocks still see "captured 1280x800 PNG"
// or similar). Errors set IsError=true on the result.
func resultToMCP(r *tools.Result) *mcp.CallToolResult {
	if r.IsError {
		return mcp.NewToolResultError(r.Text)
	}
	if r.Image != "" {
		mime := r.MIMEType
		if mime == "" {
			mime = "image/png"
		}
		return mcp.NewToolResultImage(r.Text, r.Image, mime)
	}
	return mcp.NewToolResultText(r.Text)
}
