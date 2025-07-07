package disk

import (
	"fmt"
	"os/exec"
	"strings"
)

// ZFSPoolHealth represents the health of a ZFS pool
type ZFSPoolHealth struct {
	Pool   string
	Status string
}

// if a path is a ZFS volume
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
