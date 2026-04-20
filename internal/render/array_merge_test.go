package render

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergeArray_BasicMerge(t *testing.T) {
	existing := `{"plugin": ["user/plugin-a", "user/plugin-b"]}`
	declared := []string{"declared/plugin-1", "declared/plugin-2"}

	result, err := MergeArray([]byte(existing), "plugin", declared, MergeArrayOptions{})
	if err != nil {
		t.Fatalf("MergeArray: %v", err)
	}

	root := decodeMergeResult(t, result)
	arr := stringArrayAt(t, root, "plugin")

	if len(arr) != 4 {
		t.Fatalf("len(arr)=%d want 4: %#v", len(arr), arr)
	}
	if arr[0] != "declared/plugin-1" || arr[1] != "declared/plugin-2" {
		t.Fatalf("declared entries not first: %#v", arr)
	}
	if arr[2] != "user/plugin-a" || arr[3] != "user/plugin-b" {
		t.Fatalf("user entries not preserved in place after declared: %#v", arr)
	}
	if result.AddedCount != 2 || result.PreservedCount != 2 || result.PrunedCount != 0 {
		t.Fatalf("counts wrong: %+v", result)
	}
}

func TestMergeArray_DedupesByCanonicalPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	if home == "" {
		t.Skip("cannot determine home dir")
	}

	existing := `{"plugin": ["` + home + `/dev/oc-plugins/advance"]}`
	declared := []string{"~/dev/oc-plugins/advance"}

	result, err := MergeArray([]byte(existing), "plugin", declared, MergeArrayOptions{})
	if err != nil {
		t.Fatalf("MergeArray: %v", err)
	}

	arr := stringArrayAt(t, decodeMergeResult(t, result), "plugin")
	if len(arr) != 1 {
		t.Fatalf("len(arr)=%d want 1: %#v", len(arr), arr)
	}
	if arr[0] != "~/dev/oc-plugins/advance" {
		t.Fatalf("declared form should win on dedupe: %#v", arr)
	}
	if result.AddedCount != 0 || result.PreservedCount != 0 {
		t.Fatalf("counts wrong for dedupe: %+v", result)
	}
}

func TestMergeArray_CreatesArrayIfMissing(t *testing.T) {
	existing := `{"instructions": ["some-instr"]}`
	declared := []string{"plugin/a", "plugin/b"}

	result, err := MergeArray([]byte(existing), "plugin", declared, MergeArrayOptions{})
	if err != nil {
		t.Fatalf("MergeArray: %v", err)
	}

	root := decodeMergeResult(t, result)
	arr := stringArrayAt(t, root, "plugin")
	if len(arr) != 2 {
		t.Fatalf("len(arr)=%d want 2: %#v", len(arr), arr)
	}
	if _, ok := root["instructions"]; !ok {
		t.Fatal("existing sibling key lost")
	}
}

func TestMergeArray_EmptyDeclaredPreservesUserEntries(t *testing.T) {
	existing := `{"plugin": ["user/x"]}`

	result, err := MergeArray([]byte(existing), "plugin", nil, MergeArrayOptions{})
	if err != nil {
		t.Fatalf("MergeArray: %v", err)
	}

	arr := stringArrayAt(t, decodeMergeResult(t, result), "plugin")
	if len(arr) != 1 || arr[0] != "user/x" {
		t.Fatalf("user entry not preserved: %#v", arr)
	}
	if result.AddedCount != 0 || result.PreservedCount != 1 || result.PrunedCount != 0 {
		t.Fatalf("counts wrong: %+v", result)
	}
}

func TestMergeArray_StripsStaleWorktreeUsingRuntimeXDGBase(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	stale := filepath.Join(tmp, "opencode", "worktree", "abc123", "change", "foo", "plugin.md")
	existing := `{"plugin": ["` + stale + `", "user/real-plugin"]}`
	declared := []string{"declared/x"}

	result, err := MergeArray([]byte(existing), "plugin", declared, MergeArrayOptions{StripStaleWorktree: true})
	if err != nil {
		t.Fatalf("MergeArray: %v", err)
	}

	arr := stringArrayAt(t, decodeMergeResult(t, result), "plugin")
	if len(arr) != 2 {
		t.Fatalf("len(arr)=%d want 2: %#v", len(arr), arr)
	}
	for _, entry := range arr {
		if entry == stale {
			t.Fatalf("stale worktree entry not pruned: %#v", arr)
		}
	}
	if result.PrunedCount != 1 {
		t.Fatalf("PrunedCount=%d want 1", result.PrunedCount)
	}
}

func TestMergeArray_PreservesExistingWorktreeWhenPathStillExists(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	valid := filepath.Join(tmp, "opencode", "worktree", "abc123", "change", "foo", "plugin.md")
	if err := os.MkdirAll(filepath.Dir(valid), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(valid, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}

	existing := `{"plugin": ["` + valid + `", "user/real"]}`
	declared := []string{"declared/x"}

	result, err := MergeArray([]byte(existing), "plugin", declared, MergeArrayOptions{StripStaleWorktree: true})
	if err != nil {
		t.Fatalf("MergeArray: %v", err)
	}

	arr := stringArrayAt(t, decodeMergeResult(t, result), "plugin")
	if len(arr) != 3 {
		t.Fatalf("len(arr)=%d want 3: %#v", len(arr), arr)
	}
	if !containsString(arr, valid) {
		t.Fatalf("valid worktree entry lost: %#v", arr)
	}
	if result.PrunedCount != 0 {
		t.Fatalf("PrunedCount=%d want 0", result.PrunedCount)
	}
}

func TestMergeArray_RejectsNonArrayField(t *testing.T) {
	existing := `{"plugin": {"bad": true}}`

	_, err := MergeArray([]byte(existing), "plugin", []string{"declared/x"}, MergeArrayOptions{})
	if err == nil {
		t.Fatal("expected error for non-array field")
	}
	if !strings.Contains(err.Error(), "array") {
		t.Fatalf("error should mention array, got: %v", err)
	}
}

func TestMergeArray_ResultCountsWithPrune(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	stale := filepath.Join(tmp, "opencode", "worktree", "abc123", "change", "foo", "plugin.md")
	existing := `{"plugin": ["` + stale + `", "user/a"]}`
	declared := []string{"declared/1"}

	result, err := MergeArray([]byte(existing), "plugin", declared, MergeArrayOptions{StripStaleWorktree: true})
	if err != nil {
		t.Fatalf("MergeArray: %v", err)
	}
	if result.PrunedCount != 1 || result.AddedCount != 1 || result.PreservedCount != 1 {
		t.Fatalf("counts wrong: %+v", result)
	}
}

func decodeMergeResult(t *testing.T, result *MergeArrayResult) map[string]any {
	t.Helper()
	var root map[string]any
	if err := json.Unmarshal(result.Bytes, &root); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, string(result.Bytes))
	}
	return root
}

func stringArrayAt(t *testing.T, root map[string]any, key string) []string {
	t.Helper()
	raw, ok := root[key]
	if !ok {
		t.Fatalf("missing key %q", key)
	}
	rawArr, ok := raw.([]any)
	if !ok {
		t.Fatalf("key %q not array: %#v", key, raw)
	}
	out := make([]string, 0, len(rawArr))
	for _, item := range rawArr {
		s, ok := item.(string)
		if !ok {
			t.Fatalf("array item not string: %#v", item)
		}
		out = append(out, s)
	}
	return out
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
