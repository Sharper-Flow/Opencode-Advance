package config

import (
	"os"
	"path/filepath"
	"testing"
)

const minimalTOML = `
[meta]
version = "1.0.0"
name = "test"

[mcp.servers.context7]
port = 6276
command = "npx"
args = ["-y", "@upstash/context7-mcp@latest"]
timeout = 10000
autostart = true
source = "https://github.com/upstash/context7"
description = "Library docs"
`

func TestParse_MinimalStackParsesTypedSections(t *testing.T) {
	stack, err := Parse([]byte(minimalTOML))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if stack.Meta.Version != "1.0.0" {
		t.Errorf("Meta.Version = %q, want 1.0.0", stack.Meta.Version)
	}
	srv, ok := stack.MCP.Servers["context7"]
	if !ok {
		t.Fatalf("context7 missing from servers")
	}
	if srv.Port != 6276 {
		t.Errorf("port = %d, want 6276", srv.Port)
	}
	if srv.Command != "npx" {
		t.Errorf("command = %q, want npx", srv.Command)
	}
	if len(srv.Args) != 2 {
		t.Errorf("args len = %d, want 2", len(srv.Args))
	}
	if srv.Source != "https://github.com/upstash/context7" {
		t.Errorf("source not preserved: %q", srv.Source)
	}
}

func TestParse_DeferredSectionsAreCollected(t *testing.T) {
	toml := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[plugins.advance]
source = "https://github.com/Sharper-Flow/Advance.git"
ref = "trunk"

[providers.openai]

[providers.openai.models."gpt-5.2"]
name = "GPT 5.2"
context = 272000

[agents]
build = "anthropic/claude-opus-4-6"
`
	stack, err := Parse([]byte(toml))
	if err != nil {
		t.Fatalf("Parse failed on stack with deferred sections: %v", err)
	}
	// providers is now a typed section (Phase 3 graduated).
	if _, ok := stack.DeferredSections["providers"]; ok {
		t.Errorf("providers should be typed, not deferred")
	}
	// agents remains deferred (out of Phase 3 scope).
	if _, ok := stack.DeferredSections["agents"]; !ok {
		t.Errorf("DeferredSections missing agents; got keys: %v", deferredKeys(stack))
	}
	// Typed sections still populated.
	if stack.Meta.Version != "1.0.0" {
		t.Errorf("Meta.Version lost")
	}
	if _, ok := stack.MCP.Servers["a"]; !ok {
		t.Errorf("mcp.servers.a lost")
	}
	// plugins is now a typed section.
	if _, ok := stack.Plugins["advance"]; !ok {
		t.Errorf("plugins.advance not typed")
	}
	if stack.Plugins["advance"].Source != "https://github.com/Sharper-Flow/Advance.git" {
		t.Errorf("plugins.advance source not decoded")
	}
}

// ---------------------------------------------------------------------------
// Phase 3.5 typed section parse tests
// ---------------------------------------------------------------------------

func TestParse_SkillsSectionTyped(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[skills]
order = ["lgrep", "morph", "prioritizer"]
`
	stack := mustParse(t, src)

	// Skills should be typed, not deferred.
	if _, ok := stack.DeferredSections["skills"]; ok {
		t.Fatal("skills should be typed, not deferred")
	}
	if stack.Skills.Order == nil {
		t.Fatal("Skills.Order is nil")
	}
	if got := len(stack.Skills.Order); got != 3 {
		t.Fatalf("Skills.Order len = %d, want 3", got)
	}
	want := []string{"lgrep", "morph", "prioritizer"}
	for i, s := range stack.Skills.Order {
		if s != want[i] {
			t.Errorf("Skills.Order[%d] = %q, want %q", i, s, want[i])
		}
	}
}

func TestParse_SkillsSectionEmpty(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[skills]
`
	stack := mustParse(t, src)
	// Empty [skills] is valid — typed but with no order.
	if _, ok := stack.DeferredSections["skills"]; ok {
		t.Fatal("skills should be typed, not deferred")
	}
	if stack.Skills.Order != nil {
		t.Errorf("Skills.Order should be nil for empty [skills], got %v", stack.Skills.Order)
	}
}

func TestParse_SkillsSectionNotPresent(t *testing.T) {
	src := minimalTOML
	stack := mustParse(t, src)
	if stack.Skills.Order != nil {
		t.Errorf("Skills.Order should be nil when [skills] absent, got %v", stack.Skills.Order)
	}
}

func TestParse_FormattersSectionTyped(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[formatters.prettier]
command = ["npx", "prettier", "--write", "$FILE"]
extensions = [".ts", ".tsx"]

[formatters.ruff]
command = ["ruff", "format", "$FILE"]
extensions = [".py"]

[formatters.disabled_one]
disabled = true
`
	stack := mustParse(t, src)

	if _, ok := stack.DeferredSections["formatters"]; ok {
		t.Fatal("formatters should be typed, not deferred")
	}
	if len(stack.Formatters) != 3 {
		t.Fatalf("Formatters len = %d, want 3", len(stack.Formatters))
	}

	p, ok := stack.Formatters["prettier"]
	if !ok {
		t.Fatal("formatters.prettier missing")
	}
	if len(p.Command) != 4 || p.Command[0] != "npx" {
		t.Errorf("prettier.Command = %v", p.Command)
	}
	if len(p.Extensions) != 2 || p.Extensions[0] != ".ts" {
		t.Errorf("prettier.Extensions = %v", p.Extensions)
	}
	if p.Disabled {
		t.Error("prettier should not be disabled")
	}

	d, ok := stack.Formatters["disabled_one"]
	if !ok {
		t.Fatal("formatters.disabled_one missing")
	}
	if !d.Disabled {
		t.Error("disabled_one should be disabled")
	}
}

func TestParse_FormattersWithEnvironment(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[formatters.custom]
command = ["myfmt", "$FILE"]
extensions = [".xyz"]

[formatters.custom.environment]
FOO = "bar"
`
	stack := mustParse(t, src)
	c, ok := stack.Formatters["custom"]
	if !ok {
		t.Fatal("formatters.custom missing")
	}
	if c.Environment == nil || c.Environment["FOO"] != "bar" {
		t.Errorf("custom.Environment = %v, want FOO=bar", c.Environment)
	}
}

func TestParse_FormattersExtraPassthrough(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[formatters.fmt]
command = ["fmt", "$FILE"]
extensions = [".txt"]
future_field = "preserved"
`
	stack := mustParse(t, src)
	f, ok := stack.Formatters["fmt"]
	if !ok {
		t.Fatal("formatters.fmt missing")
	}
	if f.Extra == nil || f.Extra["future_field"] != "preserved" {
		t.Errorf("formatters.fmt.Extra = %v, want future_field=preserved", f.Extra)
	}
}

func TestParse_CommandsSectionTyped(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[commands.review-pr]
description = "Review the current PR"
template = "Review the PR at $ARGUMENTS"
agent = "adv"

[commands.test-coverage]
description = "Run tests with coverage"
template = "Run the test suite"
agent = "build"
model = "google/gemini-3-flash-preview"
`
	stack := mustParse(t, src)

	if _, ok := stack.DeferredSections["commands"]; ok {
		t.Fatal("commands should be typed, not deferred")
	}
	if len(stack.Commands) != 2 {
		t.Fatalf("Commands len = %d, want 2", len(stack.Commands))
	}

	rp, ok := stack.Commands["review-pr"]
	if !ok {
		t.Fatal("commands.review-pr missing")
	}
	if rp.Description != "Review the current PR" {
		t.Errorf("review-pr.Description = %q", rp.Description)
	}
	if rp.Template != "Review the PR at $ARGUMENTS" {
		t.Errorf("review-pr.Template = %q", rp.Template)
	}
	if rp.Agent != "adv" {
		t.Errorf("review-pr.Agent = %q", rp.Agent)
	}
	if rp.Model != "" {
		t.Errorf("review-pr.Model should be empty, got %q", rp.Model)
	}

	tc, ok := stack.Commands["test-coverage"]
	if !ok {
		t.Fatal("commands.test-coverage missing")
	}
	if tc.Model != "google/gemini-3-flash-preview" {
		t.Errorf("test-coverage.Model = %q", tc.Model)
	}
}

func TestParse_CommandWithSubtask(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[commands.multi]
description = "Multi-step command"
template = "Do things"
subtask = true
`
	stack := mustParse(t, src)
	m, ok := stack.Commands["multi"]
	if !ok {
		t.Fatal("commands.multi missing")
	}
	if m.Subtask == nil || !*m.Subtask {
		t.Error("multi.Subtask should be true")
	}
}

func TestParse_CommandsExtraPassthrough(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[commands.cmd]
description = "desc"
template = "tmpl"
custom_flag = 42
`
	stack := mustParse(t, src)
	c, ok := stack.Commands["cmd"]
	if !ok {
		t.Fatal("commands.cmd missing")
	}
	if c.Extra == nil {
		t.Fatal("commands.cmd.Extra is nil")
	}
	if v, ok := c.Extra["custom_flag"]; !ok {
		t.Error("custom_flag missing from Extra")
	} else {
		// TOML integers decode as int64
		if n, ok := v.(int64); !ok || n != 42 {
			t.Errorf("custom_flag = %v (%T), want int64(42)", v, v)
		}
	}
}

func TestParse_OpenCodeSectionTyped(t *testing.T) {
	src := `
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
`
	stack := mustParse(t, src)

	if _, ok := stack.DeferredSections["opencode"]; ok {
		t.Fatal("opencode should be typed, not deferred")
	}
	oc := stack.OpenCode
	if oc.Theme != "obsidian" {
		t.Errorf("OpenCode.Theme = %q, want obsidian", oc.Theme)
	}
	if oc.DefaultAgent != "adv" {
		t.Errorf("OpenCode.DefaultAgent = %q, want adv", oc.DefaultAgent)
	}
	if oc.Share != "disabled" {
		t.Errorf("OpenCode.Share = %q, want disabled", oc.Share)
	}
	if oc.Snapshot == nil || !*oc.Snapshot {
		t.Error("OpenCode.Snapshot should be true")
	}
	// autoupdate = false should be captured
	if oc.Autoupdate == nil {
		t.Fatal("OpenCode.Autoupdate is nil")
	}
	if oc.Autoupdate.Bool == nil || *oc.Autoupdate.Bool != false {
		t.Errorf("OpenCode.Autoupdate.Bool = %v, want false", oc.Autoupdate.Bool)
	}
}

func TestParse_OpenCodeAutoupdateString(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[opencode]
autoupdate = "notify"
`
	stack := mustParse(t, src)
	if stack.OpenCode.Autoupdate == nil {
		t.Fatal("Autoupdate is nil")
	}
	if stack.OpenCode.Autoupdate.Bool != nil {
		t.Errorf("Autoupdate.Bool should be nil for string value, got %v", stack.OpenCode.Autoupdate.Bool)
	}
	if stack.OpenCode.Autoupdate.Str == nil || *stack.OpenCode.Autoupdate.Str != "notify" {
		t.Errorf("Autoupdate.Str = %v, want notify", stack.OpenCode.Autoupdate.Str)
	}
}

func TestParse_OpenCodeCompaction(t *testing.T) {
	src := `
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
`
	stack := mustParse(t, src)
	if stack.OpenCode.Compaction == nil {
		t.Fatal("OpenCode.Compaction is nil")
	}
	if stack.OpenCode.Compaction.Auto == nil || !*stack.OpenCode.Compaction.Auto {
		t.Error("Compaction.Auto should be true")
	}
	if stack.OpenCode.Compaction.Prune == nil || !*stack.OpenCode.Compaction.Prune {
		t.Error("Compaction.Prune should be true")
	}
	if stack.OpenCode.Compaction.Reserved != 10000 {
		t.Errorf("Compaction.Reserved = %d, want 10000", stack.OpenCode.Compaction.Reserved)
	}
}

func TestParse_OpenCodeProviderLists(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[opencode.disabled_providers]
list = ["ollama"]

[opencode.enabled_providers]
list = ["google", "openai"]
`
	stack := mustParse(t, src)
	if stack.OpenCode.DisabledProviders == nil {
		t.Fatal("DisabledProviders is nil")
	}
	if len(stack.OpenCode.DisabledProviders.List) != 1 || stack.OpenCode.DisabledProviders.List[0] != "ollama" {
		t.Errorf("DisabledProviders.List = %v", stack.OpenCode.DisabledProviders.List)
	}
	if stack.OpenCode.EnabledProviders == nil {
		t.Fatal("EnabledProviders is nil")
	}
	if len(stack.OpenCode.EnabledProviders.List) != 2 {
		t.Errorf("EnabledProviders.List = %v", stack.OpenCode.EnabledProviders.List)
	}
}

func TestParse_OpenCodeExtraPassthrough(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[opencode]
theme = "obsidian"
future_toggle = "some_value"
`
	stack := mustParse(t, src)
	if stack.OpenCode.Extra == nil {
		t.Fatal("OpenCode.Extra is nil")
	}
	if v, ok := stack.OpenCode.Extra["future_toggle"]; !ok || v != "some_value" {
		t.Errorf("OpenCode.Extra[future_toggle] = %v, want some_value", v)
	}
}

func TestParse_Phase35SectionsNotDeferred(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[skills]
order = ["lgrep"]

[formatters.fmt]
command = ["fmt"]
extensions = [".txt"]

[commands.hello]
description = "Hello"
template = "Say hello"

[opencode]
theme = "obsidian"
`
	stack := mustParse(t, src)
	for _, key := range []string{"skills", "formatters", "commands", "opencode"} {
		if _, ok := stack.DeferredSections[key]; ok {
			t.Errorf("%q should be typed, not deferred", key)
		}
	}
}

func TestParse_MalformedTOMLReturnsParseError(t *testing.T) {
	_, err := Parse([]byte("this is not valid toml [[[\n"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var pe *ParseError
	// errors.As via unwrap
	if pe = asParseError(err); pe == nil {
		t.Errorf("err = %T %v, want *ParseError", err, err)
	}
}

func TestParseFile_ReadsDiskFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stack.toml")
	if err := os.WriteFile(path, []byte(minimalTOML), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	stack, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if stack.Meta.Version != "1.0.0" {
		t.Errorf("Meta.Version mismatched after ParseFile")
	}
}

func TestParseFile_MissingFileReturnsParseError(t *testing.T) {
	_, err := ParseFile(filepath.Join(t.TempDir(), "does-not-exist.toml"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if asParseError(err) == nil {
		t.Errorf("err = %T, want *ParseError", err)
	}
}

// asParseError returns the inner *ParseError if present, else nil.
func asParseError(err error) *ParseError {
	for {
		if err == nil {
			return nil
		}
		if pe, ok := err.(*ParseError); ok {
			return pe
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return nil
		}
		err = u.Unwrap()
	}
}

func deferredKeys(s *Stack) []string {
	out := make([]string, 0, len(s.DeferredSections))
	for k := range s.DeferredSections {
		out = append(out, k)
	}
	return out
}
