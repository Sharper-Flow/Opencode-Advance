package dashboard

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenBrowser_CommandSelection(t *testing.T) {
	// We can't test actual browser opening, but we can verify the command
	// builder returns a valid command for each OS.
	tests := []struct {
		goos string
		cmd  string
	}{
		{"darwin", "open"},
		{"linux", "xdg-open"},
		{"windows", "cmd"},
	}
	for _, tt := range tests {
		cmd := browserCmd(tt.goos, "http://localhost:9191")
		if cmd.Path == "" && cmd.Args == nil {
			t.Errorf("browserCmd(%s) returned empty command", tt.goos)
		}
		// Verify the URL is in the args
		found := false
		for _, arg := range cmd.Args {
			if strings.Contains(arg, "localhost:9191") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("browserCmd(%s) args don't contain URL: %v", tt.goos, cmd.Args)
		}
	}
}

func TestLoadingPlaceholder(t *testing.T) {
	state := NewState()
	// Not warmed yet
	snap := state.Snapshot()
	if snap.Warm {
		t.Error("new state should not be warm")
	}
	// Template test — verify index renders loading state
	handler := indexHandler(state)
	w := httptestResponse("GET", "/", handler)
	body := w.Body.String()
	if !strings.Contains(body, "Loading") {
		t.Error("unwarmed state should show loading placeholder")
	}
}

func TestErrorBadges_TemporalDown(t *testing.T) {
	state := NewState()
	state.Update(func(snap *Snapshot) {
		snap.Warm = true
		snap.Health = HealthRow{TemporalReachable: false, WorkerRunning: false}
	})

	handler := indexHandler(state)
	w := httptestResponse("GET", "/", handler)
	body := w.Body.String()
	if !strings.Contains(body, "Temporal unreachable") {
		t.Error("should show Temporal unreachable badge when down")
	}
	if !strings.Contains(body, "Worker down") {
		t.Error("should show Worker down badge when down")
	}
}

func TestErrorBadges_AllHealthy(t *testing.T) {
	state := NewState()
	state.Update(func(snap *Snapshot) {
		snap.Warm = true
		snap.Health = HealthRow{TemporalReachable: true, WorkerRunning: true}
	})

	handler := indexHandler(state)
	w := httptestResponse("GET", "/", handler)
	body := w.Body.String()
	if !strings.Contains(body, "● Temporal") {
		t.Error("should show healthy Temporal indicator")
	}
	if strings.Contains(body, "unreachable") {
		t.Error("should NOT show unreachable when healthy")
	}
}

// httptestResponse is a test helper for making a request and getting the response.
func httptestResponse(method, path string, handler http.Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}
