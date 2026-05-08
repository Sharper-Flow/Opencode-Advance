# Archive: Resolve GitHub issue #10 by verifying Advance agent ownership docs and adding a guard against stale agent references

**Change ID:** resolveGithubIssue10Verifying
**Archived:** 2026-05-08T23:35:39.858Z
**Created:** 2026-05-04T19:52:11.483Z

## Tasks Completed

- ✅ Add TestDocsNoStaleAgentNames guard test to tests/polish_docs_test.go
  > Added TestDocsNoStaleAgentNames to polish_docs_test.go. Test asserts AGENTS.md and assets/agents/README.md contain no scout.md or refine.md references. All 18 docs audit files committed alongside.
- ✅ Verify AGENTS.md and assets/agents/README.md are clean of scout.md/refine.md transitional wording
  > Verified: AGENTS.md and assets/agents/README.md contain zero references to scout.md or refine.md. No file changes needed.
- ✅ Run full test suite (go test ./...) and verify all pass
  > go test ./... — all 18 packages pass, including new TestDocsNoStaleAgentNames. 0 failures.
- ✅ Comment on GitHub issue #10 with audit summary and close
  > GitHub issue #10 closed with full audit summary. Issue Sharper-Flow/Opencode-Advance#10.

## Specs Modified

