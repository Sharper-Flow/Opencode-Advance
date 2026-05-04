package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorktreeSkillDefersToAdvWorktree(t *testing.T) {
	contentBytes, err := os.ReadFile(filepath.Join("..", "assets", "skills", "worktree", "SKILL.md"))
	if err != nil {
		t.Fatalf("read worktree skill: %v", err)
	}
	content := string(contentBytes)
	for _, banned := range []string{"openchad", "open-chad", "oc switch", "git checkout trunk", "worktree_create succeeds"} {
		if strings.Contains(content, banned) {
			t.Fatalf("worktree skill contains stale guidance %q", banned)
		}
	}
	for _, required := range []string{"adv-worktree", "adv_worktree_create", "adv_worktree_delete"} {
		if !strings.Contains(content, required) {
			t.Fatalf("worktree skill missing %q", required)
		}
	}
}
