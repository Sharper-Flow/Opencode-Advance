# MCP Tool Selection Guide

When multiple tools could accomplish a task, use this guide to select the most appropriate one.

## MCP Invocation Mode (Critical)

MCP tools must be invoked as native tool calls, not as shell commands.

- Correct: call tool APIs directly via the agent's native tool-call mechanism.
- Incorrect: run tool names via `bash` (for example `lgrep_search_semantic "..." .`).
- Use `bash` only for terminal tasks (git, builds, tests, package managers, scripts).

If a name matches a registered MCP tool, invoke it through the MCP tool interface.

### Tool Name Discovery (Mandatory Before First Call)

There are up to **three different name forms** for the same MCP tool in any given session:

| Form | Example | Source | Use it for |
|------|---------|--------|-----------|
| **Function name** (what you invoke) | `kagi_kagi_search_fetch`, `context7_resolve-library-id`, `lgrep_search_semantic` | Agent's own tool schema | All tool calls |
| **Registry name** (shown in errors / docs) | `kagi_kagi_search_fetch`, `context7_resolve-library-id` | MCP server registry, error messages | Reading errors only, unless it exactly appears in your schema |
| **Constructed guess** | `kagi_search_fetch` (missing server segment), `context7_resolve_library_id` (underscored) | Made up — wrong | Never |

Rules:

1. **Always use the exact function name from your own tool schema.** Do not construct names from MCP server docs, error messages, or this file's prose.
2. **Current OpenCode sessions expose many MCP tools as server-prefixed snake_case** (for example `kagi_kagi_search_fetch`, `firecrawl_firecrawl_scrape`, `gh_grep_searchGitHub`, `lgrep_search_semantic`). If your schema shows a different form, use what your schema shows.
3. **Context7 exception:** current OpenCode sessions expose Context7 with exact hyphenated names: `context7_resolve-library-id` and `context7_query-docs`. These are valid function names when present in the schema.
4. **Vision exception:** current sessions may expose Vision admin tools as `vision_vision_*` even though the Vision plugin source registers `vision_*`; use the exact schema name.
5. **When unsure, scan your function list once** at the start of any research task. Treat that scan as the source of truth for the rest of the session.

Symptom of getting this wrong: repeated `Model tried to call unavailable tool 'X'` errors where `X` is a guessed snake_case, hyphenated, or `mcp_` form not present in the schema. Fix: re-read your schema; pick the function name; do not retry the bad name.

### Context7 Naming (Verified Working)

Context7 is the primary tool for library and framework documentation when its tools appear in the active schema. Use this exact two-step flow:

1. `context7_resolve-library-id` — resolve package/product name to Context7 library ID.
2. `context7_query-docs` — query docs with the resolved `/org/project` or `/org/project/version` ID.

Do **not** call guessed names like `context7_resolve_library_id`. They are not equivalent to the schema-exposed hyphenated names.

| Server | Typical callable form | Notes |
|--------|-----------------------|-------|
| Kagi (`kagi`) | `kagi_kagi_search_fetch`, `kagi_kagi_summarizer` | Search + summarizer |
| Lgrep (`lgrep`) | `lgrep_*` | Local code intelligence; snake_case form is canonical in this environment |
| Vision (`vision`) | `vision_vision_*` | Current schema names for Vision admin tools; plugin source registers `vision_*` internally |
| Firecrawl (`firecrawl`) | `firecrawl_firecrawl_*` | Scrape/crawl content extraction |
| GH-grep (`gh_grep`) | `gh_grep_searchGitHub` | Real-world GitHub code examples |
| Context7 (`context7`) | `context7_resolve-library-id`, `context7_query-docs` | Official/library docs; verified working |

If Context7 tools are absent from a future session's schema, fall back to `webfetch` against canonical docs URLs.

## MCP Server Management

**Primary: Vision (`vision`)**
- Use `vision_vision_list` to see all available MCP servers and their status
- Use `vision_vision_add` to add new MCP servers dynamically
- Use `vision_vision_remove` to remove servers
- Use `vision_vision_restart` to restart servers in place
- Use `vision_vision_status` to check daemon health and uptime
- Use `vision_vision_search` to find servers in the catalog
- Use `vision_vision_guidance` to get tool selection recommendations
- Use `vision_vision_metrics` and `vision_vision_slot_status` for operator diagnostics

**When to use**: Managing MCP server lifecycle, checking what tools are available, troubleshooting server issues

## Local Tool Policies

Dedicated always-on instruction files carry the detailed policy for high-impact local tools:

- `lgrep-tools.md` — primary policy for local code exploration
- `morph-tools.md` — primary policy for choosing `morph_edit` vs `edit`/`write`

This file keeps the cross-tool MCP routing rules, while those dedicated files own the
tool-specific first-action guidance.

> Note on tool-name examples below: the `function names` shown match the current OpenCode schema in this environment. Always verify against your own tool schema — see the Tool Name Discovery section above. Treat names below as descriptors, not literal call strings, if they don't appear verbatim in your schema.

## Web Search & Research

**Primary: Kagi**
- Use `kagi_kagi_search_fetch` for web searches, research, current information, news
- Use `kagi_kagi_summarizer` for summarizing web pages and documents
- Fast, high-quality results without tracking

**Avoid**: Using Playwright, Firecrawl, or general fetch tools for simple searches

## Library & API Documentation

**Primary: Context7**
- First call `context7_resolve-library-id` to resolve the library ID.
- Then call `context7_query-docs` with that `/org/project` or `/org/project/version` ID.
- Best for: React, Next.js, TypeScript, Python libraries, etc.

**Fallback**: If Context7 tools are not present in the active schema, use `webfetch` against the canonical docs URL:

| Library / source | URL pattern |
|---|---|
| Anthropic Claude | `https://platform.claude.com/docs/en/...` |
| Vercel AI SDK | `https://ai-sdk.dev/docs/...` or `https://ai-sdk.dev/providers/...` |
| OpenCode | `https://opencode.ai/docs/...` |
| React | `https://react.dev/reference/...` |
| Next.js | `https://nextjs.org/docs/...` |
| MDN | `https://developer.mozilla.org/...` |
| GitHub repo | `https://github.com/{org}/{repo}/blob/{ref}/{path}` |

## Code Examples & Patterns

**Primary: Grep by Vercel**
- Use `gh_grep_searchGitHub` to find real-world usage examples on GitHub
- Great for: implementation patterns, API usage, seeing how others solved problems
- Filter by language with `language`, by repo with `repo`

**Avoid**: Web searching for code examples when the GitHub-grep tool can find them directly

## Web Scraping & Data Extraction

**Primary: Firecrawl - always-on, no add needed**
- Use `firecrawl_firecrawl_scrape` for single page content extraction
- Use `firecrawl_firecrawl_crawl` for multi-page crawl (async - returns job ID)
- Use `firecrawl_firecrawl_check_crawl_status` to poll crawl job results

**When to use over Kagi**: When you need the full page content, structured data, or JS-rendered pages

## Academic Papers

**Primary: arXiv (`arxiv-mcp`)**
- Use `search_papers` for finding research papers
- Use `download_paper` and `read_paper` for full paper content
- Best for: AI/ML research, computer science, physics, math papers

## Browser Automation (Playwright - if available)

**Use ONLY for**:
- E2E testing
- Filling out forms
- Clicking buttons
- Interactive web tasks
- Screenshots of rendered pages
- Exploring interactive application behavior

**NEVER use for**:
- Web search (use Kagi)
- Fetching page content (use Firecrawl)
- Research (use Kagi)
- Documentation lookup (use Context7)
- General web browsing

**Playwright is for exploring application behavior, not browsing.**

## Decision Matrix

| Task | Tool | Why |
|------|------|-----|
| "How is auth handled in this repo?" | `lgrep_search_semantic` | Concept search in local code |
| "Find function `authenticate`" | `lgrep_search_symbols` | Exact symbol lookup |
| "What's in src/auth.py?" | `lgrep_get_file_outline` | AST-based file structure |
| "What's in this codebase?" | `lgrep_get_repo_outline` | Full repo symbol map |
| "Refactor scattered logic in a 600-line file" | `morph_edit` | Partial-file merge is more reliable than exact replacement |
| "Replace one exact string" | `edit` | Fast local change, no API call |
| "Find string `verifyToken`" | `lgrep_search_text` or `grep` | Literal lookup |
| "Open src/auth.ts" | `read` | Direct known-file inspection |
| "Search for X on the web" | Kagi | Purpose-built for search |
| "How do I use React hooks?" | Context7 | Official documentation |
| "Show me examples of useEffect" | grep.app | Real-world code |
| "Get content from example.com" | Firecrawl/fetch | Content extraction |
| "Click the login button" | Playwright | Browser automation |

## Anti-Patterns to Avoid

1. Do not start local exploration with `glob`/`grep` for intent-driven prompts.
2. Do not web search for library docs when Context7 is available.
3. Do not use Playwright for research; it is for automation.
4. Do not use grep.app for local files; use `lgrep`.
5. Do not default to `edit` for large scattered edits when `morph_edit` is available.
6. Do not invoke MCP tools by guessed names (e.g. `context7_resolve_library_id` or `kagi_search_fetch` when your schema exposes `context7_resolve-library-id` or `kagi_kagi_search_fetch`). Always invoke the exact function name from your own tool schema. See Tool Name Discovery above.
