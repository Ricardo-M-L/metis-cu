# Changelog

All notable changes to metis-cu are recorded here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); the project
follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html) once
it leaves 0.x.

## [Unreleased]

## [0.0.3] - 2026-09-30

### Added

- Expose the protocol-1 helper descriptor and conservative tool activity as
  the MCP `metis-cu://status` resource.

## [0.0.2] - 2026-09-30

### Added

- Report the current macOS Accessibility and Screen Recording permission state
  in the managed helper descriptor without requesting access.
- Add explicit `--request-permission accessibility|screen-recording` commands
  for a user-initiated macOS permission request; `--json` returns the updated
  managed helper descriptor.

### Added — Sprint 1 scaffold (2026-05-01)

- Repo skeleton with `main.go`, `pkg/server`, `pkg/tools`,
  `pkg/platform/{darwin,other}.go`.
- 24 tool names registered, mirroring Anthropic's `mcp__computer-use__*`
  namespace exactly (vision 4 / mouse 10 / keyboard 3 / clipboard 2 /
  application 3 / control 2).
- Documentation: README, CONTRIBUTING (en+zh), SECURITY,
  CODE_OF_CONDUCT (en+zh), `docs/api-spec.md`.
- CI workflow scaffold (Go build + vet + test on macOS + Ubuntu).

### TODO — Sprint 2

- macOS implementation: screencapture, CGEvent posting, NSPasteboard.
- Tier system (`gate.go`) wired into every mouse/keyboard tool.
- mcp-go-based stdio loop replacing the placeholder in `pkg/server`.
