package monitors

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"lmon/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

type MockMonitor struct {
	name   string
	status []struct {
		rag RAG
		msg string
	}
	group  string
	checks int
}

// Check implements Monitor.
func (m *MockMonitor) Check(ctx context.Context) Result {
	m.checks++
	if len(m.status) > 0 {
		rag := m.status[0].rag
		msg := m.status[0].msg
		m.status = m.status[1:]
		return Result{Status: rag, Value: msg}
	}
	return Result{Status: GREEN, Value: fmt.Sprintf("ok check %d", m.checks)}
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

func TestNewService(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	svc := NewService(ctx, 10*time.Millisecond, time.Millisecond)
	assert.NotNil(t, svc, "service started")
	svc.Add(&MockMonitor{name: "test", group: "test"})
	_, ok := svc.monitors["test"]
	require.True(t, ok, "monitor added")
	time.Sleep(15 * time.Millisecond)
	svc.mu.Lock()
	defer svc.mu.Unlock()
	assert.Equal(t, 1, svc.monitors["test"].(*MockMonitor).checks, "checks called")
	assert.Equal(t, 1, len(svc.result), "len result")

	cancel()
}

func TestService_Add(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	svc := NewService(ctx, time.Second, time.Second)
	m1 := &MockMonitor{name: "test1", group: "test"}
	m2 := &MockMonitor{name: "test2", group: "test"}

	svc.Add(m1)
	assert.Len(t, svc.monitors, 1, "one monitor added")
	assert.Contains(t, svc.monitors, "test1", "monitor test1 added")

	svc.Add(m2)
	assert.Len(t, svc.monitors, 2, "two monitors added")
	assert.Contains(t, svc.monitors, "test2", "monitor test2 added")

	// Test overwriting existing monitor
	m3 := &MockMonitor{name: "test1", group: "test"}
	svc.Add(m3)
	assert.Len(t, svc.monitors, 2, "monitor count unchanged")
	assert.Same(t, m3, svc.monitors["test1"], "monitor was replaced")
}

func TestService_Remove(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	svc := NewService(ctx, time.Second, time.Second)
	m1 := &MockMonitor{name: "test1", group: "test"}
	m2 := &MockMonitor{name: "test2", group: "test"}

	// Add monitors
	svc.Add(m1)
	svc.Add(m2)
	assert.Len(t, svc.monitors, 2, "two monitors added")

	// Test successful removal
	err := svc.Remove(m1)
	assert.NoError(t, err, "remove should succeed")
	assert.Len(t, svc.monitors, 1, "one monitor should remain")
	assert.NotContains(t, svc.monitors, "test1", "monitor test1 should be removed")
	assert.Contains(t, svc.monitors, "test2", "monitor test2 should remain")

	// Test removing non-existent monitor
	err = svc.Remove(m1)
	assert.Error(t, err, "removing non-existent monitor should fail")
	assert.ErrorAs(t, err, &ErrNotFound{}, "ErrNotFound")
	assert.Equal(t, err.Error(), "monitor test1 not found", "error message")

	// Test removing the last monitor
	err = svc.Remove(m2)
	assert.NoError(t, err, "remove should succeed")
	assert.Len(t, svc.monitors, 0, "no monitors should remain")
}

func TestService_Results_CloneAndRace(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	svc := NewService(ctx, 10*time.Millisecond, time.Millisecond)
	mon := &MockMonitor{name: "race", group: "race"}
	svc.Add(mon)

	// Wait for at least one check to occur
	time.Sleep(15 * time.Millisecond)

	// Get results and mutate the returned map
	results := svc.Results()
	require.NotEmpty(t, results, "results should not be empty after check")
	origLen := len(results)

	// Mutate the returned map
	for k := range results {
		delete(results, k)
	}
	assert.Equal(t, 0, len(results), "mutated copy should be empty")

	// Internal state should not be affected
	results = svc.Results()
	assert.Equal(t, origLen, len(results), "internal results should be unchanged after mutation of copy")

	// Try mutating content
	for _, v := range results {
		v.Value = "bad"
	}

	for _, v := range svc.result {
		assert.NotEqual(t, "bad", v.Value, "internal results should be unchanged after mutation of copy")
	}

	// Race test: call Results() concurrently and mutate the returned maps
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range 1000 {
			res := svc.Results()
			for k := range res {
				delete(res, k)
			}
		}
	}()
	go func() {
		defer wg.Done()
		for range 1000 {
			res := svc.Results()
			for _, v := range res {
				v.Value = "bad"
			}
		}
	}()
	wg.Wait()
}

func TestService_SetPeriod_UpdatesPeriodAndRestarts(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	svc := NewService(ctx, 50*time.Millisecond, time.Millisecond)
	mon := &MockMonitor{name: "period", group: "period"}
	svc.Add(mon)

	// Wait for at least one check
	time.Sleep(60 * time.Millisecond)
	svc.mu.Lock()
	initialChecks := mon.checks
	initialPeriod := svc.period
	svc.mu.Unlock()
	require.GreaterOrEqual(t, initialChecks, 1, "should have at least one check")

	// Change period to a much shorter interval
	svc.SetPeriod(ctx, 10*time.Millisecond)
	svc.mu.Lock()
	updatedPeriod := svc.period
	svc.mu.Unlock()
	assert.Equal(t, 10*time.Millisecond, updatedPeriod, "period should be updated")

	// Wait for more checks to accumulate
	time.Sleep(35 * time.Millisecond)
	svc.mu.Lock()
	newChecks := mon.checks
	svc.mu.Unlock()
	assert.Greater(t, newChecks, initialChecks+2, "checks should increase after period change")
	assert.NotEqual(t, initialPeriod, updatedPeriod, "period should have changed")
}

func TestService_SetPeriod_Race(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	svc := NewService(ctx, 10*time.Millisecond, time.Millisecond)
	mon := &MockMonitor{name: "race", group: "race"}
	svc.Add(mon)

	done := make(chan struct{})
	go func() {
		for i := range 100 {
			svc.SetPeriod(ctx, time.Duration(5+i)*time.Millisecond)
		}
		done <- struct{}{}
	}()
	go func() {
		for range 100 {
			_ = svc.Results()
		}
		done <- struct{}{}
	}()
	<-done
	<-done
}

func TestRAG_String(t *testing.T) {
	tests := []struct {
		rag      RAG
		expected string
	}{
		{UNKNOWN, "Unknown"},
		{GREEN, "Green"},
		{YELLOW, "Yellow"},
		{RED, "Red"},
		{ERROR, "Error"},
		{RAG(99), "Unknown"},
		{RAG(-1), "Unknown"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.rag.String(), "RAG(%d) string mismatch", tt.rag)
	}
}
