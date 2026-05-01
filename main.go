// metis-cu — computer-use MCP server for metis.
//
// Speaks the standard MCP protocol over stdio, exposing 24 desktop-control
// tools (screenshot / mouse / keyboard / clipboard / window) under the
// `mcp__computer-use__*` namespace, deliberately mirroring Anthropic's
// Claude Code built-in computer-use server. Same tool names, same parameter
// shapes — so prompts, traces, and eval datasets written for Claude Code
// work here without translation.
//
// Usage (stdio, the standard MCP transport):
//
//	metis-cu                       # plain stdio
//	metis-cu --debug               # log RPC frames to ~/.metis-cu/debug.log
//
// Wire it into metis via ~/.metis/mcp.toml:
//
//	[[servers]]
//	name = "computer-use"
//	command = "metis-cu"
//
// Then in metis chat the tools appear under their full `mcp__computer-use__*`
// names — Anthropic-compatible.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Ricardo-M-L/metis-cu/pkg/server"
)

const Version = "0.0.1-dev"

func main() {
	debug := flag.Bool("debug", false, "log MCP RPC frames to ~/.metis-cu/debug.log")
	version := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *version {
		fmt.Println("metis-cu", Version)
		return
	}

	if err := server.Run(server.Options{Debug: *debug}); err != nil {
		fmt.Fprintln(os.Stderr, "metis-cu:", err)
		os.Exit(1)
	}
}
