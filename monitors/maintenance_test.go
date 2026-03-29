package monitors

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"lmon/config"
)

func TestIsInMaintenanceWindow_NilConfig(t *testing.T) {
	assert.False(t, IsInMaintenanceWindow(nil, time.Now()))
}

func TestIsInMaintenanceWindow_EmptyCron(t *testing.T) {
	cfg := &config.MaintenanceConfig{Cron: "", Duration: 60}
	assert.False(t, IsInMaintenanceWindow(cfg, time.Now()))
}

func TestIsInMaintenanceWindow_ZeroDuration(t *testing.T) {
	cfg := &config.MaintenanceConfig{Cron: "* * * * *", Duration: 0}
	assert.False(t, IsInMaintenanceWindow(cfg, time.Now()))
}

func TestIsInMaintenanceWindow_NegativeDuration(t *testing.T) {
	cfg := &config.MaintenanceConfig{Cron: "* * * * *", Duration: -1}
	assert.False(t, IsInMaintenanceWindow(cfg, time.Now()))
}

func TestIsInMaintenanceWindow_InvalidCron(t *testing.T) {
	cfg := &config.MaintenanceConfig{Cron: "not a cron", Duration: 60}
	assert.False(t, IsInMaintenanceWindow(cfg, time.Now()))
}

func TestIsInMaintenanceWindow_EveryMinute_InWindow(t *testing.T) {
	// Cron fires every minute. Duration 30s. At second 15 we should be in the window.
	cfg := &config.MaintenanceConfig{Cron: "* * * * *", Duration: 30}
	now := time.Date(2026, 3, 28, 10, 5, 15, 0, time.UTC) // 10:05:15
	assert.True(t, IsInMaintenanceWindow(cfg, now))
}

func TestIsInMaintenanceWindow_EveryMinute_OutsideWindow(t *testing.T) {
	// Cron fires every minute. Duration 30s. At second 45 we should be outside.
	cfg := &config.MaintenanceConfig{Cron: "* * * * *", Duration: 30}
	now := time.Date(2026, 3, 28, 10, 5, 45, 0, time.UTC) // 10:05:45
	assert.False(t, IsInMaintenanceWindow(cfg, now))
}

func TestIsInMaintenanceWindow_Every4Hours_InWindow(t *testing.T) {
	// Cron fires at minute 0 every 4 hours. Duration 60s.
	// At 08:00:30 we should be in the window.
	cfg := &config.MaintenanceConfig{Cron: "0 */4 * * *", Duration: 60}
	now := time.Date(2026, 3, 28, 8, 0, 30, 0, time.UTC)
	assert.True(t, IsInMaintenanceWindow(cfg, now))
}

func TestIsInMaintenanceWindow_Every4Hours_OutsideWindow(t *testing.T) {
	// At 08:01:30 we should be outside the 60s window that started at 08:00:00.
	cfg := &config.MaintenanceConfig{Cron: "0 */4 * * *", Duration: 60}
	now := time.Date(2026, 3, 28, 8, 1, 30, 0, time.UTC)
	assert.False(t, IsInMaintenanceWindow(cfg, now))
}

func TestIsInMaintenanceWindow_ExactStart(t *testing.T) {
	// Exactly at the cron trigger time — should be in window.
	cfg := &config.MaintenanceConfig{Cron: "30 10 * * *", Duration: 120}
	now := time.Date(2026, 3, 28, 10, 30, 0, 0, time.UTC)
	assert.True(t, IsInMaintenanceWindow(cfg, now))
}

func TestIsInMaintenanceWindow_ExactEnd(t *testing.T) {
	// Exactly at cron trigger + duration — should be outside (window is half-open).
	cfg := &config.MaintenanceConfig{Cron: "30 10 * * *", Duration: 120}
	now := time.Date(2026, 3, 28, 10, 32, 0, 0, time.UTC)
	assert.False(t, IsInMaintenanceWindow(cfg, now))
}

func TestIsInMaintenanceWindow_OneSecondBeforeEnd(t *testing.T) {
	cfg := &config.MaintenanceConfig{Cron: "30 10 * * *", Duration: 120}
	now := time.Date(2026, 3, 28, 10, 31, 59, 0, time.UTC)
	assert.True(t, IsInMaintenanceWindow(cfg, now))
}

func TestIsInMaintenanceWindow_OneSecondAfterEnd(t *testing.T) {
	cfg := &config.MaintenanceConfig{Cron: "30 10 * * *", Duration: 120}
	now := time.Date(2026, 3, 28, 10, 32, 1, 0, time.UTC)
	assert.False(t, IsInMaintenanceWindow(cfg, now))
}

func TestIsInMaintenanceWindow_MidnightCrossing(t *testing.T) {
	// Cron fires at 23:59, duration 120s — window spans midnight.
	cfg := &config.MaintenanceConfig{Cron: "59 23 * * *", Duration: 120}
	now := time.Date(2026, 3, 29, 0, 0, 30, 0, time.UTC) // 30s after midnight
	assert.True(t, IsInMaintenanceWindow(cfg, now))
}

func TestIsInMaintenanceWindow_DSTSpringForward_InWindow(t *testing.T) {
	// Simulate BST spring-forward: clocks go from 1:00 GMT to 2:00 BST.
	// Cron fires at 1:30 UTC. With UTC evaluation, the trigger fires correctly
	// even though 1:30 local doesn't exist during spring-forward.
	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Skip("Europe/London timezone not available")
	}

	cfg := &config.MaintenanceConfig{Cron: "30 1 * * *", Duration: 120}
	// 2026-03-29 is BST spring-forward day in UK.
	// Construct in UTC to avoid non-existent local time, then convert to London.
	now := time.Date(2026, 3, 29, 1, 31, 0, 0, time.UTC).In(london)
	assert.True(t, IsInMaintenanceWindow(cfg, now))
}

func TestIsInMaintenanceWindow_DSTSpringForward_OutsideWindow(t *testing.T) {
	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Skip("Europe/London timezone not available")
	}

	cfg := &config.MaintenanceConfig{Cron: "30 1 * * *", Duration: 120}
	// At 1:33 UTC (2:33 BST), 3 minutes after trigger, outside the 120s window.
	now := time.Date(2026, 3, 29, 1, 33, 0, 0, time.UTC).In(london)
	assert.False(t, IsInMaintenanceWindow(cfg, now))
}

func TestIsInMaintenanceWindow_DSTFallBack_InWindow(t *testing.T) {
	// Simulate BST fall-back: clocks go from 2:00 BST back to 1:00 GMT.
	// The hour 1:00-2:00 occurs twice. Cron at 1:30 UTC should still work.
	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Skip("Europe/London timezone not available")
	}

	cfg := &config.MaintenanceConfig{Cron: "30 1 * * *", Duration: 120}
	// 2026-10-25 is BST fall-back day in UK.
	// Construct in UTC to avoid ambiguous local time, then convert to London.
	now := time.Date(2026, 10, 25, 1, 31, 0, 0, time.UTC).In(london)
	assert.True(t, IsInMaintenanceWindow(cfg, now))
}

func TestIsInMaintenanceWindow_LocalTimeConvertedToUTC(t *testing.T) {
	// Verify that a time passed in a non-UTC timezone is correctly
	// converted to UTC for evaluation.
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Skip("Asia/Tokyo timezone not available")
	}

	// Cron fires at 0:00 UTC. Tokyo is UTC+9, so 9:00 JST = 0:00 UTC.
	cfg := &config.MaintenanceConfig{Cron: "0 0 * * *", Duration: 120}
	// 9:01 JST = 0:01 UTC, should be in the 120s window.
	now := time.Date(2026, 3, 28, 9, 1, 0, 0, tokyo)
	assert.True(t, IsInMaintenanceWindow(cfg, now))
}
