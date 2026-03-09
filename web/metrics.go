package web

import (
	"net/http"
	"os"
	"sort"
	"time"

	"lmon/monitors"
)

// MetricsPayload is the JSON response for the /metrics endpoint.
type MetricsPayload struct {
	Node      string          `json:"node"`
	Timestamp time.Time       `json:"timestamp"`
	Monitors  []MonitorMetric `json:"monitors"`
}

// MonitorMetric represents a single monitor's status in the metrics payload.
type MonitorMetric struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// handleMetrics responds with JSON containing all monitor results.
func (s *Server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	results := s.monitor.Results()
	metrics := make([]MonitorMetric, 0, len(results))
	for id, r := range results {
		metrics = append(metrics, MonitorMetric{
			ID:      id,
			Type:    r.Group,
			Status:  r.Status.String(),
			Value:   r.Value,
			Message: r.Value2,
		})
	}

	// Sort by ID for deterministic output
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].ID < metrics[j].ID
	})

	hostname, _ := os.Hostname()
	payload := MetricsPayload{
		Node:      hostname,
		Timestamp: time.Now().UTC(),
		Monitors:  metrics,
	}

	s.writeJson(w, payload)
}

// buildMetrics is a helper for testing that builds the metrics from results without HTTP.
func buildMetrics(results map[string]monitors.Result) []MonitorMetric {
	metrics := make([]MonitorMetric, 0, len(results))
	for id, r := range results {
		metrics = append(metrics, MonitorMetric{
			ID:      id,
			Type:    r.Group,
			Status:  r.Status.String(),
			Value:   r.Value,
			Message: r.Value2,
		})
	}
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].ID < metrics[j].ID
	})
	return metrics
}
