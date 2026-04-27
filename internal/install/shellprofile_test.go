package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testBlockContent = "# test managed content\nexport FOO=bar"
)

// --- ReadBlock tests ---

func TestReadBlock(t *testing.T) {
	tests := []struct {
		name    string
		content string
		noFile  bool
		want    string
		wantErr bool
	}{
		{
			name:    "file does not exist",
			content: "",
			noFile:  true,
			want:    "",
			wantErr: false,
		},
		{
			name:    "empty file",
			content: "",
			want:    "",
			wantErr: false,
		},
		{
			name: "no block present",
			content: "export PATH=/usr/bin:$PATH\nalias ll='ls -la'\n",
			want:    "",
			wantErr: false,
		},
		{
			name: "block present",
			content: "export PATH=/usr/bin:$PATH\n" +
				"# >>> OCA Managed Block >>>\n" +
				"# managed content\n" +
				"# <<< OCA Managed Block <<<\n" +
				"alias ll='ls -la'\n",
			want:    "# managed content\n",
			wantErr: false,
		},
		{
			name: "block with empty content",
			content: "# >>> OCA Managed Block >>>\n" +
				"# <<< OCA Managed Block <<<\n",
			want:    "",
			wantErr: false,
		},
		{
			name: "multiple blocks error",
			content: "# >>> OCA Managed Block >>>\n" +
				"first\n" +
				"# <<< OCA Managed Block <<<\n" +
				"other stuff\n" +
				"# >>> OCA Managed Block >>>\n" +
				"second\n" +
				"# <<< OCA Managed Block <<<\n",
			want:    "",
			wantErr: true,
		},
		{
			name: "start sentinel without end",
			content: "# >>> OCA Managed Block >>>\n" +
				"some content\n" +
				"no end sentinel\n",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, ".bashrc")

			if !tt.noFile {
				if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			got, err := ReadBlock(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReadBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- WriteBlock tests ---

func TestWriteBlock(t *testing.T) {
	t.Run("write to nonexistent file creates file with block", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".bashrc")

		err := WriteBlock(path, testBlockContent)
		if err != nil {
			t.Fatal(err)
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(got), SentinelStart) {
			t.Error("file should contain start sentinel")
		}
		if !strings.Contains(string(got), SentinelEnd) {
			t.Error("file should contain end sentinel")
		}
		if !strings.Contains(string(got), testBlockContent) {
			t.Error("file should contain block content")
		}
	})

	t.Run("write to file without existing block appends block", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".bashrc")
		existing := "export PATH=/usr/bin:$PATH\nalias ll='ls -la'\n"
		if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
			t.Fatal(err)
		}

		err := WriteBlock(path, testBlockContent)
		if err != nil {
			t.Fatal(err)
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(string(got), existing) {
			t.Error("existing content should be preserved")
		}
		if !strings.Contains(string(got), SentinelStart) {
			t.Error("should contain start sentinel")
		}
	})

	t.Run("write replaces existing block idempotently", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".bashrc")

		// First write
		if err := WriteBlock(path, "content v1"); err != nil {
			t.Fatal(err)
		}
		after1, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		// Second write with same content — file should not change (WriteAtomic skips)
		if err := WriteBlock(path, "content v1"); err != nil {
			t.Fatal(err)
		}
		after2, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(after1) != string(after2) {
			t.Error("idempotent write should produce identical file")
		}

		// Third write with different content — block should be replaced
		if err := WriteBlock(path, "content v2"); err != nil {
			t.Fatal(err)
		}
		after3, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(after2) == string(after3) {
			t.Error("different content should produce different file")
		}
		if strings.Contains(string(after3), "content v1") {
			t.Error("old content should be replaced")
		}
		if !strings.Contains(string(after3), "content v2") {
			t.Error("new content should be present")
		}
	})

	t.Run("write creates backup of existing file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".bashrc")
		existing := "original content\n"
		if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := WriteBlock(path, testBlockContent); err != nil {
			t.Fatal(err)
		}

		// Check backup exists
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".bashrc.bak.") {
				found = true
				break
			}
		}
		if !found {
			t.Error("backup file should be created")
		}
	})
}

// --- DiffBlock / IsEdited tests ---

func TestDiffBlock(t *testing.T) {
	t.Run("no block present", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".bashrc")
		os.WriteFile(path, []byte("some content\n"), 0o644)

		diff, err := DiffBlock(path, "expected")
		if err != nil {
			t.Fatal(err)
		}
		if diff.HasBlock {
			t.Error("HasBlock should be false")
		}
	})

	t.Run("block matches expected", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".bashrc")
		content := "# before\n" +
			"# >>> OCA Managed Block >>>\n" +
			"expected content\n" +
			"# <<< OCA Managed Block <<<\n"
		os.WriteFile(path, []byte(content), 0o644)

		diff, err := DiffBlock(path, "expected content\n")
		if err != nil {
			t.Fatal(err)
		}
		if !diff.HasBlock {
			t.Error("HasBlock should be true")
		}
		if !diff.Matches {
			t.Errorf("Matches should be true, got diff:\n%s", diff.Diff)
		}
	})

	t.Run("block differs from expected", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".bashrc")
		content := "# >>> OCA Managed Block >>>\n" +
			"edited content\n" +
			"# <<< OCA Managed Block <<<\n"
		os.WriteFile(path, []byte(content), 0o644)

		diff, err := DiffBlock(path, "expected content\n")
		if err != nil {
			t.Fatal(err)
		}
		if !diff.HasBlock {
			t.Error("HasBlock should be true")
		}
		if diff.Matches {
			t.Error("Matches should be false when content differs")
		}
		if diff.Diff == "" {
			t.Error("Diff should be non-empty when content differs")
		}
	})
}

func TestIsEdited(t *testing.T) {
	t.Run("no block returns false", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".bashrc")
		os.WriteFile(path, []byte("no block here\n"), 0o644)

		edited, err := IsEdited(path, "expected")
		if err != nil {
			t.Fatal(err)
		}
		if edited {
			t.Error("should be false when no block")
		}
	})

	t.Run("matching block returns false", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".bashrc")
		content := "# >>> OCA Managed Block >>>\n" +
			"expected\n" +
			"# <<< OCA Managed Block <<<\n"
		os.WriteFile(path, []byte(content), 0o644)

		edited, err := IsEdited(path, "expected\n")
		if err != nil {
			t.Fatal(err)
		}
		if edited {
			t.Error("should be false when content matches")
		}
	})

	t.Run("edited block returns true", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".bashrc")
		content := "# >>> OCA Managed Block >>>\n" +
			"edited!\n" +
			"# <<< OCA Managed Block <<<\n"
		os.WriteFile(path, []byte(content), 0o644)

		edited, err := IsEdited(path, "expected\n")
		if err != nil {
			t.Fatal(err)
		}
		if !edited {
			t.Error("should be true when content differs")
		}
	})
}

// --- RemoveBlock tests ---

func TestRemoveBlock(t *testing.T) {
	t.Run("removes block from middle of file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".bashrc")
		content := "# header\n" +
			"# >>> OCA Managed Block >>>\n" +
			"managed\n" +
			"# <<< OCA Managed Block <<<\n" +
			"# footer\n"
		os.WriteFile(path, []byte(content), 0o644)

		err := RemoveBlock(path)
		if err != nil {
			t.Fatal(err)
		}

		got, _ := os.ReadFile(path)
		expected := "# header\n# footer\n"
		if string(got) != expected {
			t.Errorf("after remove = %q, want %q", string(got), expected)
		}
	})

	t.Run("file without block is unchanged", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".bashrc")
		content := "no block here\n"
		os.WriteFile(path, []byte(content), 0o644)

		err := RemoveBlock(path)
		if err != nil {
			t.Fatal(err)
		}

		got, _ := os.ReadFile(path)
		if string(got) != content {
			t.Error("file without block should be unchanged")
		}
	})

	t.Run("nonexistent file returns nil", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".bashrc")

		err := RemoveBlock(path)
		if err != nil {
			t.Errorf("nonexistent file should not error, got: %v", err)
		}
	})
}

// --- RenderShellProfile tests ---

func TestRenderShellProfile(t *testing.T) {
	t.Run("contains PATH export with binary dir", func(t *testing.T) {
		got := RenderShellProfile("/usr/local/bin/oca")
		if !strings.Contains(got, "/usr/local/bin") {
			t.Error("should contain the binary directory")
		}
		if !strings.Contains(got, "PATH") {
			t.Error("should contain PATH export")
		}
	})

	t.Run("starts and ends with sentinels", func(t *testing.T) {
		got := RenderShellProfile("/usr/local/bin/oca")
		if !strings.HasPrefix(got, SentinelStart+"\n") {
			t.Errorf("should start with sentinel, got: %q", got[:min(50, len(got))])
		}
		if !strings.HasSuffix(strings.TrimSpace(got), SentinelEnd) {
			t.Errorf("should end with sentinel, got: %q", got[max(0, len(got)-50):])
		}
	})

	t.Run("contains idempotent PATH guard", func(t *testing.T) {
		got := RenderShellProfile("/opt/oca/bin/oca")
		if !strings.Contains(got, "case") {
			t.Error("should contain case statement for PATH idempotency")
		}
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
