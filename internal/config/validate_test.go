package config

import (
	"errors"
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
