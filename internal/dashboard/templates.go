package dashboard

import (
	"embed"
	"encoding/json"
	"html/template"
	"io/fs"
	"net/http"
)

//go:embed frontend/dist/*
var staticFS embed.FS

// templateFS is the same embed but exposed for template parsing.
var templateFS embed.FS

func init() {
	templateFS = staticFS
}

// indexData is the template data for the index page.
type indexData struct {
	Snapshot     *Snapshot
	JSONChanges  string
}

// indexHandler returns an http.Handler that renders the dashboard index.
func indexHandler(s *State) http.Handler {
	tmpl := template.Must(template.ParseFS(templateFS,
		"frontend/dist/base.html",
		"frontend/dist/index.html",
	))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		snap := s.Snapshot()

		jsonChanges, err := json.Marshal(snap.Changes)
		if err != nil {
			jsonChanges = []byte("[]")
		}

		data := indexData{
			Snapshot:    &snap,
			JSONChanges: string(jsonChanges),
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	})
}

// staticHandler returns an http.Handler serving embedded static files.
func staticHandler() http.Handler {
	sub, err := fs.Sub(staticFS, "frontend/dist")
	if err != nil {
		panic("static FS: " + err.Error())
	}
	return http.FileServer(http.FS(sub))
}
