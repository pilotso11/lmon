package docker

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"lmon/config"
	"lmon/monitors"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

const Icon = "box"     // Default icon for Docker monitors
const Group = "docker" // Group name for Docker monitors

// Provider interface for Docker API operations (allows mocking)
type Provider interface {
	// GetRestartCounts returns a map of container names to their restart counts
	GetRestartCounts(ctx context.Context, containerNames []string) (map[string]int, error)
	// RestartContainers restarts the specified containers
	RestartContainers(ctx context.Context, containerNames []string) error
}

// DefaultDockerProvider implements Provider using the Docker SDK
type DefaultDockerProvider struct {
	client *client.Client
}

// NewDefaultDockerProvider creates a new DefaultDockerProvider
func NewDefaultDockerProvider() (*DefaultDockerProvider, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	return &DefaultDockerProvider{client: cli}, nil
}

// GetRestartCounts retrieves restart counts for the specified containers
func (d *DefaultDockerProvider) GetRestartCounts(ctx context.Context, containerNames []string) (map[string]int, error) {
	results := make(map[string]int)

	for _, name := range containerNames {
		// Inspect container to get restart count
		containerJSON, err := d.client.ContainerInspect(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("failed to inspect container %s: %w", name, err)
		}

		results[name] = containerJSON.RestartCount
	}

	return results, nil
}

// RestartContainers restarts the specified containers
func (d *DefaultDockerProvider) RestartContainers(ctx context.Context, containerNames []string) error {
	for _, name := range containerNames {
		timeout := 10 // 10 second timeout for graceful shutdown
		options := container.StopOptions{
			Timeout: &timeout,
		}
		if err := d.client.ContainerRestart(ctx, name, options); err != nil {
			return fmt.Errorf("failed to restart container %s: %w", name, err)
		}
	}
	return nil
}

// Close closes the Docker client connection
func (d *DefaultDockerProvider) Close() error {
	if d.client != nil {
		return d.client.Close()
	}
	return nil
}

// Monitor represents a Docker container monitor
type Monitor struct {
	name                     string
	containers               []string // List of container names/IDs to monitor
	threshold                int      // Max restart count threshold
	icon                     string
	impl                     Provider
	allowedRestartContainers []string // Global whitelist of containers allowed for restart
	alertThreshold           int      // Number of consecutive failures before triggering alert
}

// NewMonitor creates a new Docker monitor
func NewMonitor(name string, containers string, threshold int, icon string, alertThreshold int, allowedRestartContainers []string, impl Provider) (Monitor, error) {
	if icon == "" {
		icon = Icon
	}

	// Parse container list (support both space and comma separation)
	containerList := ParseContainerList(containers)
	if len(containerList) == 0 {
		return Monitor{}, fmt.Errorf("no containers specified")
	}

	// Create default provider if none provided
	if impl == nil {
		var err error
		impl, err = NewDefaultDockerProvider()
		if err != nil {
			return Monitor{}, err
		}
	}

	if alertThreshold <= 0 {
		alertThreshold = 1
	}

	return Monitor{
		name:                     name,
		containers:               containerList,
		threshold:                threshold,
		icon:                     icon,
		impl:                     impl,
		alertThreshold:           alertThreshold,
		allowedRestartContainers: allowedRestartContainers,
	}, nil
}

// ParseContainerList splits a container list by spaces and commas
func ParseContainerList(containers string) []string {
	// Split by comma or space
	parts := strings.FieldsFunc(containers, func(r rune) bool {
		return r == ',' || r == ' '
	})

	// Trim whitespace from each part
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// Name returns the unique name/ID of the monitor
func (m Monitor) Name() string {
	return fmt.Sprintf("%s_%s", Group, m.name)
}

// DisplayName returns the human-readable name for display
func (m Monitor) DisplayName() string {
	if len(m.containers) == 1 {
		return m.containers[0]
	}
	return fmt.Sprintf("%s (%d containers)", m.name, len(m.containers))
}

// Group returns the group/category of the monitor
func (m Monitor) Group() string {
	return Group
}

// Check performs a check and returns the result
func (m Monitor) Check(ctx context.Context) monitors.Result {
	restartCounts, err := m.impl.GetRestartCounts(ctx, m.containers)
	if err != nil {
		return monitors.Result{
			Status:      monitors.RAGError,
			Value:       fmt.Sprintf("error: %v", err),
			Group:       Group,
			DisplayName: m.DisplayName(),
		}
	}

	// Find the maximum restart count across all containers
	maxRestarts := 0
	var details []string
	for name, count := range restartCounts {
		if count > maxRestarts {
			maxRestarts = count
		}
		details = append(details, fmt.Sprintf("%s: %d", name, count))
	}

	// Determine RAG status based on threshold
	// Green: Below 90% of threshold
	// Amber: Between 90% and threshold
	// Red: At or above threshold
	status := monitors.RAGGreen
	switch {
	case maxRestarts >= m.threshold:
		status = monitors.RAGRed
	case maxRestarts >= int(float64(m.threshold)*0.9):
		status = monitors.RAGAmber
	}

	// Format output
	value := fmt.Sprintf("Max restarts: %d", maxRestarts)
	value2 := strings.Join(details, ", ")

	return monitors.Result{
		Status:      status,
		Value:       value,
		Value2:      value2,
		Group:       Group,
		DisplayName: m.DisplayName(),
	}
}

// Save saves the monitor configuration
func (m Monitor) Save(cfg *config.Config) {
	cfg.Monitoring.Docker[m.name] = config.DockerConfig{
		Containers:     strings.Join(m.containers, ", "),
		Threshold:      m.threshold,
		Icon:           m.icon,
		AlertThreshold: m.alertThreshold,
	}
}

// AlertThreshold returns the number of consecutive failures before triggering an alert
func (m Monitor) AlertThreshold() int {
	return m.alertThreshold
}

// Restart restarts all containers monitored by this monitor
// Only containers in the allowedRestartContainers whitelist can be restarted
func (m Monitor) Restart(ctx context.Context) error {
	// Filter containers to only those in the allowed list
	containersToRestart := filterAllowedContainers(m.containers, m.allowedRestartContainers)
	
	// If no allowed containers, return an error
	if len(containersToRestart) == 0 {
		return fmt.Errorf("no containers in the restart list are allowed by the global whitelist")
	}
	
	// Log if some containers were skipped
	if len(containersToRestart) < len(m.containers) {
		skipped := findSkippedContainers(m.containers, containersToRestart)
		log.Printf("Warning: Skipping restart of containers not in allowedRestartContainers whitelist: %v", skipped)
	}
	
	return m.impl.RestartContainers(ctx, containersToRestart)
}

// filterAllowedContainers returns only the containers that are in the allowed list
// If allowedList is empty, all containers are allowed (for backward compatibility)
func filterAllowedContainers(containers []string, allowedList []string) []string {
	// If no whitelist is configured, allow all containers
	if len(allowedList) == 0 {
		return containers
	}
	
	// Create a map for fast lookup
	allowed := make(map[string]bool)
	for _, c := range allowedList {
		allowed[c] = true
	}
	
	// Filter containers
	var result []string
	for _, c := range containers {
		if allowed[c] {
			result = append(result, c)
		}
	}
	
	return result
}

// findSkippedContainers returns containers that were in the original list but not in the filtered list
func findSkippedContainers(original []string, filtered []string) []string {
	// Create a map for fast lookup
	included := make(map[string]bool)
	for _, c := range filtered {
		included[c] = true
	}
	
	// Find skipped containers
	var skipped []string
	for _, c := range original {
		if !included[c] {
			skipped = append(skipped, c)
		}
	}
	
	return skipped
}

// Close closes the Docker client connection if the provider implements io.Closer
func (m Monitor) Close() error {
	if closer, ok := m.impl.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
