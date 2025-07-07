package disk

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v3/disk"

	"lmon/config"
	"lmon/monitors"
)

const Icon = "storage"
const Group = "filesystem"
const gigabyte = 1024 * 1024 * 1024

// UsageProvider is an interface for getting disk usage
type UsageProvider interface {
	Usage(path string) (*disk.UsageStat, error)
}

// DefaultDiskUsageProvider is the default implementation of DiskUsageProvider
type DefaultDiskUsageProvider struct{}

// Usage returns disk usage statistics
func (p *DefaultDiskUsageProvider) Usage(path string) (*disk.UsageStat, error) {
	return disk.Usage(path)
}

type Disk struct {
	name      string
	path      string
	threshold int
	icon      string
	impl      UsageProvider
}

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

func (d Disk) DisplayName() string {
	return fmt.Sprintf("%s (%s)", d.name, d.path)
}

func (d Disk) Group() string {
	return Group
}

func (d Disk) Name() string {
	return fmt.Sprintf("disk_%s", d.name)
}

func (d Disk) Save(cfg *config.Config) {
	cfg.Monitoring.Disk[d.name] = config.DiskConfig{
		Path:      d.path,
		Threshold: d.threshold,
		Icon:      d.icon,
	}
}

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
