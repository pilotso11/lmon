package aggregator

import (
	"context"
	"sync"
)

// MockProvider is a test double for the aggregator Provider interface.
type MockProvider struct {
	mu           sync.Mutex
	nodes        []NodeEndpoint
	metrics      map[string]*ScrapedPayload
	nodeErr      error
	scrapeErrors map[string]error
}

// NewMockProvider creates a new MockProvider with the given nodes and metrics.
func NewMockProvider(nodes []NodeEndpoint, metrics map[string]*ScrapedPayload) *MockProvider {
	if metrics == nil {
		metrics = make(map[string]*ScrapedPayload)
	}
	return &MockProvider{
		nodes:        nodes,
		metrics:      metrics,
		scrapeErrors: make(map[string]error),
	}
}

// SetNodes updates the discoverable nodes.
func (m *MockProvider) SetNodes(nodes []NodeEndpoint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nodes = nodes
}

// SetNodeError sets the discovery error.
func (m *MockProvider) SetNodeError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nodeErr = err
}

// SetMetrics sets the metrics response for a given endpoint.
func (m *MockProvider) SetMetrics(endpoint string, payload *ScrapedPayload) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics[endpoint] = payload
}

// SetScrapeError sets a scrape error for a given endpoint.
func (m *MockProvider) SetScrapeError(endpoint string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scrapeErrors[endpoint] = err
}

func (m *MockProvider) DiscoverNodes(_ context.Context, _ string) ([]NodeEndpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.nodeErr != nil {
		return nil, m.nodeErr
	}
	return m.nodes, nil
}

func (m *MockProvider) ScrapeMetrics(_ context.Context, endpoint string) (*ScrapedPayload, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err, ok := m.scrapeErrors[endpoint]; ok && err != nil {
		return nil, err
	}
	if metrics, ok := m.metrics[endpoint]; ok {
		return metrics, nil
	}
	return &ScrapedPayload{}, nil
}
