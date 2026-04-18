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

func TestValidate_MetaVersionRequired(t *testing.T) {
	stack := mustParse(t, `
[meta]
name = "x"

[mcp.servers.a]
port = 6276
command = "echo"
`)
	err := stack.Validate()
	if err == nil {
		t.Fatal("expected error for missing meta.version")
	}
	var ve ValidationErrors
	if !errors.As(err, &ve) {
		t.Fatalf("err = %T, want ValidationErrors", err)
	}
	found := false
	for _, e := range ve {
		if e.Path == "meta.version" {
			found = true
		}
	}
	if !found {
		t.Errorf("no error for meta.version; got %v", ve)
	}
}

func TestValidate_PortOutOfRange(t *testing.T) {
	stack := mustParse(t, `
[meta]
version = "1.0.0"

[mcp.servers.bad]
port = 9999
command = "echo"
`)
	err := stack.Validate()
	if err == nil {
		t.Fatal("expected error for port out of range")
	}
	if !strings.Contains(err.Error(), "out of range") {
		t.Errorf("expected 'out of range', got: %v", err)
	}
}

func TestValidate_PortUniqueness(t *testing.T) {
	stack := mustParse(t, `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "a"

[mcp.servers.b]
port = 6276
command = "b"
`)
	err := stack.Validate()
	if err == nil {
		t.Fatal("expected duplicate port error")
	}
	if !strings.Contains(err.Error(), "already used") {
		t.Errorf("expected duplicate-port message, got: %v", err)
	}
}

func TestValidate_CommandUrlConflict(t *testing.T) {
	stack := mustParse(t, `
[meta]
version = "1.0.0"

[mcp.servers.bad]
port = 6276
command = "echo"
url = "http://localhost/mcp"
`)
	err := stack.Validate()
	if err == nil {
		t.Fatal("expected command+url conflict error")
	}
	if !strings.Contains(err.Error(), "both command and url") {
		t.Errorf("expected conflict message, got: %v", err)
	}
}

func TestValidate_HTTPTransportRequiresMcpSuffix(t *testing.T) {
	stack := mustParse(t, `
[meta]
version = "1.0.0"

[mcp.servers.broken]
port = 6276
transport = "http"
url = "http://localhost:6276"
`)
	err := stack.Validate()
	if err == nil {
		t.Fatal("expected http /mcp suffix error")
	}
	if !strings.Contains(err.Error(), "/mcp") {
		t.Errorf("expected /mcp suffix message, got: %v", err)
	}
}

func TestValidate_DaemonTypeSingletonAndNamedVision(t *testing.T) {
	stack := mustParse(t, `
[meta]
version = "1.0.0"

[mcp.servers.notvision]
port = 6275
type = "daemon"
`)
	err := stack.Validate()
	if err == nil {
		t.Fatal("expected daemon-not-named-vision error")
	}
	if !strings.Contains(err.Error(), "must be named \"vision\"") {
		t.Errorf("expected daemon naming rule, got: %v", err)
	}
}

func TestValidate_DaemonTypeCannotHaveCommand(t *testing.T) {
	stack := mustParse(t, `
[meta]
version = "1.0.0"

[mcp.servers.vision]
port = 6275
type = "daemon"
command = "should-not-be-here"
`)
	err := stack.Validate()
	if err == nil {
		t.Fatal("expected daemon-cannot-have-command error")
	}
	if !strings.Contains(err.Error(), "must not specify command") {
		t.Errorf("expected daemon no-command rule, got: %v", err)
	}
}

func TestValidate_TwoDaemonServersRejected(t *testing.T) {
	stack := mustParse(t, `
[meta]
version = "1.0.0"

[mcp.servers.vision]
port = 6275
type = "daemon"

[mcp.servers.vision2]
port = 6276
type = "daemon"
`)
	err := stack.Validate()
	if err == nil {
		t.Fatal("expected multiple-daemon error")
	}
	if !strings.Contains(err.Error(), "only one server may have type=\"daemon\"") {
		t.Errorf("expected singleton rule, got: %v", err)
	}
}

func TestValidate_UnknownRestartPolicy(t *testing.T) {
	stack := mustParse(t, `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"
restart_policy = "wat"
`)
	err := stack.Validate()
	if err == nil {
		t.Fatal("expected unknown restart_policy error")
	}
	if !strings.Contains(err.Error(), "restart_policy") {
		t.Errorf("got: %v", err)
	}
}

func TestValidate_DeferredSectionsAccepted_UnknownRejected(t *testing.T) {
	stack := mustParse(t, `
[meta]
version = "1.0.0"

[mcp.servers.a]
port = 6276
command = "echo"

[plugins.foo]
source = "https://example.com"

[foobar]
`)
	err := stack.Validate()
	if err == nil {
		t.Fatal("expected unknown-section error for [foobar]")
	}
	// Error lines start with "  [<section>]:" — assert [foobar] is present
	// and [plugins] is NOT (the token "plugins" legitimately appears in the
	// "allowed sections" hint, so we need structural matching).
	if !strings.Contains(err.Error(), "[foobar]:") {
		t.Errorf("expected [foobar]: in error, got: %v", err)
	}
	if strings.Contains(err.Error(), "[plugins]:") {
		t.Errorf("plugins (known-deferred) should not error, got: %v", err)
	}
}

func TestValidate_ValidStackReturnsNil(t *testing.T) {
	stack := mustParse(t, `
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
`)
	if err := stack.Validate(); err != nil {
		t.Errorf("expected nil for valid stack, got: %v", err)
	}
}
