// Package server hosts the stdio MCP server that exposes the 24
// computer-use tools to any MCP client (metis, Claude Code, Cline, etc.).
//
// The server is a thin adapter: it owns no GUI logic itself. Each tool's
// implementation is in pkg/tools/*.go and dispatches to a Platform
// interface (pkg/platform) that has per-OS realizations.
package server

import (
	"context"
	"fmt"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
	"github.com/Ricardo-M-L/metis-cu/pkg/tools"
)

// Options controls runtime behavior. Kept separate from MCP-protocol
// internals so future flags (debug log path, sandbox mode, frontmost-app
// tier overrides) don't leak into the wire layer.
type Options struct {
	// Debug toggles RPC frame logging to ~/.metis-cu/debug.log. Useful
	// when chasing "tool reported wrong coords" type issues — the log
	// captures both the caller-supplied params and the dispatched
	// platform call.
	Debug bool
}

// Run starts the MCP server on stdio and blocks until the client
// disconnects (which on stdio means EOF on stdin). The platform
// implementation is selected at compile time via build tags in
// pkg/platform/platform_<goos>.go.
func Run(opts Options) error {
	plat, err := platform.New()
	if err != nil {
		return fmt.Errorf("platform init: %w", err)
	}
	defer plat.Close()

	reg := tools.NewRegistry(plat)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// TODO sprint-1 placeholder: real implementation will use
	// github.com/mark3labs/mcp-go to handle the JSON-RPC framing. For
	// now we just wire the registry through and exit so `go build`
	// succeeds end-to-end.
	_ = ctx
	_ = reg
	_ = opts.Debug
	return fmt.Errorf("MCP stdio loop not implemented yet — see TODO in pkg/server/server.go")
}
