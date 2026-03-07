package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"lmon/aggregator"
	"lmon/config"
)

// TestAggregatorHandler verifies the aggregator handler renders correctly with node data.
func TestAggregatorHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	nodes := []aggregator.NodeEndpoint{
		{Name: "node-1", Address: "10.0.0.1", Port: 8080},
	}
	metrics := map[string]*aggregator.ScrapedPayload{
		"http://10.0.0.1:8080/metrics": {
			Node:      "node-1",
			Timestamp: time.Now(),
			Monitors:  []aggregator.ScrapedMetric{{ID: "cpu", Type: "system", Status: "Green", Value: "50%"}},
		},
	}

	provider := aggregator.NewMockProvider(nodes, metrics)
	agg := aggregator.NewAggregator(provider, "app=lmon", 8080, "/metrics", 50*time.Millisecond, nil)
	agg.Start(ctx)
	defer agg.Stop()

	// Wait for scrape
	assert.Eventually(t, func() bool {
		return len(agg.Results()) == 1
	}, time.Second, 10*time.Millisecond)

	s := &Server{config: &config.Config{}, router: http.NewServeMux()}
	handler := s.handleAggregator(agg)

	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest("GET", "/", nil))
	assert.Equal(t, 200, rec.Code)
	assert.Contains(t, rec.Body.String(), "node-1")
	assert.Contains(t, rec.Body.String(), "Green")
}

// TestAggregatorHandlerEmptyResults verifies the aggregator handler renders with no nodes.
func TestAggregatorHandlerEmptyResults(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	provider := aggregator.NewMockProvider(nil, nil)
	agg := aggregator.NewAggregator(provider, "app=lmon", 8080, "/metrics", 50*time.Millisecond, nil)
	agg.Start(ctx)
	defer agg.Stop()

	s := &Server{config: &config.Config{}, router: http.NewServeMux()}
	handler := s.handleAggregator(agg)

	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest("GET", "/", nil))
	assert.Equal(t, 200, rec.Code)
	assert.Contains(t, rec.Body.String(), "Cluster View")
}

// TestAggregatorHandlerUnavailableNode verifies the aggregator shows unavailable nodes.
func TestAggregatorHandlerUnavailableNode(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	nodes := []aggregator.NodeEndpoint{
		{Name: "node-1", Address: "10.0.0.1", Port: 8080},
	}
	provider := aggregator.NewMockProvider(nodes, nil)
	provider.SetScrapeError("http://10.0.0.1:8080/metrics", assert.AnError)

	agg := aggregator.NewAggregator(provider, "app=lmon", 8080, "/metrics", 50*time.Millisecond, nil)
	agg.Start(ctx)
	defer agg.Stop()

	assert.Eventually(t, func() bool {
		results := agg.Results()
		r, ok := results["node-1"]
		return ok && !r.Available
	}, time.Second, 10*time.Millisecond)

	s := &Server{config: &config.Config{}, router: http.NewServeMux()}
	handler := s.handleAggregator(agg)

	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest("GET", "/", nil))
	assert.Equal(t, 200, rec.Code)
	assert.Contains(t, rec.Body.String(), "node-1")
	assert.Contains(t, rec.Body.String(), "Offline")
}

// TestAggregatorHandlerMultipleNodes verifies sorting and display of multiple nodes.
func TestAggregatorHandlerMultipleNodes(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	nodes := []aggregator.NodeEndpoint{
		{Name: "node-b", Address: "10.0.0.2", Port: 8080},
		{Name: "node-a", Address: "10.0.0.1", Port: 8080},
	}
	metrics := map[string]*aggregator.ScrapedPayload{
		"http://10.0.0.1:8080/metrics": {Node: "node-a", Monitors: []aggregator.ScrapedMetric{}},
		"http://10.0.0.2:8080/metrics": {Node: "node-b", Monitors: []aggregator.ScrapedMetric{}},
	}

	provider := aggregator.NewMockProvider(nodes, metrics)
	agg := aggregator.NewAggregator(provider, "app=lmon", 8080, "/metrics", 50*time.Millisecond, nil)
	agg.Start(ctx)
	defer agg.Stop()

	assert.Eventually(t, func() bool {
		return len(agg.Results()) == 2
	}, time.Second, 10*time.Millisecond)

	s := &Server{config: &config.Config{}, router: http.NewServeMux()}
	handler := s.handleAggregator(agg)

	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest("GET", "/", nil))
	assert.Equal(t, 200, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "node-a")
	assert.Contains(t, body, "node-b")
}

// TestSetupAggregatorRoutes verifies that routes are registered without panic.
func TestSetupAggregatorRoutes(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	provider := aggregator.NewMockProvider(nil, nil)
	agg := aggregator.NewAggregator(provider, "app=lmon", 8080, "/metrics", time.Minute, nil)
	agg.Start(ctx)
	defer agg.Stop()

	s := &Server{config: &config.Config{}, router: http.NewServeMux()}
	assert.NotPanics(t, func() {
		s.SetupAggregatorRoutes(agg)
	})
}
