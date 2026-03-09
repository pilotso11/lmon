// Package k8sservice provides a Kubernetes Service monitor implementation.
// It checks pod health behind a Kubernetes service by probing pod endpoints
// and reports RAG status based on the percentage of healthy pods.
//
// # K8s Service Monitor
//
// ## How it works:
//   - Uses a Provider interface to abstract Kubernetes API and HTTP calls.
//   - Configured with:
//   - name: Logical name for the service monitor.
//   - namespace: Kubernetes namespace of the service.
//   - service: Name of the Kubernetes service.
//   - port: Port to probe on each pod.
//   - healthPath: HTTP health check path (e.g., "/healthz").
//   - threshold: Percentage of healthy pods for green status.
//   - timeout: HTTP request timeout in milliseconds.
//   - icon: UI icon (optional, defaults to "globe").
//   - On each check:
//   - Fetches pod endpoints for the service.
//   - Status is:
//   - Green: Healthy percentage >= threshold.
//   - Amber: Healthy percentage > 50% but below threshold.
//   - Red: Healthy percentage <= 50% or zero pods.
//   - Error: Provider returned an error.
package k8sservice

import (
	"context"
	"fmt"

	"lmon/config"
	"lmon/monitors"
)

// Default values
const Icon = "globe"        // Default icon for k8s service monitors
const Group = "k8sservice" // Group name for k8s service monitors

// PodEndpoint represents a pod behind a Kubernetes service.
type PodEndpoint struct {
	Name    string
	IP      string
	Healthy bool
}

// Provider is an interface for fetching Kubernetes service pod health.
type Provider interface {
	GetPodHealth(ctx context.Context, namespace, service string, port int, healthPath string, timeout int) ([]PodEndpoint, error)
}

// NoopProvider is a no-op provider when no real k8s client is available.
type NoopProvider struct{}

// GetPodHealth returns no endpoints and no error.
func (n *NoopProvider) GetPodHealth(_ context.Context, _, _ string, _ int, _ string, _ int) ([]PodEndpoint, error) {
	return nil, nil
}

// Monitor represents a Kubernetes Service monitor.
type Monitor struct {
	name           string
	namespace      string
	service        string
	port           int
	healthPath     string
	threshold      int
	timeout        int
	icon           string
	alertThreshold int
	impl           Provider
}

// NewMonitor constructs a new k8s service Monitor.
func NewMonitor(name, namespace, service string, port int, healthPath string, threshold, timeout int, icon string, alertThreshold int, impl Provider) Monitor {
	if icon == "" {
		icon = Icon
	}
	if threshold <= 0 {
		threshold = 80
	}
	if timeout <= 0 {
		timeout = 5000
	}
	if alertThreshold <= 0 {
		alertThreshold = 1
	}
	if impl == nil {
		impl = &NoopProvider{}
	}
	return Monitor{
		name:           name,
		namespace:      namespace,
		service:        service,
		port:           port,
		healthPath:     healthPath,
		threshold:      threshold,
		timeout:        timeout,
		icon:           icon,
		alertThreshold: alertThreshold,
		impl:           impl,
	}
}

// Check performs the service health check and returns the monitor result.
func (m Monitor) Check(ctx context.Context) monitors.Result {
	pods, err := m.impl.GetPodHealth(ctx, m.namespace, m.service, m.port, m.healthPath, m.timeout)
	if err != nil {
		return monitors.Result{
			Status:      monitors.RAGError,
			Value:       fmt.Sprintf("error: %v", err),
			Group:       Group,
			DisplayName: m.DisplayName(),
		}
	}

	total := len(pods)
	if total == 0 {
		return monitors.Result{
			Status:      monitors.RAGRed,
			Value:       "0 pods",
			Group:       Group,
			DisplayName: m.DisplayName(),
		}
	}

	healthy := 0
	for _, p := range pods {
		if p.Healthy {
			healthy++
		}
	}

	pct := (healthy * 100) / total
	status := monitors.RAGGreen
	if pct < m.threshold {
		if pct > 50 {
			status = monitors.RAGAmber
		} else {
			status = monitors.RAGRed
		}
	}

	return monitors.Result{
		Status:      status,
		Value:       fmt.Sprintf("%d/%d healthy (%d%%)", healthy, total, pct),
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
	return fmt.Sprintf("K8s Service: %s/%s", m.namespace, m.service)
}

// Group returns the group/category of the monitor.
func (m Monitor) Group() string {
	return Group
}

// AlertThreshold returns the number of consecutive failures before triggering an alert.
func (m Monitor) AlertThreshold() int {
	return m.alertThreshold
}

// Save persists the service monitor configuration to the provided config struct.
func (m Monitor) Save(cfg *config.Config) {
	if cfg.Monitoring.K8sService == nil {
		cfg.Monitoring.K8sService = make(map[string]config.K8sServiceConfig)
	}
	cfg.Monitoring.K8sService[m.name] = config.K8sServiceConfig{
		Namespace:      m.namespace,
		Service:        m.service,
		HealthPath:     m.healthPath,
		Port:           m.port,
		Threshold:      m.threshold,
		Timeout:        m.timeout,
		Icon:           m.icon,
		AlertThreshold: m.alertThreshold,
	}
}
