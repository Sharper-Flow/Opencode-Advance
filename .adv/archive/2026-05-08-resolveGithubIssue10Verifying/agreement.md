## Agreement

### Objectives
1. Verify `AGENTS.md` no longer contains transitional "scout → plan, refine → build" wording
2. Verify `internal/migrate/stale.go` already guards runtime against stale agent names
3. Add a docs guard test preventing `scout.md`/`refine.md` from being reintroduced as current shipped Advance agents in `AGENTS.md` or `assets/agents/README.md`
4. Close GitHub issue #10 with evidence

### Acceptance Criteria
- [ ] `AGENTS.md` contains no transitional "scout" / "refine" renaming wording in the agent ownership section
- [ ] `internal/migrate/stale.go` still maps `scout.md` and `refine.md` as stale (unchanged — already correct)
- [ ] A test in `tests/polish_docs_test.go` asserts `AGENTS.md` and `assets/agents/README.md` do not reference `scout.md` or `refine.md` as current shipped agents
- [ ] `go test ./...` passes
- [ ] GitHub issue #10 can be closed with comment summarizing audit and guard

### Scope
- `tests/polish_docs_test.go` — add stale agent name guard test
- `AGENTS.md` — verify clean (no edits expected)
- `assets/agents/README.md` — verify clean (no edits expected)
- `internal/migrate/stale.go` — verify clean (no edits expected)

### Out of Scope
- Changing Advance's `scripts/sync-global.sh` behavior
- Modifying live `~/.config/opencode` files
- Modifying `internal/migrate/stale.go`
- Historical archive docs that reference scout/refine in past-tense context