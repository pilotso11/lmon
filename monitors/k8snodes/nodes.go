// Package k8snodes provides a Kubernetes Nodes monitor implementation.
// It checks the status of Kubernetes nodes and reports RAG status based on
// node readiness and pressure conditions.
//
// # K8s Nodes Monitor
//
// ## How it works:
//   - Uses a Provider interface to abstract Kubernetes API calls.
//   - Configured with:
//   - name: Logical name for the nodes monitor.
//   - icon: UI icon (optional, defaults to "hdd-rack").
//   - On each check:
//   - Fetches node statuses from the cluster.
//   - Status is:
//   - Green: All nodes Ready and no pressure conditions.
//   - Amber: Any node has pressure conditions or is cordoned.
//   - Red: Any node is NotReady.
//   - Error: Provider returned an error.
package k8snodes

import (
	"context"
	"fmt"
	"strings"

	"lmon/config"
	"lmon/monitors"
)

// Default values
const Icon = "hdd-rack"   // Default icon for k8s nodes monitors
const Group = "k8snodes" // Group name for k8s nodes monitors

// NodeStatus represents the status of a Kubernetes node.
type NodeStatus struct {
	Name           string
	Ready          bool
	Cordoned       bool
	MemoryPressure bool
	DiskPressure   bool
	PIDPressure    bool
}

// Provider is an interface for fetching Kubernetes node statuses.
type Provider interface {
	GetNodeStatuses(ctx context.Context) ([]NodeStatus, error)
}

// NoopProvider is a no-op provider when no real k8s client is available.
type NoopProvider struct{}

// GetNodeStatuses returns no statuses and no error.
func (n *NoopProvider) GetNodeStatuses(_ context.Context) ([]NodeStatus, error) {
	return nil, nil
}

// Monitor represents a Kubernetes Nodes monitor.
type Monitor struct {
	name           string
	icon           string
	alertThreshold int
	impl           Provider
}

// NewMonitor constructs a new k8s nodes Monitor.
func NewMonitor(name string, icon string, alertThreshold int, impl Provider) Monitor {
	if icon == "" {
		icon = Icon
	}
	if alertThreshold <= 0 {
		alertThreshold = 1
	}
	if impl == nil {
		impl = &NoopProvider{}
	}
	return Monitor{
		name:           name,
		icon:           icon,
		alertThreshold: alertThreshold,
		impl:           impl,
	}
}

// Check performs the node status check and returns the monitor result.
func (m Monitor) Check(ctx context.Context) monitors.Result {
	nodes, err := m.impl.GetNodeStatuses(ctx)
	if err != nil {
		return monitors.Result{
			Status:      monitors.RAGError,
			Value:       fmt.Sprintf("error: %v", err),
			Group:       Group,
			DisplayName: m.DisplayName(),
		}
	}

	if len(nodes) == 0 {
		return monitors.Result{
			Status:      monitors.RAGGreen,
			Value:       "0 nodes",
			Group:       Group,
			DisplayName: m.DisplayName(),
		}
	}

	status := monitors.RAGGreen
	var issues []string

	for _, node := range nodes {
		if !node.Ready {
			status = monitors.RAGRed
			issues = append(issues, fmt.Sprintf("%s: NotReady", node.Name))
			continue
		}
		if node.Cordoned {
			if status != monitors.RAGRed {
				status = monitors.RAGAmber
			}
			issues = append(issues, fmt.Sprintf("%s: Cordoned", node.Name))
		}
		if node.MemoryPressure {
			if status != monitors.RAGRed {
				status = monitors.RAGAmber
			}
			issues = append(issues, fmt.Sprintf("%s: MemoryPressure", node.Name))
		}
		if node.DiskPressure {
			if status != monitors.RAGRed {
				status = monitors.RAGAmber
			}
			issues = append(issues, fmt.Sprintf("%s: DiskPressure", node.Name))
		}
		if node.PIDPressure {
			if status != monitors.RAGRed {
				status = monitors.RAGAmber
			}
			issues = append(issues, fmt.Sprintf("%s: PIDPressure", node.Name))
		}
	}

	value := fmt.Sprintf("%d nodes", len(nodes))
	value2 := ""
	if len(issues) > 0 {
		value2 = strings.Join(issues, ", ")
	}

	return monitors.Result{
		Status:      status,
		Value:       value,
		Value2:      value2,
		Group:       Group,
		DisplayName: m.DisplayName(),
	}
}

// Name returns the unique name/ID of the monitor.
func (m Monitor) Name() string {
	return fmt.Sprintf("%s_%s", Group, m.name)
}

// DisplayName returns the human-readable name for display.
func (m Monitor) DisplayName() string {
	return fmt.Sprintf("K8s Nodes: %s", m.name)
}

// Group returns the group/category of the monitor.
func (m Monitor) Group() string {
	return Group
}

// AlertThreshold returns the number of consecutive failures before triggering an alert.
func (m Monitor) AlertThreshold() int {
	return m.alertThreshold
}

// Save persists the nodes monitor configuration to the provided config struct.
func (m Monitor) Save(cfg *config.Config) {
	if cfg.Monitoring.K8sNodes == nil {
		cfg.Monitoring.K8sNodes = make(map[string]config.K8sNodesConfig)
	}
	cfg.Monitoring.K8sNodes[m.name] = config.K8sNodesConfig{
		Icon:           m.icon,
		AlertThreshold: m.alertThreshold,
	}
}
