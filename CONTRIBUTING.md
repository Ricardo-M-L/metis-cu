# Contributing to metis-cu

Thanks for the interest. This file captures the conventions the maintainer
(and any contributors) follow.

A Chinese version lives at [CONTRIBUTING.zh-CN.md](CONTRIBUTING.zh-CN.md).

## Project layout

```
main.go                   stdio MCP server entry
pkg/server/               protocol adapter (JSON-RPC framing, tool dispatch)
pkg/tools/                24 tool registrations + tier-aware routing
pkg/platform/             OS abstraction
  platform.go             Platform interface + tier enum (read/click/full)
  platform_darwin.go      macOS — CGEvent / screencapture / NSPasteboard
  platform_other.go       non-darwin stubs (Sprint 4 fills Linux + Windows)
docs/api-spec.md          full schema reference, mirroring Anthropic
```

Anything under `pkg/platform` is the OS-specific work. Adding a new platform
means dropping a `platform_<goos>.go` file; the interface in `platform.go` is
the contract.

## API stability

The 24 tool names and parameter shapes deliberately mirror Anthropic's
`mcp__computer-use__*` namespace. **Do not rename or reshape them.** If
Anthropic ships a new tool, mirror it; if we need a tool Anthropic doesn't
have, namespace it under `metis_*` instead of polluting the shared
namespace.

## Building & testing

```sh
go build ./...                          # local binary at ./metis-cu
go test -count=1 -timeout 60s ./...     # unit suite
go vet ./...
```

cgo is required (robotgo + screenshot). For cross-platform release builds
use `make dist-darwin` etc. — pure-go cross-compile won't work.

Pre-commit checklist:

1. `go test ./...` is green
2. `go vet ./...` is clean
3. `gofmt -l .` returns nothing
4. The change has either a test or a clear "tested manually because <reason>" note
5. If you added a tool name, it appears in `pkg/tools/tools.go` allToolNames AND
   in `docs/api-spec.md` — keep them in sync

## Style

- **Comments explain WHY, not WHAT.** A well-named function does not need a
  paraphrase of its body.
- **No multi-paragraph docstrings.** One short line is the cap.
- **Platform code is platform_<goos>.go** — never use `runtime.GOOS` switches
  inside a single file; let build tags pick the implementation.
- **Tier enforcement is centralized** in `pkg/tools/gate.go`. Don't sprinkle
  frontmost-app checks across individual tool handlers.
- **No emoji unless the user asks.**

## Reporting security issues

Please don't open public issues for security findings. Use the private
advisory form linked from [SECURITY.md](SECURITY.md).
