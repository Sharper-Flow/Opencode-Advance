package render

import (
	"encoding/json"
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func mustParseStack(t *testing.T, src string) *cfg.Stack {
	t.Helper()
	stack, err := cfg.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return stack
}

func TestTranslateCommandsToOpencode_Basic(t *testing.T) {
	stack := mustParseStack(t, `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[commands.review]
description = "Review PR"
template = "Review $ARGUMENTS"
agent = "adv"
`)
	declared := translateCommandsToOpencode(stack.Commands)
	if declared == nil {
		t.Fatal("declared is nil")
	}
	cmdMap, ok := declared["command"]
	if !ok {
		t.Fatal("missing 'command' key in declared")
	}
	review, ok := cmdMap.(map[string]any)["review"]
	if !ok {
		t.Fatal("missing 'review' in command map")
	}
	r := review.(map[string]any)
	if r["description"] != "Review PR" {
		t.Errorf("description = %v", r["description"])
	}
	if r["template"] != "Review $ARGUMENTS" {
		t.Errorf("template = %v", r["template"])
	}
	if r["agent"] != "adv" {
		t.Errorf("agent = %v", r["agent"])
	}
	if _, has := r["model"]; has {
		t.Error("model should not be present when empty")
	}
}

func TestTranslateCommandsToOpencode_WithModelAndSubtask(t *testing.T) {
	stack := mustParseStack(t, `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[commands.test]
description = "Test"
template = "Run tests"
model = "google/gemini-3-flash"
subtask = true
`)
	declared := translateCommandsToOpencode(stack.Commands)
	cmdMap := declared["command"].(map[string]any)
	test := cmdMap["test"].(map[string]any)
	if test["model"] != "google/gemini-3-flash" {
		t.Errorf("model = %v", test["model"])
	}
	if test["subtask"] != true {
		t.Errorf("subtask = %v", test["subtask"])
	}
}

func TestTranslateCommandsToOpencode_ExtraPassthrough(t *testing.T) {
	stack := mustParseStack(t, `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[commands.x]
description = "X"
template = "Do X"
custom_flag = 42
`)
	declared := translateCommandsToOpencode(stack.Commands)
	cmdMap := declared["command"].(map[string]any)
	x := cmdMap["x"].(map[string]any)
	if x["custom_flag"] == nil {
		t.Error("custom_flag should be passed through")
	}
}

func TestTranslateCommandsToOpencode_Empty(t *testing.T) {
	declared := translateCommandsToOpencode(nil)
	if declared != nil {
		t.Errorf("nil CommandsSection should produce nil, got %v", declared)
	}
}

func TestMergeCommands(t *testing.T) {
	existing := json.RawMessage(`{"theme": "dark", "command": {"old-cmd": {"description": "Old"}}}`)
	stack := mustParseStack(t, `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[commands.new-cmd]
description = "New"
template = "Do new thing"
`)
	declared := translateCommandsToOpencode(stack.Commands)
	merged, err := MergeObject(existing, declared)
	if err != nil {
		t.Fatalf("MergeObject: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(merged, &result); err != nil {
		t.Fatalf("unmarshal merged: %v", err)
	}
	// Theme preserved
	if result["theme"] != "dark" {
		t.Error("theme should be preserved")
	}
	// Old command preserved
	cmds := result["command"].(map[string]any)
	if _, ok := cmds["old-cmd"]; !ok {
		t.Error("old-cmd should be preserved")
	}
	if _, ok := cmds["new-cmd"]; !ok {
		t.Error("new-cmd should be added")
	}
}
