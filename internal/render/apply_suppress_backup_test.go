package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApply_SuppressBackupSkipsBakSideFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "opencode.json")
	lockPath := filepath.Join(root, "apply.lock")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan := &Plan{
		LockPath: lockPath,
		Targets: []TargetOp{{
			Name:           "opencode.json",
			Path:           path,
			Op:             "merge",
			After:          []byte("new"),
			Mode:           0o644,
			SuppressBackup: true,
		}},
	}

	res, err := Apply(plan, ApplyOptions{LockPath: lockPath, MaxBackups: 3})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(res.Targets) != 1 {
		t.Fatalf("targets=%d want 1", len(res.Targets))
	}
	if res.Targets[0].BackupPath != "" {
		t.Fatalf("backup path=%q want empty", res.Targets[0].BackupPath)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".bak.") {
			t.Fatalf("unexpected backup side-file: %s", entry.Name())
		}
	}
}

func TestApply_WithoutSuppressBackupKeepsNormalBakBehavior(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "opencode.json")
	lockPath := filepath.Join(root, "apply.lock")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan := &Plan{
		LockPath: lockPath,
		Targets: []TargetOp{{
			Name:   "opencode.json",
			Path:   path,
			Op:     "merge",
			After:  []byte("new"),
			Mode:   0o644,
		}},
	}

	res, err := Apply(plan, ApplyOptions{LockPath: lockPath, MaxBackups: 3})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if res.Targets[0].BackupPath == "" {
		t.Fatal("backup path empty; want normal backup behavior")
	}
	if _, err := os.Stat(res.Targets[0].BackupPath); err != nil {
		t.Fatalf("backup path missing: %v", err)
	}
}
