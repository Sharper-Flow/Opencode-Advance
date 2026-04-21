package render

import (
	"encoding/json"
	"testing"
)

func TestTranslateFormattersToOpencode_Basic(t *testing.T) {
	stack := mustParseStack(t, `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[formatters.prettier]
command = ["npx", "prettier", "--write", "$FILE"]
extensions = [".ts", ".tsx"]
`)
	declared := translateFormattersToOpencode(stack.Formatters)
	fmtMap := declared["formatter"].(map[string]any)
	p := fmtMap["prettier"].(map[string]any)
	cmd := p["command"].([]any)
	if cmd[0] != "npx" {
		t.Errorf("command[0] = %v", cmd[0])
	}
	exts := p["extensions"].([]any)
	if exts[0] != ".ts" {
		t.Errorf("extensions[0] = %v", exts[0])
	}
}

func TestTranslateFormattersToOpencode_Disabled(t *testing.T) {
	stack := mustParseStack(t, `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[formatters.old]
disabled = true
`)
	declared := translateFormattersToOpencode(stack.Formatters)
	fmtMap := declared["formatter"].(map[string]any)
	old := fmtMap["old"].(map[string]any)
	if old["disabled"] != true {
		t.Error("disabled should be true")
	}
	// Should not include empty command/extensions
	if _, ok := old["command"]; ok {
		t.Error("command should not be present when empty")
	}
}

func TestTranslateFormattersToOpencode_WithEnvironment(t *testing.T) {
	stack := mustParseStack(t, `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[formatters.custom]
command = ["myfmt"]
extensions = [".x"]

[formatters.custom.environment]
FOO = "bar"
`)
	declared := translateFormattersToOpencode(stack.Formatters)
	fmtMap := declared["formatter"].(map[string]any)
	c := fmtMap["custom"].(map[string]any)
	env := c["environment"].(map[string]any)
	if env["FOO"] != "bar" {
		t.Errorf("environment.FOO = %v", env["FOO"])
	}
}

func TestTranslateFormattersToOpencode_ExtraPassthrough(t *testing.T) {
	stack := mustParseStack(t, `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[formatters.fmt]
command = ["fmt"]
extensions = [".txt"]
future_opt = "value"
`)
	declared := translateFormattersToOpencode(stack.Formatters)
	fmtMap := declared["formatter"].(map[string]any)
	f := fmtMap["fmt"].(map[string]any)
	if f["future_opt"] != "value" {
		t.Error("future_opt should be passed through")
	}
}

func TestTranslateFormattersToOpencode_Empty(t *testing.T) {
	declared := translateFormattersToOpencode(nil)
	if declared != nil {
		t.Errorf("nil FormattersSection should produce nil, got %v", declared)
	}
}

func TestMergeFormatters(t *testing.T) {
	existing := json.RawMessage(`{"theme": "dark", "formatter": {"user-fmt": {"command": ["user"]}}}`)
	stack := mustParseStack(t, `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[formatters.oca-fmt]
command = ["oca"]
extensions = [".go"]
`)
	declared := translateFormattersToOpencode(stack.Formatters)
	merged, err := MergeObject(existing, declared)
	if err != nil {
		t.Fatalf("MergeObject: %v", err)
	}
	var result map[string]any
	json.Unmarshal(merged, &result)
	fmts := result["formatter"].(map[string]any)
	if _, ok := fmts["user-fmt"]; !ok {
		t.Error("user-fmt should be preserved")
	}
	if _, ok := fmts["oca-fmt"]; !ok {
		t.Error("oca-fmt should be added")
	}
}
