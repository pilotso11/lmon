package monitors

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

// TestService_AlertThreshold_DefaultBehavior verifies alerts trigger on first failure with default threshold of 1.
func TestService_AlertThreshold_DefaultBehavior(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	push := NewMockPush()
	svc := NewService(ctx, 50*time.Millisecond, 10*time.Millisecond, push.Push)

	// Create a monitor that will fail then succeed
	m := NewMockMonitor("test1", "test")
	m.status = []struct {
		rag RAG
		msg string
	}{
		{RAGRed, "fail 1"},
		{RAGGreen, "success"},
	}

	svc.Add(ctx, m)

	// Wait for checks to run
	assert.Eventually(t, func() bool {
		return m.Checks.Load() >= 2
	}, 300*time.Millisecond, 10*time.Millisecond, "expected at least 2 checks")

	// With default threshold of 1, should get push on first failure and on recovery
	assert.Eventually(t, func() bool {
		return push.Calls.Size() >= 2
	}, 200*time.Millisecond, 10*time.Millisecond, "expected 2 pushes (failure + recovery)")
}

// TestService_AlertThreshold_RecoveryResetsCount verifies that recovery resets the failure count.
func TestService_AlertThreshold_RecoveryResetsCount(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	push := NewMockPush()
	svc := NewService(ctx, 50*time.Millisecond, 10*time.Millisecond, push.Push)

	m := NewMockMonitor("test1", "test")
	m.status = []struct {
		rag RAG
		msg string
	}{
		{RAGRed, "fail 1"},
		{RAGGreen, "success"}, // Recovery should reset count  
		{RAGRed, "fail again"},
	}

	svc.Add(ctx, m)

	// Wait for all checks
	assert.Eventually(t, func() bool {
		return m.Checks.Load() >= 3
	}, 300*time.Millisecond, 10*time.Millisecond, "expected at least 3 checks")

	// Should have:
	// 1. Push for first failure (threshold=1)
	// 2. Push for recovery
	// 3. Push for failure after recovery (count was reset)
	assert.Eventually(t, func() bool {
		return push.Calls.Size() >= 3
	}, 200*time.Millisecond, 10*time.Millisecond, "expected 3 pushes")
}

// TestService_AlertThreshold_ConsecutiveFailures verifies consecutive failure tracking.
func TestService_AlertThreshold_ConsecutiveFailures(t *testing.T) {
	defer goleak.VerifyNone(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	push := NewMockPush()
	svc := NewService(ctx, 50*time.Millisecond, 10*time.Millisecond, push.Push)

	// Create a monitor that stays red for multiple checks
	m := NewMockMonitor("test1", "test")
	m.status = []struct {
		rag RAG
		msg string
	}{
		{RAGRed, "fail 1"},
		{RAGRed, "fail 2"},
		{RAGRed, "fail 3"},
	}

	svc.Add(ctx, m)

	// Wait for checks
	assert.Eventually(t, func() bool {
		return m.Checks.Load() >= 3
	}, 300*time.Millisecond, 10*time.Millisecond, "expected at least 3 checks")

	// The monitor will get checked once on Add() and then periodically
	// We should get 1 push when it first goes to Red (from Unknown on the Add check or from the first periodic check)
	// Subsequent checks that remain Red don't change status, so no new pushes
	time.Sleep(100 * time.Millisecond)
	count := push.Calls.Size()
	// Could be 1 or 2 depending on timing of Add vs periodic checks
	// The key is it shouldn't keep growing with each check
	assert.LessOrEqual(t, count, 2, "should have at most 2 pushes")
	assert.GreaterOrEqual(t, count, 1, "should have at least 1 push")
}

