package render

import (
	"encoding/json"
	"testing"
)

func TestTranslateOpenCodeToOpencode_Basic(t *testing.T) {
	stack := mustParseStack(t, `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[opencode]
theme = "obsidian"
default_agent = "adv"
share = "disabled"
snapshot = true
autoupdate = false
`)
	declared := translateOpenCodeToOpencode(stack.OpenCode)
	if declared["theme"] != "obsidian" {
		t.Errorf("theme = %v", declared["theme"])
	}
	if declared["default_agent"] != "adv" {
		t.Errorf("default_agent = %v", declared["default_agent"])
	}
	if declared["share"] != "disabled" {
		t.Errorf("share = %v", declared["share"])
	}
	if declared["snapshot"] != true {
		t.Errorf("snapshot = %v", declared["snapshot"])
	}
	if declared["autoupdate"] != false {
		t.Errorf("autoupdate = %v", declared["autoupdate"])
	}
}

func TestTranslateOpenCodeToOpencode_AutoupdateNotify(t *testing.T) {
	stack := mustParseStack(t, `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[opencode]
autoupdate = "notify"
`)
	declared := translateOpenCodeToOpencode(stack.OpenCode)
	if declared["autoupdate"] != "notify" {
		t.Errorf("autoupdate = %v, want notify", declared["autoupdate"])
	}
}

func TestTranslateOpenCodeToOpencode_Compaction(t *testing.T) {
	stack := mustParseStack(t, `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[opencode]
theme = "obsidian"

[opencode.compaction]
auto = true
prune = true
reserved = 10000
`)
	declared := translateOpenCodeToOpencode(stack.OpenCode)
	comp := declared["compaction"].(map[string]any)
	if comp["auto"] != true {
		t.Errorf("compaction.auto = %v", comp["auto"])
	}
	if comp["prune"] != true {
		t.Errorf("compaction.prune = %v", comp["prune"])
	}
	if comp["reserved"] != int64(10000) && comp["reserved"] != 10000 {
		t.Errorf("compaction.reserved = %v (%T)", comp["reserved"], comp["reserved"])
	}
}

func TestTranslateOpenCodeToOpencode_ProviderLists(t *testing.T) {
	stack := mustParseStack(t, `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[opencode.disabled_providers]
list = ["ollama"]

[opencode.enabled_providers]
list = ["google", "openai"]
`)
	declared := translateOpenCodeToOpencode(stack.OpenCode)
	dp := declared["disabled_providers"].(map[string]any)
	dpList := dp["list"].([]any)
	if dpList[0] != "ollama" {
		t.Errorf("disabled_providers.list[0] = %v", dpList[0])
	}
	ep := declared["enabled_providers"].(map[string]any)
	epList := ep["list"].([]any)
	if len(epList) != 2 {
		t.Errorf("enabled_providers.list len = %d, want 2", len(epList))
	}
}

func TestTranslateOpenCodeToOpencode_ExtraPassthrough(t *testing.T) {
	stack := mustParseStack(t, `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[opencode]
theme = "obsidian"
future_toggle = "value"
`)
	declared := translateOpenCodeToOpencode(stack.OpenCode)
	if declared["future_toggle"] != "value" {
		t.Error("future_toggle should be passed through")
	}
}

func TestTranslateOpenCodeToOpencode_Empty(t *testing.T) {
	stack := mustParseStack(t, `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"
`)
	declared := translateOpenCodeToOpencode(stack.OpenCode)
	if len(declared) != 0 {
		t.Errorf("empty OpenCode should produce empty map, got %v", declared)
	}
}

func TestMergeToggles(t *testing.T) {
	existing := json.RawMessage(`{"theme": "dark", "custom_key": "preserved"}`)
	stack := mustParseStack(t, `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[opencode]
theme = "obsidian"
default_agent = "adv"
share = "disabled"
`)
	declared := translateOpenCodeToOpencode(stack.OpenCode)
	merged, err := MergeObject(existing, declared)
	if err != nil {
		t.Fatalf("MergeObject: %v", err)
	}
	var result map[string]any
	json.Unmarshal(merged, &result)
	if result["theme"] != "obsidian" {
		t.Errorf("theme should be updated to obsidian, got %v", result["theme"])
	}
	if result["custom_key"] != "preserved" {
		t.Error("custom_key should be preserved")
	}
	if result["default_agent"] != "adv" {
		t.Error("default_agent should be added")
	}
}
