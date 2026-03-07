package web

import (
	"html/template"
	"log"
	"net/http"
	"sort"

	"lmon/aggregator"
)

// AggregatorNodeData holds the display data for a single node in the aggregator view.
type AggregatorNodeData struct {
	Name      string
	Available bool
	Error     string
	Monitors  []aggregator.ScrapedMetric
	Timestamp string
}

// handleAggregator returns a handler that renders the aggregator dashboard.
func (s *Server) handleAggregator(agg *aggregator.Aggregator) http.HandlerFunc {
	// Parse templates once
	templ, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		log.Printf("handleAggregator (Parse Error): %v", err)
		return func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "Internal Server Error: template parse issue", http.StatusInternalServerError)
		}
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()

		results := agg.Results()
		nodes := make([]AggregatorNodeData, 0, len(results))

		for name, result := range results {
			nodeData := AggregatorNodeData{
				Name:      name,
				Available: result.Available,
				Error:     result.Error,
				Timestamp: result.Timestamp.Format("15:04:05 UTC"),
			}
			if result.Metrics != nil {
				nodeData.Monitors = result.Metrics.Monitors
			}
			nodes = append(nodes, nodeData)
		}

		// Sort nodes by name for consistent display
		sort.Slice(nodes, func(i, j int) bool {
			return nodes[i].Name < nodes[j].Name
		})

		data := map[string]any{
			"title":            s.config.Monitoring.System.Title,
			"dashboard_title":  s.config.Monitoring.System.Title,
			"refresh_interval": s.config.Monitoring.Interval,
			"Nodes":            nodes,
			"ActivePage":       "aggregator",
		}

		t := templ.Lookup("aggregator.html")
		if t == nil {
			http.Error(w, "Template not found", http.StatusNotFound)
			return
		}

		if err := templ.ExecuteTemplate(w, "aggregator.html", data); err != nil {
			log.Printf("handleAggregator template error: %v", err)
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
	}
}

// SetupAggregatorRoutes registers the aggregator-specific routes on the server.
func (s *Server) SetupAggregatorRoutes(agg *aggregator.Aggregator) {
	s.router.HandleFunc("GET /", s.handleAggregator(agg))
	s.router.HandleFunc("GET /index.html", s.handleAggregator(agg))
}
