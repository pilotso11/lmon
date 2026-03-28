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
