// Package disk provides the Disk monitor implementation for filesystem usage checks.
// It supports both production and mock/test usage providers.
package disk

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v3/disk"

	"lmon/config"
	"lmon/monitors"
)

const Icon = "storage"     // Default icon for disk monitors
const Group = "filesystem" // Group name for disk monitors
const gigabyte = 1024 * 1024 * 1024

// UsageProvider is an interface for obtaining disk usage statistics.
// It allows for production and mock implementations.
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
type Disk struct {
	name      string        // Logical name for the disk monitor
	path      string        // Filesystem path to monitor
	threshold int           // Usage percentage threshold for alerting
	icon      string        // Icon for UI display
	impl      UsageProvider // Implementation for usage statistics
}

// NewDisk constructs a new Disk monitor with the given parameters.
// If icon is empty, the default Icon is used.
// If impl is nil, the DefaultDiskUsageProvider is used.
func NewDisk(name string, path string, threshold int, icon string, impl UsageProvider) Disk {
	if icon == "" {
		icon = Icon
	}
	if impl == nil {
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
	return fmt.Sprintf("disk_%s", d.name)
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
