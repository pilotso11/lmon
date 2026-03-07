package k8sevents

import (
	"context"
	"sync/atomic"
)

// MockProvider is a mock implementation of Provider for unit and integration tests.
// It allows you to simulate Kubernetes event responses and errors.
type MockProvider struct {
	Events atomic.Pointer[[]PodEvent]
	Err    atomic.Pointer[error]
}

// NewMockProvider returns a new MockProvider with the specified events and error.
func NewMockProvider(events []PodEvent, err error) *MockProvider {
	p := &MockProvider{}
	p.Events.Store(&events)
	if err != nil {
		p.Err.Store(&err)
	}
	return p
}

// GetFailureEvents returns the configured events and error.
func (m *MockProvider) GetFailureEvents(_ context.Context, _ string, _ int) ([]PodEvent, error) {
	if err := m.Err.Load(); err != nil {
		return nil, *err
	}
	events := m.Events.Load()
	if events == nil {
		return nil, nil
	}
	return *events, nil
}
