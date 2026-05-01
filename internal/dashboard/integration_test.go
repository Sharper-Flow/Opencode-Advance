package dashboard

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestIntegration_ServerHTTP(t *testing.T) {
	if os.Getenv("CI") == "" && testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create server with random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	cfg := Config{Listener: listener, Logger: nil}
	srv, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- srv.Run(ctx)
	}()

	baseURL := "http://" + listener.Addr().String()
	client := &http.Client{Timeout: 5 * time.Second}

	// Test 1: Index returns 200
	resp, err := client.Get(baseURL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET / status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Test 2: JSON API - changes returns 200 with array
	resp, err = client.Get(baseURL + "/api/changes")
	if err != nil {
		t.Fatalf("GET /api/changes: %v", err)
	}
	var changes []ChangeRow
	if err := json.NewDecoder(resp.Body).Decode(&changes); err != nil {
		t.Fatalf("decode changes: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/changes status = %d", resp.StatusCode)
	}

	// Test 3: JSON API - health returns 200 with health object
	resp, err = client.Get(baseURL + "/api/health")
	if err != nil {
		t.Fatalf("GET /api/health: %v", err)
	}
	var health HealthRow
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("decode health: %v", err)
	}
	resp.Body.Close()

	// Test 4: Static file serves CSS
	resp, err = client.Get(baseURL + "/static/style.css")
	if err != nil {
		t.Fatalf("GET /static/style.css: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /static/style.css status = %d", resp.StatusCode)
	}

	// Test 5: SSE endpoint connects and verifies headers
	sseReq, err := http.NewRequest("GET", baseURL+"/api/events", nil)
	if err != nil {
		t.Fatalf("SSE request: %v", err)
	}
	sseReq.Header.Set("Accept", "text/event-stream")
	sseClient := &http.Client{Timeout: 3 * time.Second}
	sseResp, err := sseClient.Do(sseReq)
	if err != nil {
		t.Fatalf("SSE connect: %v", err)
	}
	if sseResp.StatusCode != http.StatusOK {
		t.Errorf("SSE status = %d", sseResp.StatusCode)
		sseResp.Body.Close()
	} else {
		ct := sseResp.Header.Get("Content-Type")
		if !strings.Contains(ct, "text/event-stream") {
			t.Errorf("SSE Content-Type = %q", ct)
		}
		// Close SSE connection before shutdown to avoid blocking
		sseResp.Body.Close()
	}

	// Test 6: Graceful shutdown within 2s
	shutdownStart := time.Now()
	cancel()

	select {
	case <-done:
		elapsed := time.Since(shutdownStart)
		if elapsed > time.Second {
			t.Errorf("shutdown took %v, want < 1s", elapsed)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not shut down within 2s")
	}
}

func TestIntegration_SSEReceivesEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	state := NewState()
	registry := newSubscriberRegistry()

	handler := sseHandler(state, registry)
	server := &http.Server{Handler: handler}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	go server.Serve(listener)
	defer server.Close()

	// Connect SSE client
	client := &http.Client{Timeout: 3 * time.Second}
	req, _ := http.NewRequest("GET", "http://"+listener.Addr().String(), nil)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// Trigger state change
	state.Update(func(snap *Snapshot) {
		snap.Warm = true
		snap.Changes = []ChangeRow{
			{ID: "ch-live", Title: "Live Test", Status: "active", Project: "test"},
		}
	})
	registry.Notify(state.Snapshot())

	// Read SSE stream
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)
	found := false
	deadline := time.After(2 * time.Second)
	for !found {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for SSE event with change data")
		default:
		}
		if scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "ch-live") {
				found = true
			}
		}
	}
}
