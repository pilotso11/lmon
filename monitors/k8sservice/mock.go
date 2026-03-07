package k8sservice

import (
	"context"
	"sync/atomic"
)

// MockProvider is a mock implementation of Provider for unit and integration tests.
// It allows you to simulate Kubernetes service pod health responses and errors.
type MockProvider struct {
	Pods atomic.Pointer[[]PodEndpoint]
	Err  atomic.Pointer[error]
}

// NewMockProvider returns a new MockProvider with the specified pod endpoints and error.
func NewMockProvider(pods []PodEndpoint, err error) *MockProvider {
	p := &MockProvider{}
	p.Pods.Store(&pods)
	if err != nil {
		p.Err.Store(&err)
	}
	return p
}

// GetPodHealth returns the configured pod endpoints and error.
func (m *MockProvider) GetPodHealth(_ context.Context, _, _ string, _ int, _ string, _ int) ([]PodEndpoint, error) {
	if err := m.Err.Load(); err != nil {
		return nil, *err
	}
	pods := m.Pods.Load()
	if pods == nil {
		return nil, nil
	}
	return *pods, nil
}
