package tests

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestDoctorMCP_WithMockVision(t *testing.T) {
	root := repoRoot(t)
	ln, err := net.Listen("tcp", "127.0.0.1:6275")
	if err != nil {
		t.Skipf("port 6275 unavailable for e2e doctor test: %v", err)
	}
	defer ln.Close()
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/version":
			w.Write([]byte(`{"version":"dev","api":{"v1_servers":true}}`))
		case "/v1/servers":
			w.Write([]byte(`{"servers":[{"name":"vision","state":"running","port":6275,"required":true},{"name":"context7","state":"running","port":6276},{"name":"sentry","state":"failed","port":6289,"required":true,"last_error":"boom"}]}`))
		default:
			http.NotFound(w, r)
		}
	})}
	go srv.Serve(ln)
	defer srv.Shutdown(context.Background())

	base := t.TempDir()
	secretDir := filepath.Join(base, "secrets")
	if err := os.MkdirAll(secretDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"sentry.env"} {
		if err := os.WriteFile(filepath.Join(secretDir, f), []byte("X=1\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	stackPath := filepath.Join(base, "stack.toml")
	content := `[meta]
version = "1.0.0"

[mcp.servers.vision]
port = 6275
type = "daemon"
required = true

[mcp.servers.context7]
port = 6276
command = "npx"

[mcp.servers.sentry]
port = 6289
command = "npx"
required = true
env_file = "` + filepath.Join(secretDir, "sentry.env") + `"
`
	if err := os.WriteFile(stackPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err := runOCA(t, root, nil, "doctor", "--scope", "mcp", "--output", "json", "--config", stackPath)
	if stdout == "" {
		t.Fatalf("expected doctor stdout, stderr=%s err=%v", stderr, err)
	}
	if err == nil {
		t.Fatalf("expected non-zero exit because required sentry failed; stdout=%s stderr=%s", stdout, stderr)
	}
}
