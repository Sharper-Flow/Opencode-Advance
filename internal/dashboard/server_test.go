package dashboard_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Sharper-Flow/Opencode-Advance/internal/dashboard"
)

func TestServer_Routes(t *testing.T) {
	srv, err := dashboard.NewServer(dashboard.Config{})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	routes := []struct {
		path       string
		wantStatus int
	}{
		{"/", http.StatusOK},
		{"/api/changes", http.StatusOK},
		{"/api/sessions", http.StatusOK},
		{"/api/health", http.StatusOK},
	}

	for _, tc := range routes {
		t.Run(tc.path, func(t *testing.T) {
			resp, err := http.Get(ts.URL + tc.path)
			if err != nil {
				t.Fatalf("GET %s: %v", tc.path, err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tc.wantStatus {
				t.Errorf("GET %s: got status %d, want %d", tc.path, resp.StatusCode, tc.wantStatus)
			}
		})
	}
}

func TestServer_ShutdownWithin100ms(t *testing.T) {
	srv, err := dashboard.NewServer(dashboard.Config{})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- srv.Run(ctx)
	}()

	cancel() // trigger shutdown

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("shutdown took longer than 100ms")
	}
}
