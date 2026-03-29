package monitors

import (
	"time"

	"github.com/robfig/cron/v3"

	"lmon/config"
)

// cronParser is a standard 5-field cron parser (minute, hour, day-of-month, month, day-of-week).
var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// IsInMaintenanceWindow returns true if the given time falls within an active
// maintenance window defined by cfg. Returns false if cfg is nil, has an empty
// Cron expression, or has an invalid cron expression.
//
// Cron expressions are evaluated in UTC to avoid issues with DST transitions.
// During a spring-forward transition, local-time evaluation can skip triggers
// that fall in the missing hour, causing maintenance windows to be missed.
// Maintenance cron expressions in configuration files and UI fields must
// therefore be written in UTC; existing schedules that were specified in
// local time should be converted to UTC to preserve their original behavior.
func IsInMaintenanceWindow(cfg *config.MaintenanceConfig, now time.Time) bool {
	if cfg == nil || cfg.Cron == "" || cfg.Duration <= 0 {
		return false
	}
	schedule, err := cronParser.Parse(cfg.Cron)
	if err != nil {
		return false
	}
	duration := time.Duration(cfg.Duration) * time.Second

	// Evaluate in UTC to avoid DST gaps where schedule.Next() would skip
	// triggers that fall in a non-existent local hour.
	nowUTC := now.UTC()

	// Find the most recent cron trigger at or before now.
	// cron.Next(t) returns the first trigger strictly after t.
	// So Next(now - duration) gives us the earliest trigger that could still
	// have an active window covering now.
	candidate := schedule.Next(nowUTC.Add(-duration))
	return !candidate.After(nowUTC)
}
