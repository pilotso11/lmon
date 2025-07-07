package monitors

import (
	"context"
	"fmt"

	"github.com/puzpuzpuz/xsync/v4"
	"go.uber.org/atomic"

	"lmon/config"
)

type mockResult struct {
	m      Monitor
	prev   Result
	Result Result
}
type MockPush struct {
	cnt   atomic.Int32
	Calls *xsync.Map[int32, mockResult]
}

func NewMockPush() *MockPush {
	return &MockPush{
		Calls: xsync.NewMap[int32, mockResult](),
	}
}

func (m *MockPush) Push(_ context.Context, mon Monitor, prev Result, result Result) {
	m.Calls.Store(m.cnt.Inc(), mockResult{mon, prev, result})
}

func (m *MockPush) ClearCalls() {
	m.Calls.Clear()
}

type MockMonitor struct {
	name   string
	status []struct {
		rag RAG
		msg string
	}
	group  string
	Checks int
}

func NewMockMonitor(name string, group string) *MockMonitor {
	return &MockMonitor{
		name:  name,
		group: group,
	}
}

// Check implements Monitor.
func (m *MockMonitor) Check(_ context.Context) Result {
	m.Checks++
	if len(m.status) > 0 {
		rag := m.status[0].rag
		msg := m.status[0].msg
		m.status = m.status[1:]
		return Result{Status: rag, Value: msg}
	}
	return Result{Status: RAGGreen, Value: fmt.Sprintf("ok check %d", m.Checks)}
}

// DisplayName implements Monitor.
func (m *MockMonitor) DisplayName() string {
	return fmt.Sprintf("MockMonitor %s", m.name)
}

// Group implements Monitor.
func (m *MockMonitor) Group() string {
	return m.group
}

// Name implements Monitor.
func (m *MockMonitor) Name() string {
	return m.name
}

// Save implements Monitor.
func (m *MockMonitor) Save(_ *config.Config) {
	// noop
}
