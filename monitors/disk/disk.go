// Package disk provides the Disk monitor implementation for filesystem usage checks.
// It supports both production and mock/test usage providers.
//
// # Disk Monitor Overview
//
// The Disk monitor checks filesystem usage for a specified path and alerts if usage exceeds a configured threshold.
//
// ## How it works
//   - Uses a UsageProvider interface to abstract disk usage retrieval (default: gopsutil).
//   - Each disk monitor is configured with:
//   - name: Logical name for the disk.
//   - path: Filesystem path to monitor.
//   - threshold: Usage percentage threshold for alerting.
//   - icon: UI icon (optional).
//   - On each check:
//   - Retrieves usage stats for the path.
//   - Calculates percentage used and formats a result string.
//   - Status is:
//   - Green: Below 90% of threshold.
//   - Amber: Between 90% of threshold and threshold.
//   - Red: At or above threshold.
//   - Configuration is persisted back to the config struct for saving.
//
// ## Example Usage
//
//	diskMonitor := NewDisk("root", "/", 80, "", nil)
//	result := diskMonitor.Check(context.Background())
//
// See the NewDisk and Disk.Check methods for implementation details.
package disk

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v3/disk"

	"lmon/common"
	"lmon/config"
	"lmon/monitors"
)

const Icon = "hdd"   // Default icon for disk monitors
const Group = "disk" // Group name for disk monitors
const gigabyte = 1024 * 1024 * 1024

// UsageProvider is an interface for obtaining disk usage statistics.
// It allows for production and mock implementations.
//
// Implementations should return usage statistics for the given filesystem path.
type UsageProvider interface {
	Usage(path string) (*disk.UsageStat, error)
}

// DefaultDiskUsageProvider is the default implementation of UsageProvider
// using gopsutil's disk.Usage.
type DefaultDiskUsageProvider struct{}

// Usage returns disk usage statistics for the given path.
func (p *DefaultDiskUsageProvider) Usage(path string) (*disk.UsageStat, error) {
	return disk.Usage(path)
}

// Disk represents a filesystem usage monitor.
//
// Fields:
//   - name: Logical name for the disk monitor.
//   - path: Filesystem path to monitor.
//   - threshold: Usage percentage threshold for alerting.
//   - icon: Icon for UI display.
//   - impl: Implementation for usage statistics (can be mocked for testing).
type Disk struct {
	name      string        // Logical name for the disk monitor
	path      string        // Filesystem path to monitor
	threshold int           // Usage percentage threshold for alerting
	icon      string        // Icon for UI display
	impl      UsageProvider // Implementation for usage statistics
}

// NewDisk constructs a new Disk monitor with the given parameters.
//
// If icon is empty, the default Icon is used.
// If impl is nil, the DefaultDiskUsageProvider is used.
//
// Example:
//
//	diskMonitor := NewDisk("root", "/", 80, "", nil)
func NewDisk(name string, path string, threshold int, icon string, impl UsageProvider) Disk {
	if icon == "" {
		icon = Icon
	}
	if common.IsNil(impl) {
		// todo: is the filesystem zfs?
		impl = &DefaultDiskUsageProvider{}
	}
	return Disk{
		name:      name,
		path:      path,
		threshold: threshold,
		icon:      icon,
		impl:      impl,
	}
}

// String returns a string representation of the Disk monitor.
func (d Disk) String() string {
	return fmt.Sprintf("Disk{name: %s, path: %s, threshold: %d, icon: %s}", d.name, d.path, d.threshold, d.icon)
}

// DisplayName returns a human-readable name for the disk monitor.
func (d Disk) DisplayName() string {
	return fmt.Sprintf("%s (%s)", d.name, d.path)
}

// Group returns the group/category for the disk monitor.
func (d Disk) Group() string {
	return Group
}

// Name returns the unique name/ID for the disk monitor.
func (d Disk) Name() string {
	return fmt.Sprintf("%s_%s", Group, d.name)
}

// Save persists the disk monitor's configuration to the provided config struct.
func (d Disk) Save(cfg *config.Config) {
	cfg.Monitoring.Disk[d.name] = config.DiskConfig{
		Path:      d.path,
		Threshold: d.threshold,
		Icon:      d.icon,
	}
}

// Check performs a usage check on the disk and returns a Result.
// It uses the configured UsageProvider implementation.
//
// The result status is:
//   - RAGGreen: Usage is below 90% of threshold.
//   - RAGAmber: Usage is between 90% of threshold and threshold.
//   - RAGRed: Usage is at or above threshold.
//   - RAGError: If disk usage cannot be determined.
func (d Disk) Check(_ context.Context) monitors.Result {
	usage, err := d.impl.Usage(d.path)
	if err != nil {
		return monitors.Result{
			Status: monitors.RAGError,
			Value:  fmt.Sprintf("error getting disk usage: %v", err),
		}
	}

	total := float64(usage.Total) / gigabyte
	used := total * usage.UsedPercent / 100.0
	res := fmt.Sprintf("%.1f%% used (%.1f GB / %.1f GB)", usage.UsedPercent, used, total)
	status := monitors.RAGGreen
	switch {
	case usage.UsedPercent >= float64(d.threshold):
		status = monitors.RAGRed
	case usage.UsedPercent >= float64(d.threshold)*0.9:
		status = monitors.RAGAmber
	}

	return monitors.Result{
		Status: status,
		Value:  res,
	}
}
