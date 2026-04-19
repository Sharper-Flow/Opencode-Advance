package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cfg "github.com/Sharper-Flow/Opencode-Advance/internal/config"
)

func testStack() *cfg.Stack {
	t := true
	return &cfg.Stack{MCP: cfg.MCPSection{Servers: map[string]cfg.Server{
		"vision":   {Port: 6275, Type: "daemon", Required: true},
		"context7": {Port: 6276, Command: "npx", Enabled: &t},
		"kagi":     {Port: 6279, Command: "uvx", Required: true},
	}}}
}

func TestCheckMCP_VersionAndServerStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/version":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":"dev","api":{"v1_servers":true}}`))
		case "/v1/servers":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"servers":[{"name":"vision","state":"running","port":6275,"required":true},{"name":"context7","state":"running","port":6276},{"name":"kagi","state":"failed","port":6279,"required":true,"last_error":"boom"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	checks, err := CheckMCP(context.Background(), testStack(), Options{VisionAdminURL: ts.URL})
	if err != nil {
		t.Fatal(err)
	}
	if len(checks) < 4 {
		t.Fatalf("checks too small: %#v", checks)
	}
	if !HasFailures(checks) {
		t.Fatalf("expected required kagi failure, got %s", Summary(checks))
	}
	seenVersion := false
	for _, c := range checks {
		if c.Name == "vision.version" && c.Status == StatusPass {
			seenVersion = true
		}
	}
	if !seenVersion {
		t.Fatalf("expected passing vision.version check, got %#v", checks)
	}
}

func TestCheckMCP_RedactsLastErrorSecrets(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/version":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":"dev","api":{"v1_servers":true}}`))
		case "/v1/servers":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"servers":[{"name":"vision","state":"running","port":6275,"required":true},{"name":"kagi","state":"failed","port":6279,"required":true,"last_error":"spawn failed: API_KEY=sk-abcdef123456 exec denied"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	checks, err := CheckMCP(context.Background(), testStack(), Options{VisionAdminURL: ts.URL})
	if err != nil {
		t.Fatal(err)
	}
	var kagi *Check
	for i := range checks {
		if checks[i].Name == "mcp.kagi" {
			kagi = &checks[i]
			break
		}
	}
	if kagi == nil {
		t.Fatalf("expected mcp.kagi check, got %#v", checks)
	}
	if strings.Contains(kagi.Message, "sk-abcdef123456") {
		t.Fatalf("expected secret to be redacted, got %q", kagi.Message)
	}
	if !strings.Contains(kagi.Message, "REDACTED") {
		t.Fatalf("expected REDACTED marker in message, got %q", kagi.Message)
	}
}

func TestCheckMCP_WarnsOnNonAbsoluteCommand(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/version":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":"dev","api":{"v1_servers":true}}`))
		case "/v1/servers":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"servers":[{"name":"vision","state":"running","port":6275,"required":true},{"name":"context7","state":"running","port":6276},{"name":"kagi","state":"running","port":6279,"required":true}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	checks, err := CheckMCP(context.Background(), testStack(), Options{VisionAdminURL: ts.URL})
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]Check{}
	for _, c := range checks {
		names[c.Name] = c
	}
	for _, want := range []string{"mcp.context7.command_path", "mcp.kagi.command_path"} {
		c, ok := names[want]
		if !ok {
			t.Fatalf("expected warn check %s, got %#v", want, checks)
		}
		if c.Status != StatusWarn {
			t.Fatalf("%s: expected warn status, got %s", want, c.Status)
		}
		if !strings.Contains(c.Hint, "absolute path") {
			t.Fatalf("%s: expected absolute path hint, got %q", want, c.Hint)
		}
	}
}

func TestCheckMCP_IncompatibleVersionWarns(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/version" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":"old","api":{"v1_servers":false}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	checks, err := CheckMCP(context.Background(), testStack(), Options{VisionAdminURL: ts.URL})
	if err != nil {
		t.Fatal(err)
	}
	if !HasWarnings(checks) {
		t.Fatalf("expected warning, got %#v", checks)
	}
	if !strings.Contains(checks[0].Hint, "upgrade Vision") {
		t.Fatalf("expected upgrade hint, got %#v", checks[0])
	}
}
