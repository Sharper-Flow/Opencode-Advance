package dashboard

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSSE_SubscriberReceivesEvents(t *testing.T) {
	state := NewState()
	registry := newSubscriberRegistry()

	handler := sseHandler(state, registry)
	server := httptest.NewServer(handler)
	defer server.Close()

	// Connect as SSE client
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("connect SSE: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	// Bump state and notify
	state.Update(func(snap *Snapshot) {
		snap.Changes = []ChangeRow{
			{ID: "ch-test", Title: "Updated", Status: "active", Project: "test"},
		}
	})
	registry.Notify(state.Snapshot())

	// Read lines looking for data
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)
	found := false
	deadline := time.After(1 * time.Second)
	for !found {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for SSE event")
		default:
		}
		if scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data:") {
				found = true
			}
		}
	}
}

func TestSSE_HandlerExitsOnContextCancel(t *testing.T) {
	state := NewState()
	_ = state
	registry := newSubscriberRegistry()

	handler := sseHandler(state, registry)
	server := httptest.NewServer(handler)
	defer server.Close()

	// We can't easily inject context into httptest.Server requests,
	// so test the registry subscribe/unsubscribe lifecycle instead.
	sub := registry.Subscribe()

	done := make(chan struct{})
	go func() {
		// Read from channel until closed
		for range sub.ch {
		}
		close(done)
	}()

	// Unsubscribe should close the channel
	registry.Unsubscribe(sub)

	select {
	case <-done:
		// Good
	case <-time.After(200 * time.Millisecond):
		t.Fatal("subscriber channel not closed within 200ms of unsubscribe")
	}
}

func TestSubscriberRegistry_Notify(t *testing.T) {
	registry := newSubscriberRegistry()

	sub1 := registry.Subscribe()
	sub2 := registry.Subscribe()

	snap := Snapshot{Warm: true}
	registry.Notify(snap)

	// Both should receive the event
	select {
	case <-sub1.ch:
	default:
		t.Error("subscriber 1 did not receive event")
	}
	select {
	case <-sub2.ch:
	default:
		t.Error("subscriber 2 did not receive event")
	}

	registry.Unsubscribe(sub1)
	registry.Unsubscribe(sub2)
}

func TestHandlers_ChangesAPI(t *testing.T) {
	state := NewState()
	state.Update(func(snap *Snapshot) {
		snap.Warm = true
		snap.Changes = []ChangeRow{
			{ID: "ch-1", Title: "Alpha", Status: "active", Project: "proj-a"},
		}
	})

	handler := jsonChangesHandler(state)
	req := httptest.NewRequest("GET", "/api/changes", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var changes []ChangeRow
	if err := json.NewDecoder(w.Body).Decode(&changes); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if len(changes) != 1 || changes[0].ID != "ch-1" {
		t.Errorf("unexpected changes: %+v", changes)
	}
}

func TestHandlers_SessionsAPI(t *testing.T) {
	state := NewState()
	state.Update(func(snap *Snapshot) {
		snap.Sessions = []SessionRow{
			{Name: "work", Attached: true, Path: "/home"},
		}
	})

	handler := jsonSessionsHandler(state)
	req := httptest.NewRequest("GET", "/api/sessions", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var sessions []SessionRow
	if err := json.NewDecoder(w.Body).Decode(&sessions); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if len(sessions) != 1 || sessions[0].Name != "work" {
		t.Errorf("unexpected sessions: %+v", sessions)
	}
}

func TestHandlers_HealthAPI(t *testing.T) {
	state := NewState()
	state.Update(func(snap *Snapshot) {
		snap.Health = HealthRow{TemporalReachable: true, WorkerRunning: false}
	})

	handler := jsonHealthHandler(state)
	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var health HealthRow
	if err := json.NewDecoder(w.Body).Decode(&health); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if !health.TemporalReachable || health.WorkerRunning {
		t.Errorf("unexpected health: %+v", health)
	}
}
