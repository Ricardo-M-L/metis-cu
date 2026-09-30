# metis-cu API spec

24 tools, mirroring Anthropic's `mcp__computer-use__*` namespace. **Wire format and parameter shapes match Anthropic's spec verbatim** — if you have a prompt or trace that worked against Claude Code's built-in computer-use, it works here.

Reference: [Anthropic computer-use tool docs](https://platform.claude.com/docs/en/agents-and-tools/tool-use/computer-use-tool)

## Vision (4)

| Tool | Params | Returns |
|---|---|---|
| `screenshot` | `{ display?: int }` | image content block (PNG, base64) |
| `cursor_position` | `{}` | `{ x: int, y: int }` |
| `switch_display` | `{ display: int }` | `{ ok: true }` |
| `zoom` | `{ region: {x,y,w,h}, factor: float }` | image content block |

## Mouse (10)

| Tool | Params | Notes |
|---|---|---|
| `mouse_move` | `{ x, y }` | Absolute, top-left origin, logical pixels |
| `left_click` | `{ x, y, modifiers?: [str] }` | Modifiers: `cmd`, `ctrl`, `alt`, `shift` |
| `right_click` | `{ x, y }` | Tier "click" apps reject this |
| `middle_click` | `{ x, y }` | |
| `double_click` | `{ x, y }` | |
| `triple_click` | `{ x, y }` | |
| `left_click_drag` | `{ from: {x,y}, to: {x,y} }` | |
| `left_mouse_down` | `{ x, y }` | Pair with `left_mouse_up` for custom drags |
| `left_mouse_up` | `{ x, y }` | |
| `scroll` | `{ x, y, dx, dy }` | Wheel ticks, +y = down |

## Keyboard (3)

| Tool | Params | Notes |
|---|---|---|
| `key` | `{ combo: str }` | "cmd+a", "esc", "F11" |
| `hold_key` | `{ combo: str, ms: int }` | |
| `type` | `{ text: str }` | UTF-8; tier "click" apps reject this |

## Clipboard (2)

| Tool | Params | Returns |
|---|---|---|
| `read_clipboard` | `{}` | `{ text: str }` |
| `write_clipboard` | `{ text: str }` | `{ ok: true }` |

## Application (3)

| Tool | Params | Notes |
|---|---|---|
| `open_application` | `{ name: str }` | "Safari", "Notes". Read-tier op |
| `list_granted_applications` | `{}` | `[{ name, tier }]` |
| `request_access` | `{ apps: [str] }` | Returns `{ name: tier }` after user approval |

## Control (2)

| Tool | Params | Notes |
|---|---|---|
| `wait` | `{ ms: int }` | Server-side sleep — cheaper than client-side polling |
| `computer_batch` | `{ steps: [{tool, params}] }` | Atomic-ish sequence; aborts on first error |

## Tier system

Mirrors Claude Code's frontmost-app gating:

- **read**: visible in screenshots, but `left_click`, `type` rejected. Browsers fall here — use a chrome-mcp for navigation.
- **click**: visible + `left_click` allowed; everything else (typing, right-click, modifiers, drag) rejected. Terminals + IDEs fall here.
- **full**: no restrictions. Most native apps.

The tier is decided by `Platform.FrontmostApp()` at the start of every tool call. `request_access` returns the tier the user approved.

## Managed status resource

MCP clients can list and read `metis-cu://status` (`application/json`). It
contains the same protocol-1 descriptor fields as `metis-cu --describe`, plus
`lifecycle.state`. The state is `running` while a registered MCP tool handler
is executing and `idle` when none is executing. It does not describe the host
turn, ownership of OS input, or activity outside this helper process.
