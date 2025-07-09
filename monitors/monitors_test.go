// monitors_test.go contains unit tests for the monitors package.
// These tests cover the Service lifecycle, monitor management, concurrency, and RAG status logic.
package monitors

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

// TestNewService verifies that a Service can be created, a monitor can be added, and checks are performed.
func TestNewService(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	svc := NewService(ctx, 10*time.Millisecond, time.Millisecond, nil)
	assert.NotNil(t, svc, "service started")
	svc.Add(ctx, NewMockMonitor("test", "test"))
	_, ok := svc.monitors.Load("test")
	require.True(t, ok, "monitor added")
	time.Sleep(15 * time.Millisecond)
	m, ok := svc.monitors.Load("test")
	require.True(t, ok, "monitor exists")
	assert.Equal(t, int32(3), m.(*MockMonitor).Checks.Load(), "checks called start + add + timer")
	assert.Equal(t, 1, svc.result.Size(), "len result")

	cancel()
}

// TestService_Add verifies adding monitors to the Service, including replacing an existing monitor.
func TestService_Add(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	svc := NewService(ctx, time.Second, time.Second, nil)
	m1 := NewMockMonitor("test1", "test")
	m2 := NewMockMonitor("test2", "test")

	svc.Add(ctx, m1)
	assert.Equal(t, 1, svc.monitors.Size(), "one monitor added")
	_, ok := svc.monitors.Load("test1")
	assert.True(t, ok, "monitor test1 added")

	svc.Add(ctx, m2)
	assert.Equal(t, 2, svc.monitors.Size(), "two monitors added")
	_, ok = svc.monitors.Load("test2")
	assert.True(t, ok, "monitor test2 added")

	// Test overwriting existing monitor
	m3 := NewMockMonitor("test1", "test")
	svc.Add(ctx, m3)
	assert.Equal(t, 2, svc.monitors.Size(), "monitor count unchanged")
	m, _ := svc.monitors.Load("test1")
	assert.Same(t, m3, m, "monitor was replaced")
}

// TestService_Remove verifies removing monitors from the Service, including error handling for missing monitors.
func TestService_Remove(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	svc := NewService(ctx, time.Second, time.Second, nil)
	m1 := NewMockMonitor("test1", "test")
	m2 := NewMockMonitor("test2", "test")

	// Add monitors
	svc.Add(ctx, m1)
	svc.Add(ctx, m2)
	assert.Equal(t, 2, svc.monitors.Size(), "two monitors added")

	// Test successful removal
	err := svc.Remove(m1)
	assert.NoError(t, err, "remove should succeed")
	assert.Equal(t, 1, svc.monitors.Size(), "one monitor should remain")
	_, ok := svc.monitors.Load("test1")
	assert.False(t, ok, "monitor test1 should be removed")
	_, ok = svc.monitors.Load("test2")
	assert.True(t, ok, "monitor test2 should remain")

	// Test removing non-existent monitor
	err = svc.Remove(m1)
	assert.Error(t, err, "removing non-existent monitor should fail")
	assert.ErrorAs(t, err, &ErrNotFound{}, "ErrNotFound")
	assert.Equal(t, err.Error(), "monitor test1 not found", "error message")

	// Test removing the last monitor
	err = svc.Remove(m2)
	assert.NoError(t, err, "remove should succeed")
	assert.Equal(t, 0, svc.monitors.Size(), "no monitors should remain")
}

// TestService_Results_CloneAndRace verifies that Results returns a safe-to-mutate clone and tests for race conditions.
func TestService_Results_CloneAndRace(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	svc := NewService(ctx, 10*time.Millisecond, time.Millisecond, nil)
	mon := NewMockMonitor("race", "race")
	svc.Add(ctx, mon)

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

	svc.result.Range(func(_ string, v Result) bool {
		assert.NotEqual(t, "bad", v.Value, "internal results should be unchanged after mutation of copy")
		return true
	})

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

// TestService_SetPeriod_UpdatesPeriodAndRestarts verifies that SetPeriod updates the check interval and restarts monitoring.
func TestService_SetPeriod_UpdatesPeriodAndRestarts(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	svc := NewService(ctx, 50*time.Millisecond, time.Millisecond, nil)
	mon := NewMockMonitor("period", "period")
	svc.Add(ctx, mon)

	// Wait for at least one check
	time.Sleep(60 * time.Millisecond)
	initialChecks := mon.Checks.Load()
	initialPeriod := svc.period.Load()
	require.GreaterOrEqual(t, initialChecks, int32(1), "should have at least one check")

	// Change period to a much shorter interval
	svc.SetPeriod(ctx, 10*time.Millisecond, 0)
	updatedPeriod := svc.period.Load()
	assert.Equal(t, 10*time.Millisecond, updatedPeriod, "period should be updated")

	// Wait for more checks to accumulate
	time.Sleep(35 * time.Millisecond)
	newChecks := mon.Checks.Load()
	assert.Greater(t, newChecks, initialChecks+2, "checks should increase after period change")
	assert.NotEqual(t, initialPeriod, updatedPeriod, "period should have changed")
}

// TestService_SetPeriod_Race verifies that SetPeriod and Results can be called concurrently without race conditions.
func TestService_SetPeriod_Race(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	svc := NewService(ctx, 10*time.Millisecond, time.Millisecond, nil)
	mon := NewMockMonitor("race", "race")
	svc.Add(ctx, mon)

	done := make(chan struct{})
	go func() {
		for i := range 100 {
			svc.SetPeriod(ctx, time.Duration(5+i)*time.Millisecond, 0)
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

// TestRAG_String verifies the string representations of RAG status values.
func TestRAG_String(t *testing.T) {
	tests := []struct {
		rag      RAG
		expected string
	}{
		{RAGUnknown, "Unknown"},
		{RAGGreen, "Green"},
		{RAGAmber, "Amber"},
		{RAGRed, "Red"},
		{RAGError, "Error"},
		{RAG(99), "Unknown"},
		{RAG(-1), "Unknown"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.rag.String(), "RAG(%d) string mismatch", tt.rag)
	}
}
