---
name: mcp-selection
description: "MCP tool selection guide — use when choosing between Kagi, Context7, grep.app, Firecrawl, Playwright, or other MCP tools for a task. Provides decision matrix and anti-patterns."
keywords: ["mcp", "tool-selection", "context7", "kagi", "grep.app", "firecrawl", "playwright", "documentation", "web-search"]
license: MIT
metadata:
  priority: high
  replaces: none
---

## When to Load This Skill

Load this skill when you need to decide **which MCP tool** to use for a task. If you already know the right tool, skip this and call it directly.

## MCP Server Management

**Primary: Vision (`vision`)**
- Use `vision_vision_list` to see all available MCP servers and their status
- Use `vision_vision_add` to add new MCP servers dynamically
- Use `vision_vision_remove` to remove servers
- Use `vision_vision_status` to check daemon health and uptime
- Use `vision_vision_search` to find servers in the catalog
- Use `vision_vision_guidance` to get tool selection recommendations

**When to use**: Managing MCP server lifecycle, checking what tools are available, troubleshooting server issues

## Web Search & Research

**Primary: Kagi (`kagimcp`)**
- Use `kagi_kagi_search_fetch` for web searches, research, current information, news
- Use `kagi_kagi_summarizer` for summarizing web pages and documents
- Fast, high-quality results without tracking

**Avoid**: Using Playwright, Firecrawl, or general fetch tools for simple searches

## Library & API Documentation

**Primary: Context7 (`context7`)**
- Use `context7_resolve-library-id` first to find the library ID
- Use `context7_query-docs` to get documentation for specific questions
- Best for: React, Next.js, TypeScript, Python libraries, etc.

**Avoid**: Web searching for documentation that Context7 has indexed

## Code Examples & Patterns

**Primary: Grep by Vercel (`gh_grep`)**
- Use `gh_grep_searchGitHub` to find real-world usage examples on GitHub
- Great for: implementation patterns, API usage, seeing how others solved problems
- Filter by language with `langFilter`, by repo with `repoFilter`

**Avoid**: Web searching for code examples when `gh_grep` can find them directly

## Web Scraping & Data Extraction

**Primary: Firecrawl (`firecrawl`) — always-on, no add needed**
- Use `firecrawl_firecrawl_scrape` for single page content extraction
- Use `firecrawl_firecrawl_crawl` for multi-page crawl (async — returns job ID)
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
- Fetching page content (use Firecrawl or fetch)
- Research (use Kagi)
- Documentation lookup (use Context7)
- General web browsing

**Playwright is for exploring application behavior, not browsing.**

## Decision Matrix

| Task | Tool | Why |
|------|------|-----|
| "Search for X" | Kagi | Purpose-built for search |
| "How do I use React hooks?" | Context7 | Official documentation |
| "Show me examples of useEffect" | grep.app | Real-world code |
| "Get content from example.com" | Firecrawl | Content extraction |
| "Summarize this article" | Kagi summarizer | Built-in summarization |
| "Find recent AI papers" | arXiv | Academic paper search |
| "Click the login button" | Playwright | Browser automation |

## Anti-Patterns to Avoid

1. **Don't use Playwright for research** - It's slow and meant for automation
2. **Don't web search for library docs** - Context7 has them indexed
3. **Don't scrape when you can search** - Kagi is faster for finding info
4. **Don't use fetch for complex pages** - Firecrawl handles JavaScript rendering

## Keywords
mcp tool selection, which tool to use, kagi vs context7, web search, documentation lookup,
code examples, web scraping, browser automation, playwright, firecrawl, grep.app, arxiv,
fetch, url content, tool choice, decision matrix
