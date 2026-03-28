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
	m.SetStatuses([]MockStatus{
		{RAGRed, "fail 1"},
		{RAGGreen, "success"},
	})

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
	m.SetStatuses([]MockStatus{
		{RAGRed, "fail 1"},
		{RAGGreen, "success"}, // Recovery should reset count
		{RAGRed, "fail again"},
	})

	svc.Add(ctx, m)

	// Wait for all checks
	assert.Eventually(t, func() bool {
		return m.Checks.Load() >= 3
	}, 500*time.Millisecond, 10*time.Millisecond, "expected at least 3 checks")

	// Should have:
	// 1. Push for first failure (threshold=1)
	// 2. Push for recovery
	// 3. Push for failure after recovery (count was reset)
	assert.Eventually(t, func() bool {
		return push.Calls.Size() >= 3
	}, 500*time.Millisecond, 10*time.Millisecond, "expected 3 pushes")
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
	m.SetAlertThreshold(3) // Require 3 consecutive failures before alerting
	m.SetStatuses([]MockStatus{
		{RAGRed, "fail 1"},
		{RAGRed, "fail 2"},
		{RAGRed, "fail 3"},
		{RAGRed, "fail 4"},
		{RAGRed, "fail 5"},
		{RAGRed, "fail 6"},
	})

	svc.Add(ctx, m)

	// Wait for checks
	assert.Eventually(t, func() bool {
		return m.Checks.Load() >= 3
	}, 300*time.Millisecond, 10*time.Millisecond, "expected at least 3 checks")

	// With alert threshold of 3, we should get exactly 1 push when the 3rd failure occurs
	time.Sleep(100 * time.Millisecond)
	count := push.Calls.Size()
	assert.Equal(t, 1, count, "should have exactly 1 push when threshold of 3 is reached")
}
