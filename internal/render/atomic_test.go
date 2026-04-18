package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteAtomic_WritesAndCreatesBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	backup, err := WriteAtomic(path, []byte("new"), 0o600, 3)
	if err != nil {
		t.Fatalf("WriteAtomic: %v", err)
	}
	if backup == "" || !strings.Contains(backup, ".bak.") {
		t.Fatalf("backup path missing: %q", backup)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "new" {
		t.Fatalf("file = %q, want new", got)
	}
	st, _ := os.Stat(path)
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, want 0600", st.Mode().Perm())
	}
	old, _ := os.ReadFile(backup)
	if string(old) != "old" {
		t.Fatalf("backup = %q, want old", old)
	}
}

func TestWriteAtomic_NoopWhenUnchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "same.txt")
	if err := os.WriteFile(path, []byte("same"), 0o644); err != nil {
		t.Fatal(err)
	}
	backup, err := WriteAtomic(path, []byte("same"), 0o644, 3)
	if err != nil {
		t.Fatal(err)
	}
	if backup != "" {
		t.Fatalf("expected no backup on noop, got %q", backup)
	}
}
