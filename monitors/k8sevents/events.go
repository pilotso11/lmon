// Package k8sevents provides a Kubernetes Events monitor implementation.
// It checks for failure events (e.g., CrashLoopBackOff, OOMKilled) in specified namespaces
// and reports RAG status based on the number of unique events found.
//
// # K8s Events Monitor
//
// ## How it works:
//   - Uses a Provider interface to abstract Kubernetes API calls.
//   - Configured with:
//   - name: Logical name for the events monitor.
//   - namespaces: Kubernetes namespace(s) to watch.
//   - threshold: Number of unique events to trigger Red status.
//   - window: Time window in seconds to look back for events.
//   - icon: UI icon (optional, defaults to "lightning").
//   - On each check:
//   - Fetches failure events from the configured namespace(s).
//   - Deduplicates events by pod+reason.
//   - Status is:
//   - Green: No unique events.
//   - Amber: Some events but below threshold.
//   - Red: Events at or above threshold.
//   - Error: Provider returned an error.
package k8sevents

import (
	"context"
	"fmt"

	"lmon/config"
	"lmon/monitors"
)

// Default values
const Icon = "lightning"   // Default icon for k8s events monitors
const Group = "k8sevents" // Group name for k8s events monitors

// PodEvent represents a Kubernetes pod event.
type PodEvent struct {
	Namespace string
	Pod       string
	Reason    string
	Message   string
}

// Provider is an interface for fetching Kubernetes failure events.
type Provider interface {
	GetFailureEvents(ctx context.Context, namespace string, windowSeconds int) ([]PodEvent, error)
}

// NoopProvider is a no-op provider when no real k8s client is available.
type NoopProvider struct{}

// GetFailureEvents returns no events and no error.
func (n *NoopProvider) GetFailureEvents(_ context.Context, _ string, _ int) ([]PodEvent, error) {
	return nil, nil
}

// Monitor represents a Kubernetes Events monitor.
type Monitor struct {
	name           string
	namespaces     string
	threshold      int
	window         int
	icon           string
	alertThreshold int
	impl           Provider
}

// NewMonitor constructs a new k8s events Monitor.
func NewMonitor(name, namespaces string, threshold, window int, icon string, alertThreshold int, impl Provider) Monitor {
	if icon == "" {
		icon = Icon
	}
	if threshold <= 0 {
		threshold = 5
	}
	if window <= 0 {
		window = 300
	}
	if alertThreshold <= 0 {
		alertThreshold = 1
	}
	if impl == nil {
		impl = &NoopProvider{}
	}
	return Monitor{
		name:           name,
		namespaces:     namespaces,
		threshold:      threshold,
		window:         window,
		icon:           icon,
		alertThreshold: alertThreshold,
		impl:           impl,
	}
}

// Check performs the events check and returns the monitor result.
func (m Monitor) Check(ctx context.Context) monitors.Result {
	events, err := m.impl.GetFailureEvents(ctx, m.namespaces, m.window)
	if err != nil {
		return monitors.Result{
			Status:      monitors.RAGError,
			Value:       fmt.Sprintf("error: %v", err),
			Group:       Group,
			DisplayName: m.DisplayName(),
		}
	}

	// Deduplicate by pod+reason
	seen := make(map[string]bool)
	unique := make([]PodEvent, 0)
	for _, e := range events {
		key := e.Pod + ":" + e.Reason
		if !seen[key] {
			seen[key] = true
			unique = append(unique, e)
		}
	}

	count := len(unique)
	status := monitors.RAGGreen
	if count > 0 && count < m.threshold {
		status = monitors.RAGAmber
	}
	if count >= m.threshold {
		status = monitors.RAGRed
	}

	return monitors.Result{
		Status:      status,
		Value:       fmt.Sprintf("%d events", count),
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
	return fmt.Sprintf("K8s Events: %s", m.name)
}

// Group returns the group/category of the monitor.
func (m Monitor) Group() string {
	return Group
}

// AlertThreshold returns the number of consecutive failures before triggering an alert.
func (m Monitor) AlertThreshold() int {
	return m.alertThreshold
}

// Save persists the events monitor configuration to the provided config struct.
func (m Monitor) Save(cfg *config.Config) {
	if cfg.Monitoring.K8sEvents == nil {
		cfg.Monitoring.K8sEvents = make(map[string]config.K8sEventsConfig)
	}
	cfg.Monitoring.K8sEvents[m.name] = config.K8sEventsConfig{
		Namespaces:     m.namespaces,
		Threshold:      m.threshold,
		Window:         m.window,
		Icon:           m.icon,
		AlertThreshold: m.alertThreshold,
	}
}
