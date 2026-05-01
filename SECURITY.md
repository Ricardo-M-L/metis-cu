# Security Policy

## Threat model

metis-cu sits between an LLM and the user's desktop. Whatever the model
asks for — clicks, keystrokes, screenshots, app launches — runs against
the live machine. The threat surface is therefore the same as any GUI
automation tool, plus the prompt-injection risk inherent to LLM agents.

The mitigations metis-cu ships:

- **Tier system** — browsers run "read" (no input), terminals/IDEs run
  "click" (left-click only), other apps run "full". Implemented in
  `pkg/tools/gate.go` and consulted before every tool call. Mirrors what
  Claude Code's built-in computer-use does.
- **Explicit access flow** — `request_access` requires the user to
  approve each application. Granted apps are persisted; the model can
  query the list via `list_granted_applications`.
- **No sensitive-data-in-args** — handlers reject obviously suspicious
  payloads (credit-card-like, SSN-like, API-key-like literals in `type`
  arguments) before dispatch.
- **No arbitrary process spawn** — `open_application` resolves names
  through the OS launcher only. There is no shell-eval tool in this
  repo (use the calling agent's existing Bash tool if you need that —
  it has its own permission model).

## Reporting a vulnerability

Please do **not** open a public issue. Use GitHub's private security
advisory form:

  https://github.com/Ricardo-M-L/metis-cu/security/advisories/new

We aim for a response within 7 days, a fix or mitigation within 30 days
for confirmed issues, and coordinated disclosure on the schedule the
reporter prefers.

If you can't or don't want to use GitHub, email the maintainer
(Ricardo-M-L) — find an address via the metis README.

## Out of scope

- Models that misuse granted access (e.g. ask the user for a screenshot
  then exfiltrate it via another tool). That's the calling agent's
  problem, not metis-cu's — we just expose primitives.
- Bugs in the underlying OS APIs (CGEvent, X11, SendInput).
- Any "the user gave the model permission and the model did the thing
  the user asked" scenario.
