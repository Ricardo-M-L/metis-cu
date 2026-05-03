# metis-cu

Computer-use MCP server for [metis](https://github.com/Ricardo-M-L/metis), implemented in Go.

Exposes 24 desktop-control tools (screenshot / mouse / keyboard / clipboard / window) to any MCP client over stdio. **API-compatible with Anthropic's `mcp__computer-use__*` namespace** — same tool names, same parameter shapes, same return semantics — so prompts and traces written for Claude Code's built-in computer-use server work here without rewriting.

CI status: macOS · Ubuntu · Windows all green.

## Why a separate binary

metis itself stays cgo-free for clean cross-compilation (`make build` produces 4 platforms in one shot). All the platform-specific GUI code lives here, in a sub-binary the user opts into:

```
metis (main, cgo-free)
   │  stdio MCP protocol ←→
metis-cu (this repo, cgo-required)
   ├── platform_darwin_*.go   (CGEvent + osascript + screencapture)
   ├── platform_linux_*.go    (XTEST + xdotool + xdg-open)
   ├── platform_windows_*.go  (SendInput + user32 GetForegroundWindow)
   └── platform_other.go      (stub fallback for freebsd / openbsd)
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
- [x] Sprint 2 — macOS implementation (CGEvent + screencapture); all 24 tools wired
- [x] Sprint 3 — tier system (terminal/IDE → click; browser → read; everything else → full); access flow with `~/.metis-cu/granted.json` persistence
- [x] Sprint 4 — Linux X11 / XWayland (xdotool); Windows (user32 + gopsutil)
- [x] Sprint 5 — metis integration via `/cu enable` slash command

## Tier-based safety gate

Each call goes through `EnforceTier` first: the frontmost app's tier (Read < Click < Full) must match what the tool requires. Defaults:

* **Read** (screenshot + read_clipboard only): browsers — Safari, Chrome, Firefox, Edge, Brave, Arc
* **Click** (+ left_click, mouse_move, scroll): terminals + IDEs — iTerm2, VS Code, Cursor, GoLand, IntelliJ, …
* **Full** (everything): every other frontmost app

The LLM can request a wider tier on a specific app via `request_access`; approvals persist to `~/.metis-cu/granted.json`.

## Install

```bash
git clone https://github.com/Ricardo-M-L/metis-cu && cd metis-cu
make install      # writes ~/go/bin/metis-cu + ~/.local/bin/metis-cu
```

Then register with metis (one-liner in chat — no manual TOML edit):

```text
> /cu enable
cu: enabled — computer-use (24 tools); binary=/Users/.../go/bin/metis-cu
```

Or by hand if you'd rather see the TOML:

```toml
# ~/.metis/mcp.toml
[[servers]]
name = "computer-use"
command = "metis-cu"
```

Tools appear under `mcp__computer-use__screenshot`, `mcp__computer-use__left_click`, etc.

## Platforms

CI matrix (ubuntu-latest + macos-latest + windows-latest) builds & tests every commit. Each row reflects the production-quality state of that platform's input + screenshot + clipboard pipeline:

| Platform | Vision | Mouse / Keyboard | Clipboard | Frontmost App | Launcher |
|---|---|---|---|---|---|
| **macOS** (darwin) | kbinani/screenshot (CGImage) | robotgo (CGEvent) | golang.design/x/clipboard | osascript (System Events) | `open -a` |
| **Linux** (X11/XWayland) | kbinani/screenshot (XGetImage) | robotgo (XTEST) | golang.design/x/clipboard (XCB) | xdotool getactivewindow | xdg-open / gtk-launch |
| **Windows** | kbinani/screenshot (GDI BitBlt) | robotgo (SendInput) | golang.design/x/clipboard (Win32) | user32!GetForegroundWindow + gopsutil | `cmd /c start` |
| freebsd / openbsd | stub (ErrNotImplemented) | — | — | — | — |

Native Wayland (without XWayland) is the only known gap — most distros still ship XWayland by default.

## License

Apache-2.0
