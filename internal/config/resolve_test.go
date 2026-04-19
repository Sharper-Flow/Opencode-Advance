package config

import (
	"os"
	"strings"
	"testing"
)

func TestResolve_HomeExpansion(t *testing.T) {
	home, _ := os.UserHomeDir()
	if home == "" {
		t.Skip("no home dir")
	}
	stack := mustParse(t, `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "~/bin/tool"
env_file = "~/.secrets/a.env"
args = ["--home=$HOME/config"]
`)
	stack.Resolve()
	srv := stack.MCP.Servers["a"]
	if !strings.HasPrefix(srv.Command, home) {
		t.Errorf("command ~/-expansion failed: %q", srv.Command)
	}
	if !strings.HasPrefix(srv.EnvFile, home) {
		t.Errorf("env_file ~/-expansion failed: %q", srv.EnvFile)
	}
	if !strings.Contains(srv.Args[0], home) {
		t.Errorf("args $HOME expansion failed: %q", srv.Args[0])
	}
}

func TestResolve_EnvVarSubstitution(t *testing.T) {
	t.Setenv("OCA_TEST_VAR", "hello")
	stack := mustParse(t, `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"
args = ["${OCA_TEST_VAR}", "${OCA_MISSING:-fallback}"]
`)
	stack.Resolve()
	args := stack.MCP.Servers["a"].Args
	if args[0] != "hello" {
		t.Errorf("args[0] = %q, want hello", args[0])
	}
	if args[1] != "fallback" {
		t.Errorf("args[1] = %q, want fallback", args[1])
	}
}

func TestResolve_PreservesNativeOpenCodeTokens(t *testing.T) {
	stack := mustParse(t, `
[meta]
version = "1.0.0"

[mcp.servers.kagi]
port = 6279
command = "uvx"

[mcp.servers.kagi.env]
KAGI_API_KEY = "{env:KAGI_API_KEY}"
OTHER = "{file:/etc/secrets/other}"
`)
	stack.Resolve()
	env := stack.MCP.Servers["kagi"].Env
	if env["KAGI_API_KEY"] != "{env:KAGI_API_KEY}" {
		t.Errorf("native {env:...} token was modified: %q", env["KAGI_API_KEY"])
	}
	if env["OTHER"] != "{file:/etc/secrets/other}" {
		t.Errorf("native {file:...} token was modified: %q", env["OTHER"])
	}
}

func TestLoad_CollectsEnvFileWarnings(t *testing.T) {
	dir := t.TempDir()
	stackPath := dir + "/stack.toml"
	content := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"
env_file = "` + dir + `/nope.env"
`
	if err := os.WriteFile(stackPath, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	stack, err := Load(stackPath)
	if err != nil {
		t.Fatalf("Load should succeed with missing env_file (soft-warn); got: %v", err)
	}
	if len(stack.Warnings) == 0 {
		t.Fatal("expected Warnings populated for missing env_file")
	}
	found := false
	for _, w := range stack.Warnings {
		if strings.Contains(w.Path, "env_file") {
			found = true
		}
	}
	if !found {
		t.Errorf("no env_file warning among: %v", stack.Warnings)
	}
}
