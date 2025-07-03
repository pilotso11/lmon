package monitor

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
	"lmon/config"
)

// DiskUsageProvider is an interface for getting disk usage
type DiskUsageProvider interface {
	Usage(path string) (*disk.UsageStat, error)
}

// DefaultDiskUsageProvider is the default implementation of DiskUsageProvider
type DefaultDiskUsageProvider struct{}

// Usage returns disk usage statistics
func (p *DefaultDiskUsageProvider) Usage(path string) (*disk.UsageStat, error) {
	return disk.Usage(path)
}

// DiskMonitor represents a disk space monitor
type DiskMonitor struct {
	config        *config.Config
	usageProvider DiskUsageProvider
}

// NewDiskMonitor creates a new disk monitor
func NewDiskMonitor(cfg *config.Config) *DiskMonitor {
	return &DiskMonitor{
		config:        cfg,
		usageProvider: &DefaultDiskUsageProvider{},
	}
}

// NewDiskMonitorWithProvider creates a new disk monitor with a custom usage provider
func NewDiskMonitorWithProvider(cfg *config.Config, provider DiskUsageProvider) *DiskMonitor {
	return &DiskMonitor{
		config:        cfg,
		usageProvider: provider,
	}
}

// Check checks disk space usage
func (m *DiskMonitor) Check() ([]*Item, error) {
	var items []*Item

	for _, diskCfg := range m.config.Monitoring.Disk {
		// Get disk usage statistics
		usage, err := m.usageProvider.Usage(diskCfg.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to get disk usage for %s: %w", diskCfg.Path, err)
		}

		// Calculate usage percentage
		usagePercent := usage.UsedPercent

		// Determine status based on threshold
		status := StatusOK
		if usagePercent >= float64(diskCfg.Threshold) {
			status = StatusCritical
		} else if usagePercent >= float64(diskCfg.Threshold)*0.8 {
			status = StatusWarning
		}

		// Create item
		name := filepath.Base(diskCfg.Path)
		if name == "" || name == "." {
			name = diskCfg.Path
		}
		if name == "/" {
			name = "Root"
		}

		item := &Item{
			ID:        fmt.Sprintf("disk-%s", diskCfg.Path),
			Name:      fmt.Sprintf("Disk (%s)", name),
			Type:      "disk",
			Status:    status,
			Value:     usagePercent,
			Threshold: float64(diskCfg.Threshold),
			Unit:      "%",
			Icon:      diskCfg.Icon,
			LastCheck: time.Now(),
			Message:   fmt.Sprintf("%.2f%% used (%.1f GB / %.1f GB)", usagePercent, float64(usage.Used)/1024/1024/1024, float64(usage.Total)/1024/1024/1024),
		}

		items = append(items, item)
	}

	return items, nil
}
