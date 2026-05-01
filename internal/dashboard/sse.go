package dashboard

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/starfederation/datastar-go/datastar"
)

// sseHandler handles SSE connections on /api/events.
func sseHandler(s *State, registry *subscriberRegistry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := slog.Default()

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		// Flush headers
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		sub := registry.Subscribe()
		defer registry.Unsubscribe(sub)

		sse := datastar.NewSSE(w, r)

		for {
			select {
			case <-r.Context().Done():
				logger.Debug("SSE client disconnected")
				return
			case snap, ok := <-sub.ch:
				if !ok {
					return
				}
				if err := sendSnapshotPatch(sse, snap); err != nil {
					logger.Debug("SSE write error", "error", err)
					return
				}
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			}
		}
	})
}

// sendSnapshotPatch sends updated table rows and health bar as HTML patches.
func sendSnapshotPatch(sse *datastar.ServerSentEventGenerator, snap Snapshot) error {
	// Build changes table body
	var b strings.Builder
	b.WriteString(`<tbody id="changes-table">`)
	if len(snap.Changes) == 0 {
		b.WriteString(`<tr><td colspan="4" class="loading">No active changes</td></tr>`)
	} else {
		for _, c := range snap.Changes {
			fmt.Fprintf(&b,
				`<tr><td>%s</td><td>%s</td><td><span class="badge badge-%s">%s</span></td><td>%s</td></tr>`,
				escapeHTML(c.ID), escapeHTML(c.Title), escapeHTML(c.Status), escapeHTML(c.Status), escapeHTML(c.Project),
			)
		}
	}
	b.WriteString(`</tbody>`)

	if err := sse.PatchElements(b.String()); err != nil {
		return err
	}

	// Build health bar
	var hb strings.Builder
	hb.WriteString(`<div class="health-bar" id="health-bar">`)
	if snap.Health.TemporalReachable {
		hb.WriteString(`<span class="ok">● Temporal</span> `)
	} else {
		hb.WriteString(`<span class="warn">● Temporal unreachable</span> `)
	}
	if snap.Health.WorkerRunning {
		hb.WriteString(`<span class="ok">● Worker</span>`)
	} else {
		hb.WriteString(`<span class="warn">● Worker down</span>`)
	}
	hb.WriteString(`</div>`)

	return sse.PatchElements(hb.String())
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
