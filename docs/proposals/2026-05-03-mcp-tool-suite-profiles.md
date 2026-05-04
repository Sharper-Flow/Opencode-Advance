# MCP Tool Suite Profiles and Per-Agent Exposure

**Status:** Proposal — staged  
**Date:** 2026-05-03  
**Target repo:** `~/dev/opencodeadvance`  
**Resume from:** `~/dev/opencodeadvance`  
**Suggested change ID:** `mcpToolSuiteProfiles`  
**Priority:** SHOULD

---

## Problem Statement

OCA models MCP mostly as servers. That is correct for install/render, but agent
runtime behavior depends on tool exposure. Exposing every MCP tool to every
agent increases prompt/schema load and increases wrong-tool risk.

Layer strategy says MCP servers are external capabilities; OCA should manage
which agents see which tool suites.

---

## Success Criteria

- [ ] Stack schema supports named MCP tool suites or agent-level MCP exposure
      profiles.
- [ ] OCA can express at least these suites:
      - `research`: Kagi, Context7, Firecrawl, gh_grep
      - `code-intel`: lgrep
      - `browser`: Playwright slot groups
      - `ops`: Vision/Sentry/system MCP tools
- [ ] Agent profiles can include/exclude suites.
- [ ] Rendered OpenCode config uses permissions/wildcards consistent with current
      OpenCode docs.
- [ ] Doctor reports exposed MCP suite counts per agent.

---

## Out of Scope

- Changing Vision server lifecycle.
- Implementing new MCP servers.
- Removing manual user-added MCP entries.

---

## Implementation Sketch

1. Add schema for tool suites or agent profile references.
2. Translate suites into `permission` wildcard entries.
3. Preserve user-added permissions while overwriting OCA-declared profile keys.
4. Add tests for suite rendering and unknown suite validation.
5. Update `mcp-selection` skill to align with suite names.

---

## Acceptance Criteria

1. `explore` sees lgrep/read/search tools but not browser/web scraping tools.
2. `librarian` sees docs/web tools but not local write/bash tools.
3. `build/general` exposure matches configured profile.
4. `go test ./...` passes.
