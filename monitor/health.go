package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"lmon/config"
)

// HTTPClient is an interface for HTTP clients
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// HealthMonitor represents a health check monitor
type HealthMonitor struct {
	config *config.Config
	client HTTPClient
}

// NewHealthMonitor creates a new health check monitor
func NewHealthMonitor(cfg *config.Config) *HealthMonitor {
	return &HealthMonitor{
		config: cfg,
		client: &http.Client{
			Timeout: time.Second * 10, // Default timeout
		},
	}
}

// NewHealthMonitorWithClient creates a new health check monitor with a custom HTTP client
func NewHealthMonitorWithClient(cfg *config.Config, client HTTPClient) *HealthMonitor {
	return &HealthMonitor{
		config: cfg,
		client: client,
	}
}

// Check checks all configured health endpoints
func (m *HealthMonitor) Check() ([]*Item, error) {
	var items []*Item

	for _, healthCfg := range m.config.Monitoring.Healthchecks {
		// Create a context with the configured timeout
		timeout := time.Duration(healthCfg.Timeout) * time.Second
		if timeout == 0 {
			timeout = time.Second * 10 // Default timeout
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		// Create a new request
		req, err := http.NewRequestWithContext(ctx, "GET", healthCfg.URL, nil)
		if err != nil {
			items = append(items, &Item{
				ID:        fmt.Sprintf("health-%s", healthCfg.Name),
				Name:      healthCfg.Name,
				Type:      "health",
				Status:    StatusUnknown,
				Value:     0,
				Threshold: 0,
				Unit:      "",
				Icon:      healthCfg.Icon,
				LastCheck: time.Now(),
				Message:   fmt.Sprintf("Failed to create request: %v", err),
			})
			continue
		}

		// Send the request
		resp, err := m.client.Do(req)
		if err != nil {
			items = append(items, &Item{
				ID:        fmt.Sprintf("health-%s", healthCfg.Name),
				Name:      healthCfg.Name,
				Type:      "health",
				Status:    StatusCritical,
				Value:     0,
				Threshold: 0,
				Unit:      "",
				Icon:      healthCfg.Icon,
				LastCheck: time.Now(),
				Message:   fmt.Sprintf("Failed to connect: %v", err),
			})
			continue
		}
		defer resp.Body.Close()

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			items = append(items, &Item{
				ID:        fmt.Sprintf("health-%s", healthCfg.Name),
				Name:      healthCfg.Name,
				Type:      "health",
				Status:    StatusCritical,
				Value:     0,
				Threshold: 0,
				Unit:      "",
				Icon:      healthCfg.Icon,
				LastCheck: time.Now(),
				Message:   fmt.Sprintf("Failed to read response: %v", err),
			})
			continue
		}

		// Determine status based on response
		status := StatusOK
		message := "Service is healthy"

		// Check HTTP status code
		if resp.StatusCode != http.StatusOK {
			status = StatusCritical
			message = fmt.Sprintf("Unhealthy status code: %d", resp.StatusCode)
		} else {
			// Try to parse as Kubernetes/Docker health check format
			var healthResponse struct {
				Status string `json:"status"`
			}
			if err := json.Unmarshal(body, &healthResponse); err == nil {
				if healthResponse.Status != "ok" && healthResponse.Status != "UP" && healthResponse.Status != "healthy" {
					status = StatusCritical
					message = fmt.Sprintf("Unhealthy status: %s", healthResponse.Status)
				}
			}
		}

		// Create item
		item := &Item{
			ID:        fmt.Sprintf("health-%s", healthCfg.Name),
			Name:      healthCfg.Name,
			Type:      "health",
			Status:    status,
			Value:     float64(resp.StatusCode),
			Threshold: 200, // Expect 2xx status codes
			Unit:      "status",
			Icon:      healthCfg.Icon,
			LastCheck: time.Now(),
			Message:   message,
		}

		items = append(items, item)
	}

	return items, nil
}
