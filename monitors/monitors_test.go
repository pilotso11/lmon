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

func TestNewService(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	svc := NewService(ctx, 10*time.Millisecond, time.Millisecond, nil)
	assert.NotNil(t, svc, "service started")
	err := svc.Add(ctx, NewMockMonitor("test", "test"))
	assert.NoError(t, err, "monitor added")
	_, ok := svc.Monitors["test"]
	require.True(t, ok, "monitor added")
	time.Sleep(15 * time.Millisecond)
	svc.mu.Lock()
	defer svc.mu.Unlock()
	assert.Equal(t, 2, svc.Monitors["test"].(*MockMonitor).Checks, "checks called add + timer")
	assert.Equal(t, 1, len(svc.result), "len result")

	cancel()
}

func TestService_Add(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	svc := NewService(ctx, time.Second, time.Second, nil)
	m1 := NewMockMonitor("test1", "test")
	m2 := NewMockMonitor("test2", "test")

	err := svc.Add(ctx, m1)
	assert.NoError(t, err, "add should succeed")
	assert.Len(t, svc.Monitors, 1, "one monitor added")
	assert.Contains(t, svc.Monitors, "test1", "monitor test1 added")

	err = svc.Add(ctx, m2)
	assert.NoError(t, err, "add should succeed")
	assert.Len(t, svc.Monitors, 2, "two monitors added")
	assert.Contains(t, svc.Monitors, "test2", "monitor test2 added")

	// Test overwriting existing monitor
	m3 := NewMockMonitor("test1", "test")
	err = svc.Add(ctx, m3)
	assert.NoError(t, err, "add should succeed")
	assert.Len(t, svc.Monitors, 2, "monitor count unchanged")
	assert.Same(t, m3, svc.Monitors["test1"], "monitor was replaced")
}

func TestService_Remove(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	svc := NewService(ctx, time.Second, time.Second, nil)
	m1 := NewMockMonitor("test1", "test")
	m2 := NewMockMonitor("test2", "test")

	// Add monitors
	_ = svc.Add(ctx, m1)
	_ = svc.Add(ctx, m2)
	assert.Len(t, svc.Monitors, 2, "two monitors added")

	// Test successful removal
	err := svc.Remove(m1)
	assert.NoError(t, err, "remove should succeed")
	assert.Len(t, svc.Monitors, 1, "one monitor should remain")
	assert.NotContains(t, svc.Monitors, "test1", "monitor test1 should be removed")
	assert.Contains(t, svc.Monitors, "test2", "monitor test2 should remain")

	// Test removing non-existent monitor
	err = svc.Remove(m1)
	assert.Error(t, err, "removing non-existent monitor should fail")
	assert.ErrorAs(t, err, &ErrNotFound{}, "ErrNotFound")
	assert.Equal(t, err.Error(), "monitor test1 not found", "error message")

	// Test removing the last monitor
	err = svc.Remove(m2)
	assert.NoError(t, err, "remove should succeed")
	assert.Len(t, svc.Monitors, 0, "no monitors should remain")
}

func TestService_Results_CloneAndRace(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	svc := NewService(ctx, 10*time.Millisecond, time.Millisecond, nil)
	mon := NewMockMonitor("race", "race")
	_ = svc.Add(ctx, mon)

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

	svc := NewService(ctx, 50*time.Millisecond, time.Millisecond, nil)
	mon := NewMockMonitor("period", "period")
	_ = svc.Add(ctx, mon)

	// Wait for at least one check
	time.Sleep(60 * time.Millisecond)
	svc.mu.Lock()
	initialChecks := mon.Checks
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
	newChecks := mon.Checks
	svc.mu.Unlock()
	assert.Greater(t, newChecks, initialChecks+2, "checks should increase after period change")
	assert.NotEqual(t, initialPeriod, updatedPeriod, "period should have changed")
}

func TestService_SetPeriod_Race(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	svc := NewService(ctx, 10*time.Millisecond, time.Millisecond, nil)
	mon := NewMockMonitor("race", "race")
	_ = svc.Add(ctx, mon)

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
