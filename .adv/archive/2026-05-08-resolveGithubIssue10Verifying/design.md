## Design

### Approach
Add a single test function `TestDocsNoStaleAgentNames` to `tests/polish_docs_test.go` that asserts `AGENTS.md` and `assets/agents/README.md` do not contain `scout.md` or `refine.md` as current shipped agents.

### Implementation
- Pattern: reuse existing `polish_docs_test.go` pattern (read file, check forbidden substrings, fail with `t.Fatalf`)
- Forbidden strings: `"scout.md"`, `"refine.md"` — both in the context of agent file lists
- Files checked: `AGENTS.md`, `assets/agents/README.md`
- Note: `internal/migrate/stale.go` intentionally NOT checked — it correctly references these names as stale markers

### Error Handling
- Test failure = clear `t.Fatalf` message indicating which file contains the stale name
- No runtime error handling needed — this is a static docs test

### Verification
- `go test ./tests/ -run TestDocsNoStaleAgentNames` passes
- `go test ./...` passes