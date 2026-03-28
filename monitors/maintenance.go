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
func IsInMaintenanceWindow(cfg *config.MaintenanceConfig, now time.Time) bool {
	if cfg == nil || cfg.Cron == "" || cfg.Duration <= 0 {
		return false
	}
	schedule, err := cronParser.Parse(cfg.Cron)
	if err != nil {
		return false
	}
	duration := time.Duration(cfg.Duration) * time.Second

	// Find the most recent cron trigger at or before now.
	// cron.Next(t) returns the first trigger strictly after t.
	// So Next(now - duration) gives us the earliest trigger that could still
	// have an active window covering now.
	candidate := schedule.Next(now.Add(-duration))
	return !candidate.After(now)
}
