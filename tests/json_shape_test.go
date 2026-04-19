package tests

import (
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

// TestDebugPlanJSONShape verifies `oca debug plan --output json` emits a
// single object with the documented Plan shape (PascalCase keys from the
// render.Plan struct — no json tags).
func TestDebugPlanJSONShape(t *testing.T) {
	root := repoRoot(t)
	tmp := t.TempDir()
	env := []string{
		"OCA_OPENCODE_CONFIG_DIR=" + filepath.Join(tmp, "opencode"),
		"OCA_VISION_CONFIG_DIR=" + filepath.Join(tmp, "vision"),
		"OCA_CACHE_DIR=" + filepath.Join(tmp, "cache"),
	}
	stdout, stderr, err := runOCA(t, root, env, "debug", "plan", "--config", "stack.example.toml", "--output", "json")
	if err != nil {
		t.Fatalf("debug plan failed: err=%v stderr=%s", err, stderr)
	}
	var plan map[string]any
	if uerr := json.Unmarshal([]byte(stdout), &plan); uerr != nil {
		t.Fatalf("debug plan stdout is not a single JSON object: %v\nstdout=%s", uerr, stdout)
	}
	for _, key := range []string{"Source", "LockPath", "Targets"} {
		if _, ok := plan[key]; !ok {
			t.Errorf("Plan JSON missing key %q; got keys=%v", key, mapKeys(plan))
		}
	}
	targets, ok := plan["Targets"].([]any)
	if !ok {
		t.Fatalf("Plan.Targets is not an array; got %T", plan["Targets"])
	}
	if len(targets) == 0 {
		t.Fatalf("Plan.Targets is empty; expected at least one TargetOp for stack.example.toml")
	}
	first, ok := targets[0].(map[string]any)
	if !ok {
		t.Fatalf("Plan.Targets[0] is not an object; got %T", targets[0])
	}
	for _, key := range []string{"Name", "Path", "Op"} {
		if _, ok := first[key]; !ok {
			t.Errorf("TargetOp JSON missing key %q; got keys=%v", key, mapKeys(first))
		}
	}
}

// TestApplyJSONShape verifies `oca apply --target mcp --output json` emits
// a stream of 1-or-2 JSON objects, where the final object has the
// ApplyResult shape (PascalCase Targets with TargetResult entries).
//
// When warnings are present, apply prints {"warnings":[...]} first then
// the result. Dry-run prints the Plan instead of the ApplyResult. This
// test runs a real apply (not --dry-run) and tolerates either a leading
// warnings object or a single result object.
func TestApplyJSONShape(t *testing.T) {
	root := repoRoot(t)
	tmp := t.TempDir()
	env := []string{
		"OCA_OPENCODE_CONFIG_DIR=" + filepath.Join(tmp, "opencode"),
		"OCA_VISION_CONFIG_DIR=" + filepath.Join(tmp, "vision"),
		"OCA_CACHE_DIR=" + filepath.Join(tmp, "cache"),
	}
	stdout, stderr, err := runOCA(t, root, env, "apply", "--target", "mcp", "--config", "stack.example.toml", "--output", "json")
	if err != nil {
		t.Fatalf("apply failed: err=%v stderr=%s", err, stderr)
	}
	dec := json.NewDecoder(strings.NewReader(stdout))
	var last map[string]any
	count := 0
	for {
		var obj map[string]any
		if derr := dec.Decode(&obj); derr != nil {
			if derr == io.EOF {
				break
			}
			t.Fatalf("apply stdout is not a valid JSON object stream: %v\nstdout=%s", derr, stdout)
		}
		last = obj
		count++
	}
	if count == 0 {
		t.Fatalf("apply produced no JSON objects; stdout=%q", stdout)
	}
	if count > 2 {
		t.Fatalf("apply produced %d JSON objects; expected 1 or 2 (optional warnings + result)", count)
	}
	targets, ok := last["Targets"].([]any)
	if !ok {
		t.Fatalf("final apply object missing Targets array (ApplyResult shape); got keys=%v", mapKeys(last))
	}
	if len(targets) == 0 {
		t.Fatalf("ApplyResult.Targets is empty; expected at least one TargetResult")
	}
	first, ok := targets[0].(map[string]any)
	if !ok {
		t.Fatalf("ApplyResult.Targets[0] is not an object; got %T", targets[0])
	}
	for _, key := range []string{"Path", "Op", "Wrote"} {
		if _, ok := first[key]; !ok {
			t.Errorf("TargetResult JSON missing key %q; got keys=%v", key, mapKeys(first))
		}
	}
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
