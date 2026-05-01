# Dashboard Architecture Research

**Date:** 2026-05-01
**Status:** Direction locked — ready for `/adv-proposal`
**Builds on:** [`2026-04-23-temporal-dashboard-research.md`](2026-04-23-temporal-dashboard-research.md)
  (Temporal API surface + browser→Temporal feasibility — still authoritative)
**Prerequisites:** Phase 4 (session lifecycle, shipped) + Phase 5 (Temporal enablement, shipped)

## Locked Direction

| Decision | Choice | Rationale (one line) |
|---|---|---|
| **Frontend stack** | Datastar + server-rendered HTML + Tailwind | No Node bundler, no JS framework, no JS table lib — matches Go-first single-maintainer reality |
| **Binary delivery** | Single Go binary via `//go:embed` | PocketBase / Caddy / Coder pattern — proven for ops tools |
| **Real-time** | Server polls Temporal (gRPC, 2-5s) → SSE push to browser | All ops dashboards poll; SSE is cleanest server→client push for single-user |
| **Temporal access** | `client.ListWorkflow` + existing search attributes | `AdvProjectId`/`AdvChangeId`/`AdvActiveGate`/`AdvDoomLoopActive` already registered |
| **tmux state** | Control-mode client (`tmux -Loca -C`), parse `%`-notifications | Event-driven, no polling — iTerm2 pattern |
| **v1.1 scope** | Read-only unified table | Validate UX, defer auth complexity, ship faster |
| **Cross-host** | Out of scope for v1.x; pulled in by long-term session-resume goal (see below) | Local-only loopback first; design REST endpoints host-scoped but no federation layer until session resume lands |
| **Conservative fallback** | HTMX (instead of Datastar) | Same architecture; template-level swap if Datastar RC stability bites |

## Long-Term: Session Resume / Open from Dashboard

> Captured 2026-05-01 as a roadmap signal — does **not** change v1.1 architecture. Drives v1.2 and v2 scoping.

The eventual goal: clicking a session row (or "new session" button) in the dashboard opens or resumes that session — including from a browser on another device (phone, laptop, remote host).

### Three approaches, ranked by capability and cost

| Approach | UX | Cost | Where it works |
|---|---|---|---|
| **A. tmux switch-client** | Click "open" → user's already-attached tmux client switches to that session | Trivial (one tmux command) | Local only; requires existing tmux client on the same host as OCA |
| **B. Spawn OS terminal + tmux attach** | Click "open" → dashboard server runs `gnome-terminal -- tmux -Loca attach -t <session>` (or platform equivalent) | Low (per-OS terminal probing) | Local only; requires display server + terminal app installed |
| **C. Web terminal (xterm.js + WebSocket PTY broker)** | Click "open" → terminal pane renders inside the dashboard tab, fully interactive | Medium-high (WebSocket layer, PTY broker, auth, input forwarding) | Anywhere a browser works — including remote/mobile/tailnet |

### Why C is the real long-term answer

A and B are useful local conveniences but cap out at "user is sitting at the same machine." The session-resume goal implies:

- Resume a session from another room (laptop in bedroom → dashboard attaches the session running on the dev box)
- Check in from phone (Tailscale-exposed dashboard, brief read or short interaction)
- Multi-device handoff (close laptop, open phone, keep going)

That entire surface needs C. The reference impls are well-trodden:

| Impl | Stack | Notes |
|---|---|---|
| [Coder workspaces](https://github.com/coder/coder) | Go + xterm.js + WebSocket | Gold-standard open-source reference; production-grade PTY broker (`coder/pty`) |
| [gotty](https://github.com/yudai/gotty) | Go + xterm.js | Minimal; share a single command's TTY over WebSocket. Closer in scope to OCA's needs |
| [Wetty](https://github.com/butlerx/wetty) | Node + xterm.js | Browser SSH terminal; older but proven |
| [ttyd](https://github.com/tsl0922/ttyd) | C + xterm.js | Lightweight; single-binary mindset |

For OCA the cleanest implementation is: spawn `tmux -Loca attach -t <session>` inside a PTY (`creack/pty` or `coder/pty`), pipe stdout to a WebSocket, forward keystrokes back. tmux supports multiple clients on the same session, so attaching from the browser does not disconnect an existing terminal client.

### Architectural implications (forces decisions)

| Decision | Without session resume (current v1.1 lock) | With session resume (long-term) |
|---|---|---|
| Real-time transport | SSE only | SSE for state + **WebSocket for terminal PTY** (both, side by side) |
| Auth model | None (loopback only) | **Required** — token / cookie / Tailscale identity / OAuth |
| Cross-host scope | Deferred | **Pulled forward** — dashboard on phone needs to reach OCA on dev box |
| Network exposure | `127.0.0.1` only | LAN, Tailscale, or `0.0.0.0` with auth |
| Bundle size | ~Datastar 14 KB + Tailwind | + xterm.js (~200 KB gzip) — only loaded on terminal route |
| TLS | Not needed (loopback) | Recommended for any non-loopback bind |

**The good news:** none of these break the v1.1 architecture. Datastar/SSE for the table is orthogonal to xterm.js/WebSocket for the terminal route. They coexist cleanly. The Go HTTP server serves both; routes split by URL.

### Phased rollout

| Phase | Capability | Cost |
|---|---|---|
| v1.1 (current proposal) | Read-only unified table; loopback only | 2-3 weeks |
| v1.2 | Per-row controls (gate approve, retry, cancel) + local "switch session" (Approach A) | 1-2 weeks incremental |
| v1.3 | Local "open in new terminal window" (Approach B) + auth scaffolding (token-based) | 1 week |
| v2.0 | Web terminal (xterm.js + PTY broker) + Tailscale-friendly bind + cross-host federation | 3-4 weeks |

### Open questions for v2.0 design (not v1.1)

- Auth: bearer token in env / shared secret / Tailscale identity / OAuth? Tailscale is the LBP for personal dev tooling
- One OCA per host vs federation: dashboard on host A talks to OCA daemons on hosts B and C, or a federation proxy aggregates?
- Terminal session permissions: who can attach to which session? (Same answer as auth, probably)
- Recording / audit: should every web-attached session be replayable (asciinema-style)? Defer

### What we keep an eye on starting v1.1

To avoid v2 reshuffling, v1.1 implementation should:

- Keep the HTTP server pluggable on bind address (not hardcoded to loopback in code, just defaulted)
- Keep route handlers thin enough that auth middleware can wrap them later without rewrites
- Avoid baking single-host assumptions into the data model (changes/sessions already have project-id; that's host-aware enough)
- Don't ship a session table column that only makes sense locally (e.g. local PID without host context)

## Why Server-Rendered, Not SPA

The dashboard is fundamentally a **filterable table + per-row actions + live status dots** — exactly the category where server-rendered + SSE wins, and exactly the wrong category for an SPA stack.

- HTMX/Datastar's strongest documented use case is "admin dashboards and internal tools"
- Plane.so migrated *away* from Next.js for ops tooling, citing 20-30s reload pain
- Server owns sort/filter/pagination state — table is just `<table>` HTML returned as fragments
- Eliminates an entire dependency category (TanStack Table / AG Grid / Tabulator all require a JS framework host)

## Rejected Alternatives

| Option | Why rejected |
|---|---|
| Fork Temporal Web | No plugin system; custom-columns issue ([temporalio/ui#771](https://github.com/temporalio/ui/issues/771)) still open since 2023; permanent maintenance burden |
| SvelteKit SPA | Full Node pipeline; breaks single-binary aesthetic; heaviest maintenance for one maintainer |
| React + Vite + TanStack | Heaviest ecosystem overhead; Plane.so case-study explicitly migrated away from this category for ops |
| Solid/SolidStart/Qwik | No Go integration story; minimal production ops-dashboard examples; community too small for multi-year maintenance |
| Datastar + TanStack Table island (hybrid) | Reintroduces React + bundler just for the table; defeats the no-SPA win |
| Browser → Temporal direct | Confirmed impossible (no gRPC-Web on 7233; no CORS on 7243) |

## Architecture Sketch

```
OCA Go binary (oca dashboard)
├── HTTP server (chi or stdlib net/http)
├── //go:embed frontend/         (HTML templates + Tailwind CSS + Datastar bundle)
├── Temporal client (cross-project ListWorkflow + per-change queries)
├── tmux control-mode client (event stream → in-memory session cache)
├── REST endpoints (/api/changes, /api/sessions, /api/health) — host-scoped
└── SSE channel (/api/events) — fans out state diffs to browser
                ↑
        Browser + Datastar
        ├── data-on-load=GET /api/changes (initial render)
        └── data-on-sse=patch row elements + signals
```

## Phasing (Proposed)

| Phase | Deliverable |
|---|---|
| 1 | Go HTTP server + `//go:embed` skeleton + Tailwind pipeline + `oca dashboard` CLI command |
| 2 | Temporal client wrapper — cross-project list via search attributes; per-change task fetch |
| 3 | tmux control-mode watcher + in-memory session cache |
| 4 | Unified table view (read-only): cross-project changes + sessions + Temporal/worker health |
| 5 | Filters + sorts (server-side), drill-in detail panel |
| 6 | SSE channel + Datastar wiring for live updates |
| 7 | Polish: density toggle, keyboard nav, status semantics, empty states |

**Estimate:** 2-3 weeks focused. v1.1 scope.

**Deferred to v1.2+:** Per-row controls (approve gate, retry, cancel, attach session), auth model, cross-host aggregation.

## Risks + Mitigations

| Risk | Severity | Mitigation |
|---|---|---|
| Datastar RC.7 instability (no v1 yet) | Medium | Pin CDN version; HTMX swap is template-level if needed |
| Datastar CSP requires `unsafe-eval` | Low (loopback only) | Document; revisit if exposed beyond loopback |
| Stale signals on SSE reconnect | Medium | Manual reconnection handler via `data-on:datastar-fetch` |
| tmux control-mode parser complexity | Low | ~200 LOC Go; well-understood protocol; iTerm2 reference impl |
| Tailwind needs build step | Low | Pre-build, commit, embed |
| Spec divergence across worktrees (per ADV worktree policy) | Low | Dashboard reads Temporal state (shared external store), not in-repo specs |

## Key Sources

- [Datastar SDK Reference](https://data-star.dev/reference/sdks) — official SDK list, Go SDK confirmed
- [Datastar Go SDK](https://github.com/starfederation/datastar-go) — MIT, `PatchElements`/`PatchSignals`/`ExecuteScript` via SSE
- [Datastar production considerations](https://alvarolm.github.io/datastar-resources/docs/considerations.html) — RC stability caveats
- [HTMX vs React 2026 Decision Framework](https://pockit.tools/blog/htmx-vs-react-2026-when-you-dont-need-spa/) — admin dashboards as the strongest HTMX category
- [Plane.so: Next.js → React Router + Vite migration](https://plane.so/blog/why-did-we-migrate-plane-from-nextjs-to-react-router-vite) — SPA overhead case study
- [PocketBase //go:embed pattern](https://github.com/pocketbase/pocketbase/discussions/4810) — reference single-binary impl
- [tmux Control Mode wiki](https://github.com/tmux/tmux/wiki/Control-Mode) — `%`-notification protocol
- [r3labs/sse](https://github.com/r3labs/sse) — Go SSE fan-out (optional; stdlib `net/http` is sufficient)
- [Temporal Go SDK ListWorkflow](https://docs.temporal.io/develop/go/platform/observability) — cross-workflow listing with search attributes
- [Advanced Data Table with HTMX](https://benoitaverty.com/articles/en/data-table-with-htmx) — server-rendered table patterns
- [Fly.io Dashboard Framework History](https://community.fly.io/t/a-brief-history-of-fly-io-dashboard-frameworks/19698) — server-push pattern reference

## Next Step

Direction is locked for v1.1. Natural next step is `/adv-proposal "OCA operator dashboard v1.1 — read-only unified table"` to convert this research into a formal change. v1.2 + v2.0 stay as roadmap signal in this note until v1.1 ships.

Decision points still open at v1.1 proposal time (not architecture, just product scope):
- Density default (compact like Linear vs comfortable like Vercel)
- Refresh cadence default (2s, 5s, 10s)
- Browser auto-launch on `oca dashboard` (yes/no)
- Empty-state copy and onboarding hint
- Bind address default — `127.0.0.1` (locked) but configurable via flag for future v2.0 readiness
