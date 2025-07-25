package ping

import (
	"context"
	"sync/atomic"
)

// MockPingProvider is a mock implementation of PingProvider for unit and integration tests.
// It allows you to simulate ping responses and errors.
type MockPingProvider struct {
	ResponseMs atomic.Int32          // Simulated response time in milliseconds
	Err        atomic.Pointer[error] // Simulated error (if any)
}

// Ping simulates an ICMP ping by returning the configured response time and error.
func (m *MockPingProvider) Ping(_ context.Context, _ string, _ int) (int, error) {

	err := m.Err.Load()
	if err != nil {
		return int(m.ResponseMs.Load()), *err
	}
	return int(m.ResponseMs.Load()), nil
}

// NewMockPingProvider returns a new MockPingProvider with the specified response time and error.
func NewMockPingProvider(responseMs int, err error) *MockPingProvider {
	p := &MockPingProvider{}
	p.ResponseMs.Store(int32(responseMs))
	if err != nil {
		p.Err.Store(&err)
	} else {
		p.Err.Store(nil)
	}
	return p
}
