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
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/Ricardo-M-L/metis-cu/pkg/server"
)

const Version = "0.0.1-dev"

const managedProtocolVersion = 1

type description struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	ProtocolVersion int               `json:"protocolVersion"`
	Platform        string            `json:"platform"`
	Arch            string            `json:"arch"`
	Capabilities    []string          `json:"capabilities"`
	Permissions     map[string]string `json:"permissions"`
}

func printDescription() error {
	return json.NewEncoder(os.Stdout).Encode(description{
		Name:            "metis-cu",
		Version:         Version,
		ProtocolVersion: managedProtocolVersion,
		Platform:        runtime.GOOS,
		Arch:            runtime.GOARCH,
		Capabilities: []string{
			"status",
			"stop",
			"end-turn",
			"serialized-input",
			"input-ownership",
		},
		Permissions: map[string]string{
			"screenRecording": "runtime",
			"accessibility":   "runtime",
		},
	})
}

func main() {
	server.Version = Version

	debug := flag.Bool("debug", false, "log MCP RPC frames to ~/.metis-cu/debug.log")
	version := flag.Bool("version", false, "print version and exit")
	describe := flag.Bool("describe", false, "print the side-effect-free managed helper descriptor")
	jsonOutput := flag.Bool("json", false, "emit descriptor output as JSON")
	flag.Parse()

	if *version {
		fmt.Println("metis-cu", Version)
		return
	}
	if *describe {
		if err := printDescription(); err != nil {
			fmt.Fprintln(os.Stderr, "metis-cu:", err)
			os.Exit(1)
		}
		return
	}
	if *jsonOutput {
		fmt.Fprintln(os.Stderr, "metis-cu: --json is only valid with --describe")
		os.Exit(2)
	}

	if err := server.Run(server.Options{Debug: *debug}); err != nil {
		fmt.Fprintln(os.Stderr, "metis-cu:", err)
		os.Exit(1)
	}
}
