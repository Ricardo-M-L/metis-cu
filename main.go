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
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
	"github.com/Ricardo-M-L/metis-cu/pkg/server"
)

const Version = "0.0.3"

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr, server.Run, platform.RequestPermission))
}

// runCLI separates metadata and an explicit, user-initiated TCC request from
// MCP startup. The request callback also lets tests avoid showing OS prompts.
func runCLI(args []string, stdout, stderr io.Writer, serve func(server.Options) error, requestPermission func(string) error) int {
	server.Version = Version

	flags := flag.NewFlagSet("metis-cu", flag.ContinueOnError)
	flags.SetOutput(stderr)
	debug := flags.Bool("debug", false, "log MCP RPC frames to ~/.metis-cu/debug.log")
	version := flags.Bool("version", false, "print version and exit")
	describeFlag := flags.Bool("describe", false, "print the side-effect-free managed helper descriptor")
	permission := flags.String("request-permission", "", "request macOS accessibility or screen-recording permission")
	jsonOutput := flags.Bool("json", false, "emit descriptor output as JSON")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "metis-cu: unexpected positional arguments")
		return 2
	}
	permissionRequested := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "request-permission" {
			permissionRequested = true
		}
	})
	if permissionRequested {
		if *version || *describeFlag || *debug {
			fmt.Fprintln(stderr, "metis-cu: --request-permission cannot be combined with --version, --describe, or --debug")
			return 2
		}
		if *permission != "accessibility" && *permission != "screen-recording" {
			fmt.Fprintln(stderr, "metis-cu: --request-permission must be accessibility or screen-recording")
			return 2
		}
		if err := requestPermission(*permission); err != nil {
			fmt.Fprintln(stderr, "metis-cu: request permission:", err)
			return 1
		}
		current := server.Describe()
		if *jsonOutput {
			if err := json.NewEncoder(stdout).Encode(current); err != nil {
				fmt.Fprintln(stderr, "metis-cu: write descriptor:", err)
				return 1
			}
			return 0
		}
		key := "accessibility"
		if *permission == "screen-recording" {
			key = "screenRecording"
		}
		if _, err := fmt.Fprintf(stdout, "%s: %s\n", *permission, current.Permissions[key]); err != nil {
			fmt.Fprintln(stderr, "metis-cu: write permission status:", err)
			return 1
		}
		return 0
	}

	if *version {
		if _, err := fmt.Fprintln(stdout, "metis-cu", Version); err != nil {
			fmt.Fprintln(stderr, "metis-cu: write version:", err)
			return 1
		}
		return 0
	}
	if *describeFlag {
		if err := json.NewEncoder(stdout).Encode(server.Describe()); err != nil {
			fmt.Fprintln(stderr, "metis-cu: write descriptor:", err)
			return 1
		}
		return 0
	}
	if *jsonOutput {
		fmt.Fprintln(stderr, "metis-cu: --json is only valid with --describe or --request-permission")
		return 2
	}

	if err := serve(server.Options{Debug: *debug}); err != nil {
		fmt.Fprintln(stderr, "metis-cu:", err)
		return 1
	}
	return 0
}
