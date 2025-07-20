package ping

import (
	"context"
)

// MockPingProvider is a mock implementation of PingProvider for unit and integration tests.
// It allows you to simulate ping responses and errors.
type MockPingProvider struct {
	ResponseMs int   // Simulated response time in milliseconds
	Err        error // Simulated error (if any)
}

// Ping simulates an ICMP ping by returning the configured response time and error.
func (m *MockPingProvider) Ping(_ context.Context, _ string, _ int) (int, error) {
	return m.ResponseMs, m.Err
}

// NewMockPingProvider returns a new MockPingProvider with the specified response time and error.
func NewMockPingProvider(responseMs int, err error) *MockPingProvider {
	return &MockPingProvider{
		ResponseMs: responseMs,
		Err:        err,
	}
}
