package tests

import (
	"bytes"
	"testing"

	"github.com/Sharper-Flow/Opencode-Advance/internal/render"
)

// contains returns true if data contains needle, accepting both
// spaced and non-spaced JSON formats.
func contains(data, needle []byte) bool {
	return bytes.Contains(data, needle) ||
		bytes.Contains(data, bytes.ReplaceAll(needle, []byte(`": `), []byte(`":`)))
}

// TestMergeObject_BasicMerge tests that declared keys overwrite existing ones.
func TestMergeObject_BasicMerge(t *testing.T) {
	existing := []byte(`{"mcp": {"kagi": {"port": 6276}}}`)
	declared := map[string]any{
		"mcp": map[string]any{
			"kagi": map[string]any{
				"port": 6277,
				"type": "remote",
			},
		},
	}
	result, err := render.MergeObject(existing, declared)
	if err != nil {
		t.Fatalf("MergeObject failed: %v", err)
	}
	if !contains(result, []byte(`"port":`)) || !bytes.Contains(result, []byte(`6277`)) {
		t.Errorf("result does not contain port 6277: %s", result)
	}
	if !contains(result, []byte(`"type":`)) {
		t.Errorf("result does not contain type field: %s", result)
	}
}

// TestMergeObject_PreservesUserKeys tests that undeclared keys in existing
// JSON are preserved (key design property).
func TestMergeObject_PreservesUserKeys(t *testing.T) {
	existing := []byte(`{"provider": {"google": {"models": {"gemini": {"name": "Gemini"}}}}}`)
	declared := map[string]any{
		"provider": map[string]any{
			"google": map[string]any{
				"models": map[string]any{
					"gemini": map[string]any{
						"limit": map[string]any{"context": 1048576},
					},
				},
			},
		},
	}
	result, err := render.MergeObject(existing, declared)
	if err != nil {
		t.Fatalf("MergeObject failed: %v", err)
	}
	// name field should be preserved from existing.
	if !contains(result, []byte(`"name":`)) {
		t.Errorf("result does not preserve existing name field: %s", result)
	}
	// limit should be added.
	if !contains(result, []byte(`"limit"`)) {
		t.Errorf("result does not contain limit: %s", result)
	}
}

// TestMergeObject_NestedDeep tests 3-level nesting with user keys at all levels.
func TestMergeObject_NestedDeep(t *testing.T) {
	existing := []byte(`{"lsp": {"pyright": {"user_field": "preserved"}}}`)
	declared := map[string]any{
		"lsp": map[string]any{
			"pyright": map[string]any{
				"command":    []string{"pyright", "lsp"},
				"extensions": []string{".py"},
			},
		},
	}
	result, err := render.MergeObject(existing, declared)
	if err != nil {
		t.Fatalf("MergeObject failed: %v", err)
	}
	if !contains(result, []byte(`"user_field":`)) {
		t.Errorf("user_field not preserved at lsp.pyright level: %s", result)
	}
}

// TestMergeObject_EmptyExisting tests that a nil/empty existing doc works.
func TestMergeObject_EmptyExisting(t *testing.T) {
	result, err := render.MergeObject(nil, map[string]any{
		"provider": map[string]any{"google": map[string]any{}},
	})
	if err != nil {
		t.Fatalf("MergeObject with nil failed: %v", err)
	}
	if !bytes.HasPrefix(result, []byte("{")) {
		t.Errorf("expected JSON object, got: %s", result)
	}
}

// TestMergeObject_UserOnlyExisting tests that if declared is a subset of existing,
// the result preserves the full existing (declared adds nothing at object level).
func TestMergeObject_UserOnlyExisting(t *testing.T) {
	existing := []byte(`{"provider": {"google": {"user_key": "user_val"}}}`)
	declared := map[string]any{
		"provider": map[string]any{
			"google": map[string]any{}, // empty — no new keys
		},
	}
	result, err := render.MergeObject(existing, declared)
	if err != nil {
		t.Fatalf("MergeObject failed: %v", err)
	}
	// existing should be fully preserved (declared added nothing new at object level).
	if !contains(result, []byte(`"user_key":`)) {
		t.Errorf("user key not preserved: %s", result)
	}
}

// TestMergeObject_ArrayOverwrite tests that top-level array keys are overwritten,
// not merged (matching existing MergeArray behavior for arrays).
func TestMergeObject_ArrayKeyOverwrite(t *testing.T) {
	existing := []byte(`{"watcher": {"ignore": ["old"]}}`)
	declared := map[string]any{
		"watcher": map[string]any{
			"ignore": []any{"node_modules/**", ".git/**"},
		},
	}
	result, err := render.MergeObject(existing, declared)
	if err != nil {
		t.Fatalf("MergeObject failed: %v", err)
	}
	if !contains(result, []byte(`"node_modules/**`)) {
		t.Errorf("node_modules/** not in result: %s", result)
	}
	// new array replaces old, so "old" should NOT be present.
	if bytes.Contains(result, []byte(`"old"`)) {
		t.Errorf("old value should be overwritten by new array: %s", result)
	}
}

// TestMergeObject_NewTopLevelKey tests that entirely new top-level keys are added.
func TestMergeObject_NewTopLevelKey(t *testing.T) {
	existing := []byte(`{"mcp": {}}`)
	declared := map[string]any{
		"permission": map[string]any{"*": "allow"},
	}
	result, err := render.MergeObject(existing, declared)
	if err != nil {
		t.Fatalf("MergeObject failed: %v", err)
	}
	if !bytes.Contains(result, []byte(`"permission"`)) {
		t.Errorf("permission key not added: %s", result)
	}
}
