package config

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func mustParse(t *testing.T, src string) *Stack {
	t.Helper()
	stack, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return stack
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		src       string
		wantErr   bool   // true if Validate() should return non-nil
		wantPath  string // optional: ValidationErrors entry with this Path must exist
		wantSub   string // optional: err.Error() must contain this substring
		wantNoSub string // optional: err.Error() must NOT contain this substring
	}{
		{
			name: "meta.version required",
			src: `
[meta]
name = "x"

[mcp.servers.a]
port = 6276
command = "echo"
`,
			wantErr:  true,
			wantPath: "meta.version",
		},
		{
			name: "port out of range",
			src: `
[meta]
version = "1.0.0"

[mcp.servers.bad]
port = 9999
command = "echo"
`,
			wantErr: true,
			wantSub: "out of range",
		},
		{
			name: "port uniqueness",
			src: `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "a"

[mcp.servers.b]
port = 6276
command = "b"
`,
			wantErr: true,
			wantSub: "already used",
		},
		{
			name: "command + url conflict",
			src: `
[meta]
version = "1.0.0"

[mcp.servers.bad]
port = 6276
command = "echo"
url = "http://localhost/mcp"
`,
			wantErr: true,
			wantSub: "both command and url",
		},
		{
			name: "http transport requires /mcp suffix",
			src: `
[meta]
version = "1.0.0"

[mcp.servers.broken]
port = 6276
transport = "http"
url = "http://localhost:6276"
`,
			wantErr: true,
			wantSub: "/mcp",
		},
		{
			name: "daemon type must be named vision",
			src: `
[meta]
version = "1.0.0"

[mcp.servers.notvision]
port = 6275
type = "daemon"
`,
			wantErr: true,
			wantSub: `must be named "vision"`,
		},
		{
			name: "daemon type cannot have command",
			src: `
[meta]
version = "1.0.0"

[mcp.servers.vision]
port = 6275
type = "daemon"
command = "should-not-be-here"
`,
			wantErr: true,
			wantSub: "must not specify command",
		},
		{
			name: "two daemon servers rejected",
			src: `
[meta]
version = "1.0.0"

[mcp.servers.vision]
port = 6275
type = "daemon"

[mcp.servers.vision2]
port = 6276
type = "daemon"
`,
			wantErr: true,
			wantSub: `only one server may have type="daemon"`,
		},
		{
			name: "unknown restart_policy",
			src: `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"
restart_policy = "wat"
`,
			wantErr: true,
			wantSub: "restart_policy",
		},
		{
			name: "invalid request_timeout duration",
			src: `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"
request_timeout = "soon"
`,
			wantErr:  true,
			wantPath: "mcp.servers.a.request_timeout",
			wantSub:  "invalid duration",
		},
		{
			name: "deferred sections accepted, unknown rejected",
			src: `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[plugins.foo]
source = "https://example.com"

[foobar]
`,
			wantErr:   true,
			wantSub:   "[foobar]:",
			wantNoSub: "[plugins]:",
		},
		{
			name: "valid stack returns nil",
			src: `
[meta]
version = "1.0.0"

[mcp.servers.context7]
port = 6276
command = "npx"
args = ["-y", "@upstash/context7-mcp@latest"]
autostart = true

[mcp.servers.vision]
port = 6275
type = "daemon"
`,
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			stack := mustParse(t, tc.src)
			err := stack.Validate()

			if !tc.wantErr {
				if err != nil {
					t.Errorf("Validate() = %v, want nil", err)
				}
				return
			}

			if err == nil {
				t.Fatal("Validate() = nil, want error")
			}

			if tc.wantPath != "" {
				var ve ValidationErrors
				if !errors.As(err, &ve) {
					t.Fatalf("err = %T, want ValidationErrors", err)
				}
				found := false
				for _, e := range ve {
					if e.Path == tc.wantPath {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("no error for path %q; got %v", tc.wantPath, ve)
				}
			}

			if tc.wantSub != "" && !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("err = %v, want substring %q", err, tc.wantSub)
			}

			if tc.wantNoSub != "" && strings.Contains(err.Error(), tc.wantNoSub) {
				t.Errorf("err = %v, must NOT contain substring %q", err, tc.wantNoSub)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Phase 3.5 validation tests
// ---------------------------------------------------------------------------

func TestValidate_SkillsOrderUnique(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[skills]
order = ["lgrep", "morph", "lgrep"]
`
	stack := mustParse(t, src)
	err := stack.Validate()
	if err == nil {
		t.Fatal("expected error for duplicate skills.order entry")
	}
	if !strings.Contains(err.Error(), "duplicate") || !strings.Contains(err.Error(), "skills.order") {
		t.Errorf("err = %v, want duplicate skills.order", err)
	}
}

func TestValidate_SkillsRejectAdvPrefix(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[skills]
order = ["adv-apply-methodology", "lgrep"]
`
	stack := mustParse(t, src)
	err := stack.Validate()
	if err == nil {
		t.Fatal("expected error for adv-* reserved skill name")
	}
	if !strings.Contains(err.Error(), "reserved") || !strings.Contains(err.Error(), "adv-") {
		t.Errorf("err = %v, want reserved adv-*", err)
	}
}

func TestValidate_SkillsEmptyOrderAllowed(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[skills]
`
	stack := mustParse(t, src)
	err := stack.Validate()
	if err != nil {
		t.Errorf("empty [skills] should be valid, got %v", err)
	}
}

func TestValidate_SkillsEmptyEntryRejected(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[skills]
order = ["lgrep", ""]
`
	stack := mustParse(t, src)
	err := stack.Validate()
	if err == nil {
		t.Fatal("expected error for empty skills.order entry")
	}
	if !strings.Contains(err.Error(), "must not be empty") {
		t.Errorf("err = %v, want must not be empty", err)
	}
}

func TestValidate_FormattersRequireCommandAndExtensions(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[formatters.prettier]
command = ["npx", "prettier", "--write", "$FILE"]
`
	stack := mustParse(t, src)
	err := stack.Validate()
	if err == nil {
		t.Fatal("expected error for formatter without extensions")
	}
	if !strings.Contains(err.Error(), "extensions") || !strings.Contains(err.Error(), "formatters.prettier") {
		t.Errorf("err = %v, want formatters.prettier extensions required", err)
	}
}

func TestValidate_FormattersDisabledSkipsRequired(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[formatters.pyright]
disabled = true
`
	stack := mustParse(t, src)
	err := stack.Validate()
	if err != nil {
		t.Errorf("disabled formatter should not require command/extensions, got %v", err)
	}
}

func TestValidate_FormattersMissingCommand(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[formatters.fmt]
extensions = [".txt"]
`
	stack := mustParse(t, src)
	err := stack.Validate()
	if err == nil {
		t.Fatal("expected error for formatter without command")
	}
	if !strings.Contains(err.Error(), "command") || !strings.Contains(err.Error(), "formatters.fmt") {
		t.Errorf("err = %v, want formatters.fmt command required", err)
	}
}

func TestValidate_CommandsRequireDescriptionAndTemplate(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[commands.broken]
description = "Has desc but no template"
`
	stack := mustParse(t, src)
	err := stack.Validate()
	if err == nil {
		t.Fatal("expected error for command without template")
	}
	if !strings.Contains(err.Error(), "template") || !strings.Contains(err.Error(), "commands.broken") {
		t.Errorf("err = %v, want commands.broken template required", err)
	}
}

func TestValidate_CommandsMissingDescription(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[commands.broken]
template = "Has template but no description"
`
	stack := mustParse(t, src)
	err := stack.Validate()
	if err == nil {
		t.Fatal("expected error for command without description")
	}
	if !strings.Contains(err.Error(), "description") {
		t.Errorf("err = %v, want description required", err)
	}
}

func TestValidate_CommandsValidWithOptionalFields(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[commands.hello]
description = "Hello"
template = "Say hello"
agent = "adv"
model = "google/gemini-3-flash"
`
	stack := mustParse(t, src)
	err := stack.Validate()
	if err != nil {
		t.Errorf("valid command with optional fields should pass, got %v", err)
	}
}

func TestValidate_OpenCodeShareEnum(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[opencode]
share = "invalid"
`
	stack := mustParse(t, src)
	err := stack.Validate()
	if err == nil {
		t.Fatal("expected error for invalid share value")
	}
	if !strings.Contains(err.Error(), "share") || !strings.Contains(err.Error(), "invalid") {
		t.Errorf("err = %v, want share invalid", err)
	}
}

func TestValidate_OpenCodeShareValidValues(t *testing.T) {
	for _, val := range []string{"manual", "auto", "disabled", ""} {
		t.Run(val, func(t *testing.T) {
			src := fmt.Sprintf(`
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[opencode]
share = "%s"
`, val)
			stack := mustParse(t, src)
			err := stack.Validate()
			if err != nil {
				t.Errorf("share=%q should be valid, got %v", val, err)
			}
		})
	}
}

func TestValidate_OpenCodeAutoupdateValidValues(t *testing.T) {
	// true, false, "notify" should all be valid
	for name, src := range map[string]string{
		"bool_true": `
[meta]
version = "1.0.0"
[mcp.servers.a]
port = 6276
command = "echo"
[opencode]
autoupdate = true
`,
		"bool_false": `
[meta]
version = "1.0.0"
[mcp.servers.a]
port = 6276
command = "echo"
[opencode]
autoupdate = false
`,
		"string_notify": `
[meta]
version = "1.0.0"
[mcp.servers.a]
port = 6276
command = "echo"
[opencode]
autoupdate = "notify"
`,
	} {
		t.Run(name, func(t *testing.T) {
			stack := mustParse(t, src)
			err := stack.Validate()
			if err != nil {
				t.Errorf("autoupdate %s should be valid, got %v", name, err)
			}
		})
	}
}

func TestValidate_OpenCodeAutoupdateInvalidString(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[opencode]
autoupdate = "bad_value"
`
	stack := mustParse(t, src)
	err := stack.Validate()
	if err == nil {
		t.Fatal("expected error for invalid autoupdate string")
	}
	if !strings.Contains(err.Error(), "autoupdate") {
		t.Errorf("err = %v, want autoupdate error", err)
	}
}

func TestValidate_Phase35AllValid(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[skills]
order = ["lgrep", "morph"]

[formatters.prettier]
command = ["npx", "prettier", "--write", "$FILE"]
extensions = [".ts"]

[commands.review]
description = "Review"
template = "Review $ARGUMENTS"
agent = "adv"

[opencode]
theme = "obsidian"
default_agent = "adv"
share = "disabled"
autoupdate = false
`
	stack := mustParse(t, src)
	err := stack.Validate()
	if err != nil {
		t.Errorf("fully valid Phase 3.5 stack should pass, got %v", err)
	}
}
