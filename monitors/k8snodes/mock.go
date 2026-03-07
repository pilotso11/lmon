package k8snodes

import (
	"context"
	"sync/atomic"
)

// MockProvider is a mock implementation of Provider for unit and integration tests.
// It allows you to simulate Kubernetes node status responses and errors.
type MockProvider struct {
	Nodes atomic.Pointer[[]NodeStatus]
	Err   atomic.Pointer[error]
}

// NewMockProvider returns a new MockProvider with the specified node statuses and error.
func NewMockProvider(nodes []NodeStatus, err error) *MockProvider {
	p := &MockProvider{}
	p.Nodes.Store(&nodes)
	if err != nil {
		p.Err.Store(&err)
	}
	return p
}

// GetNodeStatuses returns the configured node statuses and error.
func (m *MockProvider) GetNodeStatuses(_ context.Context) ([]NodeStatus, error) {
	if err := m.Err.Load(); err != nil {
		return nil, *err
	}
	nodes := m.Nodes.Load()
	if nodes == nil {
		return nil, nil
	}
	return *nodes, nil
}
