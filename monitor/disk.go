package monitor

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
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

// ZFSPoolHealth represents the health of a ZFS pool
type ZFSPoolHealth struct {
	Pool   string
	Status string
}

// ZFSVolumeChecker is a function type for checking if a path is a ZFS volume
type ZFSVolumeChecker func(path string) bool

// ZFSPoolHealthChecker is a function type for getting ZFS pool health
type ZFSPoolHealthChecker func(path string) (*ZFSPoolHealth, error)

// DiskMonitor represents a disk space monitor
type DiskMonitor struct {
	config           *config.Config
	usageProvider    DiskUsageProvider
	isZFSVolume      ZFSVolumeChecker
	getZFSPoolHealth ZFSPoolHealthChecker
}

// NewDiskMonitor creates a new disk monitor
func NewDiskMonitor(cfg *config.Config) *DiskMonitor {
	return &DiskMonitor{
		config:           cfg,
		usageProvider:    &DefaultDiskUsageProvider{},
		isZFSVolume:      isZFSVolume,
		getZFSPoolHealth: getZFSPoolHealth,
	}
}

// NewDiskMonitorWithProvider creates a new disk monitor with a custom usage provider
func NewDiskMonitorWithProvider(cfg *config.Config, provider DiskUsageProvider) *DiskMonitor {
	return &DiskMonitor{
		config:           cfg,
		usageProvider:    provider,
		isZFSVolume:      isZFSVolume,
		getZFSPoolHealth: getZFSPoolHealth,
	}
}

// NewDiskMonitorWithCustomFuncs creates a new disk monitor with custom ZFS functions
func NewDiskMonitorWithCustomFuncs(cfg *config.Config, provider DiskUsageProvider,
	isZFSVolume ZFSVolumeChecker, getZFSPoolHealth ZFSPoolHealthChecker) *DiskMonitor {
	return &DiskMonitor{
		config:           cfg,
		usageProvider:    provider,
		isZFSVolume:      isZFSVolume,
		getZFSPoolHealth: getZFSPoolHealth,
	}
}

// isZFSVolume checks if a path is a ZFS volume
func isZFSVolume(path string) bool {
	cmd := exec.Command("zfs", "list", "-H", "-o", "name", path)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(output))) > 0
}

// getZFSPoolHealth gets the health of a ZFS pool
func getZFSPoolHealth(path string) (*ZFSPoolHealth, error) {
	// First, get the pool name from the path
	cmd := exec.Command("zfs", "list", "-H", "-o", "name", path)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get ZFS dataset name: %w", err)
	}

	dataset := strings.TrimSpace(string(output))
	if dataset == "" {
		return nil, fmt.Errorf("no ZFS dataset found for path: %s", path)
	}

	// Extract pool name (everything before the first '/')
	poolName := dataset
	if idx := strings.Index(dataset, "/"); idx > 0 {
		poolName = dataset[:idx]
	}

	// Get pool health
	cmd = exec.Command("zpool", "status", "-H", "-p", poolName)
	output, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get ZFS pool status: %w", err)
	}

	// Parse the output to get the health status
	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("unexpected zpool status output format")
	}

	// The health status is typically in the second line, third field
	fields := strings.Fields(lines[1])
	if len(fields) < 3 {
		return nil, fmt.Errorf("unexpected zpool status line format")
	}

	return &ZFSPoolHealth{
		Pool:   poolName,
		Status: fields[2],
	}, nil
}

// Check checks disk space usage
func (m *DiskMonitor) Check() ([]*Item, error) {
	var items []*Item

	for _, diskCfg := range m.config.Monitoring.Disk {
		// Get disk usage statistics
		usage, err := m.usageProvider.Usage(diskCfg.Path)
		if err != nil {
			// If we can't get disk usage, report the disk as missing but continue with other disks
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
				Status:    StatusCritical,
				Value:     0,
				Threshold: float64(diskCfg.Threshold),
				Unit:      "%",
				Icon:      diskCfg.Icon,
				LastCheck: time.Now(),
				Message:   fmt.Sprintf("Disk not accessible: %v", err),
			}

			items = append(items, item)
			continue
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

		// Check if this is a ZFS volume
		message := fmt.Sprintf("%.2f%% used (%.1f GB / %.1f GB)", usagePercent, float64(usage.Used)/1024/1024/1024, float64(usage.Total)/1024/1024/1024)

		if m.isZFSVolume(diskCfg.Path) {
			// Get ZFS pool health
			poolHealth, err := m.getZFSPoolHealth(diskCfg.Path)
			if err == nil {
				message = fmt.Sprintf("%s - ZFS Pool '%s' health: %s", message, poolHealth.Pool, poolHealth.Status)

				// Update status based on pool health
				if poolHealth.Status != "ONLINE" {
					status = StatusCritical
				}
			}
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
			Message:   message,
		}

		items = append(items, item)
	}

	return items, nil
}
