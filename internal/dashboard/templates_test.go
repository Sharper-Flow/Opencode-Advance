package dashboard

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTemplates_EmbedCompiles(t *testing.T) {
	// Verify that the embedded FS is non-empty and parseable.
	// embed.FS zero-value has no files; check ReadDir instead.
	entries, err := templateFS.ReadDir("frontend/dist")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("templateFS has no files — embed failed")
	}
	tmpl, err := template.ParseFS(templateFS, "frontend/dist/*.html")
	if err != nil {
		t.Fatalf("ParseFS: %v", err)
	}
	if tmpl == nil {
		t.Fatal("parsed template tree is nil")
	}
}

func TestTemplates_IndexRenders(t *testing.T) {
	s := NewState()
	s.Update(func(snap *Snapshot) {
		snap.Warm = true
		snap.Changes = []ChangeRow{
			{ID: "ch-abc", Title: "Test Change", Status: "active", Project: "myproject"},
		}
		snap.Health = HealthRow{TemporalReachable: true, WorkerRunning: true}
	})

	handler := indexHandler(s)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Test Change") {
		t.Error("response should contain change title")
	}
	if !strings.Contains(body, `data-star`) && !strings.Contains(body, `datastar`) {
		t.Error("response should reference datastar")
	}
}

func TestTemplates_DatastarJSLoads(t *testing.T) {
	// Verify that the embedded FS contains a datastar JS file.
	entries, err := templateFS.ReadDir("frontend/dist")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	found := false
	for _, e := range entries {
		if strings.Contains(e.Name(), "datastar") && strings.HasSuffix(e.Name(), ".js") {
			found = true
			break
		}
	}
	if !found {
		t.Error("no datastar JS file found in embedded frontend/dist")
	}
}
