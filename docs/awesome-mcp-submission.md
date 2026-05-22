# Awesome MCP submission

Drafts for getting metis-cu into upstream MCP discovery channels. Each entry
is sized to the host repo's existing entry style — copy/paste-ready.

## Target 1: punkpeye/awesome-mcp-servers

The most-watched community list (~30k stars). Section: "Browser Automation"
sits next to it but the right home is "Operating System" or a new
"Desktop Automation" section if the maintainer prefers.

PR title:

```
Add metis-cu (open-source computer-use MCP server, prompt-compatible with Claude Code)
```

PR body (Markdown):

```markdown
## What

[metis-cu](https://github.com/Ricardo-M-L/metis-cu) — a Go implementation of
the `mcp__computer-use__*` tool namespace defined by Anthropic. Drops into
any MCP client (Claude Code, Codex, Continue, Cline, metis, …) over stdio
and exposes 24 desktop-control tools (screenshot / mouse / keyboard /
clipboard / window).

## Why this fills a gap

The `mcp__computer-use__*` namespace is **well-defined by Anthropic** but
existing implementations are closed:

- Claude Code: in-process built-in (not extractable, can't share with
  other clients)
- Codex: plugin restricted to OpenAI's macOS desktop app, not the CLI

metis-cu is the first **open-source, prompt-compatible, cross-platform**
(macOS / Linux / Windows) implementation. Prompts and traces written for
Claude Code's built-in computer-use server work here without rewriting,
making it the natural choice for any MCP client that wants Anthropic-spec
desktop control.

## Suggested entry

> **[metis-cu](https://github.com/Ricardo-M-L/metis-cu)** — Cross-platform
> computer-use MCP server in Go. Drop-in for Anthropic's
> `mcp__computer-use__*` namespace; works with any MCP client. macOS /
> Linux / Windows.
```

## Target 2: modelcontextprotocol/servers (official)

The official MCP servers list maintained by Anthropic. Likely an issue
rather than a PR — this list curates first-party + closely-vetted
community servers, so pitch it via "I'd like to add" issue and let
maintainers decide on inclusion form.

Issue title:

```
Add metis-cu to community servers list (open-source `computer-use` namespace impl)
```

Issue body:

```markdown
Hi MCP team,

I'd like to suggest adding [metis-cu](https://github.com/Ricardo-M-L/metis-cu)
to the community servers list. It's the first open-source implementation of
the `mcp__computer-use__*` tool namespace defined in Anthropic's spec, and
it's prompt-compatible with Claude Code's built-in computer-use server (same
24 tool names, same parameter shapes, same return semantics).

Why a community impl is useful:
- Other MCP clients (Codex CLI, Continue, Cline, …) can't currently access
  Claude Code's in-process built-in. metis-cu lets any of them adopt the
  canonical namespace without each project building their own.
- Cross-platform from day one: macOS / Linux (X11+XWayland) / Windows, all
  green on CI.
- Tier-based safety gate built in (browsers default to Read; terminals and
  IDEs default to Click; everything else Full) with explicit
  `request_access` flow.

Apache-2.0 licensed. Happy to adapt the entry style to whatever fits the
list — let me know if you'd prefer a PR with a specific format.
```

## Target 3: Reddit / HN / X announcement (optional)

Once both lists accept, a single short post:

> **metis-cu** — open-source, prompt-compatible computer-use MCP server.
> Drop into Claude Code, Codex CLI, Continue, or any MCP client. Same
> `mcp__computer-use__*` tool names as Anthropic's built-in. macOS / Linux
> / Windows. Apache-2.0.
> https://github.com/Ricardo-M-L/metis-cu

## When to actually submit

Pre-flight checklist before opening either PR/issue:

- [ ] LICENSE is full Apache 2.0 (200+ lines, not the 17-line stub) — DONE
- [ ] README has badges + `go install` one-liner + multi-client config
      examples — DONE
- [ ] CI is green on macOS / Linux / Windows
- [ ] At least one tagged release (`v0.1.0` minimum) — gives the entry a
      stable thing to link to and signals "ready to use"
- [ ] A short demo (GIF or asciinema) embedded in README — biggest single
      conversion lever for a desktop-automation tool
- [ ] Issue tracker is open and you're prepared to triage incoming bugs

The first three are the hard prerequisites; the demo + release + issue
triage capacity are what turn a passing entry into adoption.
