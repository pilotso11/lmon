package web

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"lmon/db"
)

var (
	historyTmplOnce sync.Once
	historyTmpl     *template.Template
	historyTmplErr  error
)

func getHistoryTemplate() (*template.Template, error) {
	historyTmplOnce.Do(func() {
		historyTmpl, historyTmplErr = template.ParseFS(templateFS, "templates/history.html", "templates/header.html", "templates/nav.html", "templates/footer.html")
	})
	return historyTmpl, historyTmplErr
}

// handleGetHistory responds with JSON containing monitor snapshots for the given query params.
// Query parameters: node, monitor, from (RFC3339), to (RFC3339), limit (int).
// Route: GET /api/history
func (s *Server) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	if s.store == nil || !s.store.IsAvailable() {
		http.Error(w, "Database unavailable", http.StatusServiceUnavailable)
		return
	}
	node := r.URL.Query().Get("node")
	monitor := r.URL.Query().Get("monitor")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	limitStr := r.URL.Query().Get("limit")

	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()
	limit := 500

	if fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			from = t
		}
	}
	if toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			to = t
		}
	}
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	// Cap limit to prevent excessive queries
	if limit > 10000 {
		limit = 10000
	}

	snapshots, err := s.store.GetHistory(r.Context(), node, monitor, from, to, limit)
	if err != nil {
		log.Printf("handleGetHistory: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	s.writeJson(w, snapshots)
}

// handleGetSummary responds with JSON containing aggregated status counts for all monitors.
// Query parameters: from (RFC3339), to (RFC3339).
// Route: GET /api/summary
func (s *Server) handleGetSummary(w http.ResponseWriter, r *http.Request) {
	if s.store == nil || !s.store.IsAvailable() {
		http.Error(w, "Database unavailable", http.StatusServiceUnavailable)
		return
	}
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()

	if fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			from = t
		}
	}
	if toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			to = t
		}
	}

	summary, err := s.store.GetSummary(r.Context(), from, to)
	if err != nil {
		log.Printf("handleGetSummary: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	s.writeJson(w, summary)
}

// handleHistoryPage serves the history page template.
// If the database is unavailable, it renders a friendly message.
// Uses map[string]any for template data to match existing template conventions.
// Route: GET /history
func (s *Server) handleHistoryPage(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tmpl, err := getHistoryTemplate()
	if err != nil {
		log.Printf("handleHistoryPage template parse error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	dbAvailable := s.store != nil && s.store.IsAvailable()
	node := r.URL.Query().Get("node")
	monitor := r.URL.Query().Get("monitor")

	data := map[string]any{
		"title":               s.config.Monitoring.System.Title,
		"ActivePage":          "history",
		"DBAvailable":         dbAvailable,
		"Node":                node,
		"Monitor":             monitor,
		"Config":              s.config,
		"UpdateAt":            time.Now().Format("2006-01-02 15:04:05Z"),
		"default_health_icon": "",
		"default_disk_icon":   "",
		"default_ping_icon":   "",
		"default_docker_icon": "",
	}

	if dbAvailable {
		from := time.Now().Add(-24 * time.Hour)
		to := time.Now()

		if fromStr := r.URL.Query().Get("from"); fromStr != "" {
			if t, parseErr := time.Parse(time.RFC3339, fromStr); parseErr == nil {
				from = t
			}
		}
		if toStr := r.URL.Query().Get("to"); toStr != "" {
			if t, parseErr := time.Parse(time.RFC3339, toStr); parseErr == nil {
				to = t
			}
		}

		snapshots, queryErr := s.store.GetHistory(r.Context(), node, monitor, from, to, 200)
		if queryErr != nil {
			log.Printf("handleHistoryPage query error: %v", queryErr)
		} else {
			data["Snapshots"] = snapshots
			// Reverse for sparkline (oldest first)
			reversed := make([]db.MonitorSnapshot, len(snapshots))
			for i, snap := range snapshots {
				reversed[len(snapshots)-1-i] = snap
			}
			data["Sparkline"] = GenerateSparklineSVG(reversed)
		}
	}

	err = tmpl.ExecuteTemplate(w, "history.html", data)
	if err != nil {
		log.Printf("handleHistoryPage execute error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}
