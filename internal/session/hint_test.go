package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWriteKillSentinel(t *testing.T) {
	tmpDir := t.TempDir()
	name := "oca-test-1"

	if err := WriteKillSentinel(name, tmpDir); err != nil {
		t.Fatalf("WriteKillSentinel: %v", err)
	}

	path := filepath.Join(tmpDir, "kill-sentinel", name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("sentinel file not created")
	}

	// Verify directory permissions
	info, err := os.Stat(filepath.Join(tmpDir, "kill-sentinel"))
	if err != nil {
		t.Fatalf("stat sentinel dir: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Errorf("sentinel dir perms = %o, want 0755", info.Mode().Perm())
	}
}

func TestCheckKillSentinel(t *testing.T) {
	tmpDir := t.TempDir()
	name := "oca-test-1"

	// No sentinel → false
	if CheckKillSentinel(name, tmpDir) {
		t.Fatal("expected false when no sentinel exists")
	}

	// Write sentinel → true
	if err := WriteKillSentinel(name, tmpDir); err != nil {
		t.Fatalf("WriteKillSentinel: %v", err)
	}
	if !CheckKillSentinel(name, tmpDir) {
		t.Fatal("expected true when sentinel exists")
	}
}

func TestCleanupKillSentinel(t *testing.T) {
	tmpDir := t.TempDir()
	name := "oca-test-1"

	if err := WriteKillSentinel(name, tmpDir); err != nil {
		t.Fatalf("WriteKillSentinel: %v", err)
	}

	CleanupKillSentinel(name, tmpDir)

	if CheckKillSentinel(name, tmpDir) {
		t.Fatal("sentinel should be removed after cleanup")
	}

	// Cleanup non-existent is fine (no error)
	CleanupKillSentinel("nonexistent", tmpDir)
}

func TestFormatHint(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		workdir  string
		noColor  bool
		wantHas  []string
		wantSkip []string
	}{
		{
			name:    "full format with emoji",
			id:      "ses_abc123",
			workdir: "/home/user/project",
			noColor: false,
			wantHas: []string{"💤", "opencode --session ses_abc123", "/home/user/project"},
		},
		{
			name:     "NO_COLOR plain format",
			id:       "ses_abc123",
			workdir:  "/home/user/project",
			noColor:  true,
			wantHas:  []string{"opencode --session ses_abc123", "/home/user/project"},
			wantSkip: []string{"💤"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatHint(tt.id, tt.workdir, tt.noColor)
			for _, want := range tt.wantHas {
				if !strings.Contains(got, want) {
					t.Errorf("FormatHint() = %q, want to contain %q", got, want)
				}
			}
			for _, skip := range tt.wantSkip {
				if strings.Contains(got, skip) {
					t.Errorf("FormatHint() = %q, want NOT to contain %q", got, skip)
				}
			}
		})
	}
}

func TestFormatHintFile(t *testing.T) {
	got := FormatHintFile("ses_abc123", "/home/user/project")
	want := "opencode --session ses_abc123 --project /home/user/project"
	if got != want {
		t.Errorf("FormatHintFile() = %q, want %q", got, want)
	}
}

func TestEmitHintToFile(t *testing.T) {
	tmpDir := t.TempDir()
	hint := "opencode --session ses_abc123 --project /home/user/project"

	if err := EmitHintToFile(hint, tmpDir); err != nil {
		t.Fatalf("EmitHintToFile: %v", err)
	}

	path := filepath.Join(tmpDir, "last-session-hint")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read hint file: %v", err)
	}

	if string(data) != hint {
		t.Errorf("hint file = %q, want %q", string(data), hint)
	}

	// Verify file permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat hint file: %v", err)
	}
	// On non-Windows, check for 0600
	if runtime.GOOS != "windows" {
		if info.Mode().Perm() != 0o600 {
			t.Errorf("hint file perms = %o, want 0600", info.Mode().Perm())
		}
	}
}

func TestParseOpenCodeSessions(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		workdir string
		wantID  string
		wantErr bool
	}{
		{
			name: "single matching session",
			input: `[{"id":"ses_111","title":"test","updated":1778546399316,"created":1778520585734,"projectId":"abc","directory":"/home/user/project"}]`,
			workdir: "/home/user/project",
			wantID:  "ses_111",
		},
		{
			name: "multiple sessions pick most recent",
			input: `[
				{"id":"ses_old","title":"old","updated":1000000,"created":1000000,"projectId":"abc","directory":"/home/user/project"},
				{"id":"ses_new","title":"new","updated":9999999,"created":9999999,"projectId":"abc","directory":"/home/user/project"}
			]`,
			workdir: "/home/user/project",
			wantID:  "ses_new",
		},
		{
			name: "no matching session",
			input: `[{"id":"ses_111","title":"test","updated":1778546399316,"created":1778520585734,"projectId":"abc","directory":"/other/project"}]`,
			workdir: "/home/user/project",
			wantErr: true,
		},
		{
			name:    "empty array",
			input:   `[]`,
			workdir: "/home/user/project",
			wantErr: true,
		},
		{
			name:    "malformed JSON",
			input:   `{broken`,
			workdir: "/home/user/project",
			wantErr: true,
		},
		{
			name: "missing directory field",
			input: `[{"id":"ses_111","title":"test","updated":1778546399316}]`,
			workdir: "/home/user/project",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseOpenCodeSessions([]byte(tt.input), tt.workdir)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseOpenCodeSessions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.wantID {
				t.Errorf("ParseOpenCodeSessions() = %q, want %q", got, tt.wantID)
			}
		})
	}
}

func TestFilterTMUX(t *testing.T) {
	env := []string{
		"PATH=/usr/bin",
		"TMUX=/tmp/tmux-1000/default,12345,0",
		"TMUX_PANE=%0",
		"HOME=/home/user",
	}
	filtered := FilterTMUX(env)

	for _, e := range filtered {
		if strings.HasPrefix(e, "TMUX=") || strings.HasPrefix(e, "TMUX_PANE=") {
			t.Errorf("filterTMUX should remove TMUX vars, got %q", e)
		}
	}

	// Should keep non-TMUX vars
	has := make(map[string]bool)
	for _, e := range filtered {
		has[strings.SplitN(e, "=", 2)[0]] = true
	}
	if !has["PATH"] || !has["HOME"] {
		t.Error("filterTMUX should keep non-TMUX vars")
	}
}

func TestResolveCacheDir(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantDir string
	}{
		{
			name:    "OCA_CACHE_DIR set",
			env:     map[string]string{"OCA_CACHE_DIR": "/custom/cache"},
			wantDir: "/custom/cache",
		},
		{
			name:    "XDG_RUNTIME_DIR fallback",
			env:     map[string]string{"XDG_RUNTIME_DIR": "/run/user/1000"},
			wantDir: "/run/user/1000/opencode-advance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore env
			saved := map[string]string{}
			for _, key := range []string{"OCA_CACHE_DIR", "XDG_RUNTIME_DIR"} {
				saved[key] = os.Getenv(key)
				os.Unsetenv(key)
			}
			defer func() {
				for k, v := range saved {
					os.Setenv(k, v)
				}
			}()

			for k, v := range tt.env {
				os.Setenv(k, v)
			}

			got := ResolveCacheDir()
			if got != tt.wantDir {
				t.Errorf("ResolveCacheDir() = %q, want %q", got, tt.wantDir)
			}
		})
	}
}

// Verify the openCodeSession struct matches the live schema
func TestOpenCodeSessionJSONRoundTrip(t *testing.T) {
	raw := `{"id":"ses_1e7e989f9ffe72hBYizcv9T02I","title":"Test session","updated":1778546399316,"created":1778520585734,"projectId":"abc123","directory":"/home/user/project"}`
	var s openCodeSession
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if s.ID != "ses_1e7e989f9ffe72hBYizcv9T02I" {
		t.Errorf("ID = %q, want ses_1e7e989f9ffe72hBYizcv9T02I", s.ID)
	}
	if s.Directory != "/home/user/project" {
		t.Errorf("Directory = %q", s.Directory)
	}
	if s.Updated != 1778546399316 {
		t.Errorf("Updated = %d, want 1778546399316", s.Updated)
	}

	// Verify unknown fields don't break parse
	rawWithExtra := raw[:len(raw)-1] + `,"extra":"field"}`
	var s2 openCodeSession
	if err := json.Unmarshal([]byte(rawWithExtra), &s2); err != nil {
		t.Fatalf("unmarshal with extra fields: %v", err)
	}
	if s2.ID != s.ID {
		t.Error("extra fields should not affect parsed values")
	}
}

func TestWriteKillSentinelAtomic(t *testing.T) {
	tmpDir := t.TempDir()
	name := "oca-atomic-test"

	if err := WriteKillSentinel(name, tmpDir); err != nil {
		t.Fatalf("WriteKillSentinel: %v", err)
	}

	// Verify no temp files left behind
	entries, err := os.ReadDir(filepath.Join(tmpDir, "kill-sentinel"))
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") || strings.Contains(e.Name(), ".") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}

	// Verify exact file exists
	sentinelPath := filepath.Join(tmpDir, "kill-sentinel", name)
	if _, err := os.Stat(sentinelPath); err != nil {
		t.Errorf("sentinel file missing: %v", err)
	}

	// File content should be minimal (timestamp or empty)
	data, err := os.ReadFile(sentinelPath)
	if err != nil {
		t.Fatalf("read sentinel: %v", err)
	}
	// Content should be non-empty (timestamp) but small
	if len(data) == 0 {
		t.Error("sentinel file is empty, expected timestamp or marker")
	}
	_ = fmt.Sprintf("sentinel content: %s", string(data)) // debug hint
}
