# metis-cu

Computer-use MCP server for [metis](https://github.com/Ricardo-M-L/metis), implemented in Go.

Exposes 24 desktop-control tools (screenshot / mouse / keyboard / clipboard / window) to any MCP client over stdio. **API-compatible with Anthropic's `mcp__computer-use__*` namespace** — same tool names, same parameter shapes, same return semantics — so prompts and traces written for Claude Code's built-in computer-use server work here without rewriting.

## Why a separate binary

metis itself stays cgo-free for clean cross-compilation (`make build` produces 4 platforms in one shot). All the platform-specific GUI code lives here, in a sub-binary the user opts into:

```
metis (main, cgo-free)
   │  stdio MCP protocol ←→
metis-cu (this repo)
   ├── platform_darwin.go   (CoreGraphics + CGEvent + Accessibility)
   ├── platform_linux.go    (X11 / xdotool fallback)
   └── platform_windows.go  (SendInput / GDI)
```

## Tool surface (24, mirroring Anthropic spec)

| Category | Tools |
|---|---|
| Vision | `screenshot`, `cursor_position`, `switch_display`, `zoom` |
| Mouse | `mouse_move`, `left_click`, `right_click`, `middle_click`, `double_click`, `triple_click`, `left_click_drag`, `left_mouse_down`, `left_mouse_up`, `scroll` |
| Keyboard | `key`, `hold_key`, `type` |
| Clipboard | `read_clipboard`, `write_clipboard` |
| Application | `open_application`, `list_granted_applications`, `request_access` |
| Control | `wait`, `computer_batch` |

Full schemas in [docs/api-spec.md](docs/api-spec.md).

## Roadmap

- [x] Sprint 1 — repo skeleton, stdio MCP server, all 24 tool stubs returning `not implemented`
- [ ] Sprint 2 — macOS implementation (CGEvent + screencapture); 8 core tools wired
- [ ] Sprint 3 — tier system (terminal/IDE → click; browser → read; everything else → full); access flow
- [ ] Sprint 4 — Linux X11 + Wayland; Windows SendInput
- [ ] Sprint 5 — metis integration (`/cu enable` auto-loads via mcp.toml)

## Install

```bash
go install github.com/Ricardo-M-L/metis-cu@latest
```

Then in `~/.metis/mcp.toml`:

```toml
[[servers]]
name = "computer-use"
command = "metis-cu"
```

Restart `metis chat`; tools appear under `mcp__computer-use__*`.

## License

Apache-2.0
