package render

import (
	"bytes"
	"strings"
	"testing"
)

func TestRedact_NilAndEmpty(t *testing.T) {
	if got := Redact(nil); got != nil {
		t.Fatalf("Redact(nil) = %q, want nil", got)
	}
	if got := Redact([]byte{}); !bytes.Equal(got, []byte{}) {
		t.Fatalf("Redact(empty) = %q, want empty slice", got)
	}
}

func TestRedact_ReturnsFreshSlice(t *testing.T) {
	// Caller must not be able to mutate our input via the returned slice.
	in := []byte("no secrets here")
	out := Redact(in)
	if &in[0] == &out[0] {
		t.Fatal("Redact must return a fresh slice, got shared backing array")
	}
	if !bytes.Equal(in, out) {
		t.Fatalf("non-secret input changed: in=%q out=%q", in, out)
	}
}

func TestRedact_Cases(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string // exact output
		wantSub string // optional substring that must appear (for flexible cases)
	}{
		{
			name: "authorization bearer",
			in:   "Authorization: Bearer abc123xyz",
			want: "Authorization: Bearer ***REDACTED***",
		},
		{
			name: "authorization equals",
			in:   "authorization=abc123",
			want: "authorization=***REDACTED***",
		},
		{
			name: "json api_key preserves quotes and colon",
			in:   `"api_key": "sk-live-abc123"`,
			want: `"api_key": "***REDACTED***"`,
		},
		{
			name: "json password preserves quotes",
			in:   `"password": "hunter2"`,
			want: `"password": "***REDACTED***"`,
		},
		{
			name: "yaml api_token colon-space",
			in:   "api_token: sk-abc123",
			want: "api_token: ***REDACTED***",
		},
		{
			name: "uppercase env token",
			in:   "GITHUB_TOKEN=ghp_abc123xyz",
			want: "GITHUB_TOKEN=***REDACTED***",
		},
		{
			name: "lowercase env secret",
			in:   "aws_secret_access_key=AKIAIOSFODNN7EXAMPLE",
			want: "aws_secret_access_key=***REDACTED***",
		},
		{
			name: "hyphenated api-key json",
			in:   `"api-key":"xyz"`,
			want: `"api-key":"***REDACTED***"`,
		},
		{
			name: "url embedded credential",
			in:   "https://alice:s3cret@db.example.com/prod",
			want: "https://alice:***REDACTED***@db.example.com/prod",
		},
		{
			name: "http url embedded credential",
			in:   "http://user:pwd@host:5432/db",
			want: "http://user:***REDACTED***@host:5432/db",
		},
		{
			name:    "multiple secrets in one blob",
			in:      `{"GITHUB_TOKEN":"ghp_x","api_key":"sk_y"}`,
			wantSub: "***REDACTED***", // both must be scrubbed
		},
		{
			name:    "preserves non-secret keys",
			in:      `{"name":"context7","url":"https://mcp.context7.com/mcp","token":"secret"}`,
			wantSub: `"name":"context7"`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := string(Redact([]byte(tc.in)))
			if tc.want != "" && got != tc.want {
				t.Errorf("Redact(%q)\n  got:  %q\n  want: %q", tc.in, got, tc.want)
			}
			if tc.wantSub != "" && !strings.Contains(got, tc.wantSub) {
				t.Errorf("Redact(%q) = %q, want substring %q", tc.in, got, tc.wantSub)
			}
			// Sanity: nothing resembling the raw secret values should survive
			// for the one-off multi-secret case.
			if tc.name == "multiple secrets in one blob" {
				if strings.Contains(got, "ghp_x") || strings.Contains(got, "sk_y") {
					t.Errorf("raw secret leaked through redaction: %q", got)
				}
			}
		})
	}
}

func TestRedact_PreservesNonSecretSeparators(t *testing.T) {
	// Validate that surrounding JSON structure (braces, commas, other keys)
	// survives redaction intact — the bug LE-14 was that the old redactor
	// collapsed `key: value` into `key=value` and broke YAML/JSON shape.
	in := `{"GITHUB_TOKEN":"ghp_abc","other":"kept"}`
	got := string(Redact([]byte(in)))
	if !strings.Contains(got, `"other":"kept"`) {
		t.Errorf("non-secret field mangled: %q", got)
	}
	if !strings.Contains(got, `"GITHUB_TOKEN":"***REDACTED***"`) {
		t.Errorf("JSON shape lost around secret: %q", got)
	}
}

func TestRedactPlan_NilAndDeepCopy(t *testing.T) {
	if got := RedactPlan(nil); got != nil {
		t.Fatalf("RedactPlan(nil) = %+v, want nil", got)
	}

	p := &Plan{
		Source:   "stack.toml",
		LockPath: "/tmp/.oca.lock",
		Targets: []TargetOp{{
			Name:   "opencode.json",
			Path:   "/tmp/opencode.json",
			Op:     "write",
			Before: []byte(`{"GITHUB_TOKEN":"ghp_abc"}`),
			After:  []byte(`{"GITHUB_TOKEN":"ghp_xyz"}`),
			Mode:   0o600,
			Reason: "schema change",
		}},
	}
	out := RedactPlan(p)

	if out == p {
		t.Fatal("RedactPlan must return a fresh *Plan, not the same pointer")
	}
	if &out.Targets[0] == &p.Targets[0] {
		t.Fatal("RedactPlan must deep-copy Targets")
	}
	if out.Targets[0].Path != "/tmp/opencode.json" || out.Targets[0].Reason != "schema change" {
		t.Errorf("non-secret fields not preserved: %+v", out.Targets[0])
	}
	if bytes.Contains(out.Targets[0].Before, []byte("ghp_abc")) {
		t.Errorf("Before not redacted: %q", out.Targets[0].Before)
	}
	if bytes.Contains(out.Targets[0].After, []byte("ghp_xyz")) {
		t.Errorf("After not redacted: %q", out.Targets[0].After)
	}
	// Original plan must remain untouched.
	if !bytes.Contains(p.Targets[0].Before, []byte("ghp_abc")) {
		t.Errorf("input plan mutated; Before=%q", p.Targets[0].Before)
	}
}
