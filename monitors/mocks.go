// Package monitors provides core monitoring abstractions and helpers for lmon.
// This file contains mock implementations for testing monitor logic and push notifications.
package monitors

import (
	"context"
	"fmt"
	"sync"

	"github.com/puzpuzpuz/xsync/v4"
	"go.uber.org/atomic"

	"lmon/config"
)

type mockResult struct {
	m      Monitor
	prev   Result
	Result Result
}

// MockPush is a test double for PushFunc that records calls and their arguments.
// It is safe for concurrent use in tests.
type MockPush struct {
	cnt   atomic.Int32
	Calls *xsync.Map[int32, mockResult]
}

// NewMockPush creates a new MockPush instance for use in tests.
func NewMockPush() *MockPush {
	return &MockPush{
		Calls: xsync.NewMap[int32, mockResult](),
	}
}

// Push implements the PushFunc signature and records each call with its arguments.
func (m *MockPush) Push(_ context.Context, mon Monitor, prev Result, result Result) {
	m.Calls.Store(m.cnt.Inc(), mockResult{mon, prev, result})
}

// ClearCalls removes all recorded push calls.
func (m *MockPush) ClearCalls() {
	m.Calls.Clear()
}

// MockMonitor is a test double for the Monitor interface.
// It allows simulation of status changes and tracks the number of checks.
type MockMonitor struct {
	name   string
	status []struct {
		rag RAG
		msg string
	}
	group        string
	Checks       atomic.Int32
	mu           sync.Mutex // Protects status slice
	alertThreshold int
}

// NewMockMonitor creates a new MockMonitor with the given name and group.
func NewMockMonitor(name string, group string) *MockMonitor {
	return &MockMonitor{
		name:  name,
		group: group,
		alertThreshold: 1, // Default alert threshold
	}
}

// Check implements Monitor. It returns the next status in the queue, or RAGGreen by default.
// Each call increments the Checks counter.
func (m *MockMonitor) Check(_ context.Context) Result {
	m.Checks.Inc()
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if len(m.status) > 0 {
		rag := m.status[0].rag
		msg := m.status[0].msg
		m.status = m.status[1:]
		return Result{Status: rag, Value: msg}
	}
	return Result{Status: RAGGreen, Value: fmt.Sprintf("ok check %d", m.Checks.Load())}
}

// DisplayName implements Monitor. Returns a human-readable name for the mock monitor.
func (m *MockMonitor) DisplayName() string {
	return fmt.Sprintf("MockMonitor %s", m.name)
}

// Group implements Monitor. Returns the group name for the mock monitor.
func (m *MockMonitor) Group() string {
	return m.group
}

// Name implements Monitor. Returns the unique name for the mock monitor.
func (m *MockMonitor) Name() string {
	return m.name
}

// Save implements Monitor. No-op for the mock monitor.
func (m *MockMonitor) Save(_ *config.Config) {
	// noop
}

// AlertThreshold implements Monitor. Returns the configured alert threshold for the mock monitor.
func (m *MockMonitor) AlertThreshold() int {
	return m.alertThreshold
}

// SetStatuses sets the status queue for the mock monitor in a thread-safe manner.
// This is used in tests to configure expected status returns.
func (m *MockMonitor) SetStatuses(statuses []struct {
	rag RAG
	msg string
}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = statuses
}

// SetAlertThreshold sets the alert threshold for the mock monitor.
func (m *MockMonitor) SetAlertThreshold(threshold int) {
	m.alertThreshold = threshold
}

// StatusLen returns the number of statuses remaining in the queue.
// This is useful for tests to verify status consumption.
func (m *MockMonitor) StatusLen() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.status)
}
