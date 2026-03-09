package k8sservice

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"lmon/config"
	"lmon/monitors"
)

func TestGreenAllHealthy(t *testing.T) {
	pods := []PodEndpoint{
		{Name: "pod-1", IP: "10.0.0.1", Healthy: true},
		{Name: "pod-2", IP: "10.0.0.2", Healthy: true},
		{Name: "pod-3", IP: "10.0.0.3", Healthy: true},
	}
	m := NewMonitor("test", "default", "my-svc", 8080, "/healthz", 80, 5000, "", 0, NewMockProvider(pods, nil))
	result := m.Check(t.Context())
	assert.Equal(t, monitors.RAGGreen, result.Status)
	assert.Equal(t, "3/3 healthy (100%)", result.Value)
	assert.Equal(t, Group, result.Group)
}

func TestGreenAboveThreshold(t *testing.T) {
	pods := []PodEndpoint{
		{Name: "pod-1", IP: "10.0.0.1", Healthy: true},
		{Name: "pod-2", IP: "10.0.0.2", Healthy: true},
		{Name: "pod-3", IP: "10.0.0.3", Healthy: true},
		{Name: "pod-4", IP: "10.0.0.4", Healthy: true},
		{Name: "pod-5", IP: "10.0.0.5", Healthy: false},
	}
	// 4/5 = 80%, threshold is 80%, should be green
	m := NewMonitor("test", "default", "my-svc", 8080, "/healthz", 80, 5000, "", 0, NewMockProvider(pods, nil))
	result := m.Check(t.Context())
	assert.Equal(t, monitors.RAGGreen, result.Status)
	assert.Equal(t, "4/5 healthy (80%)", result.Value)
}

func TestAmberBelowThresholdAbove50(t *testing.T) {
	pods := []PodEndpoint{
		{Name: "pod-1", IP: "10.0.0.1", Healthy: true},
		{Name: "pod-2", IP: "10.0.0.2", Healthy: true},
		{Name: "pod-3", IP: "10.0.0.3", Healthy: false},
		{Name: "pod-4", IP: "10.0.0.4", Healthy: false},
	}
	// 2/4 = 50%, threshold is 80%, 50% is NOT > 50, so Red
	// Use 3/4 for amber case
	pods2 := []PodEndpoint{
		{Name: "pod-1", IP: "10.0.0.1", Healthy: true},
		{Name: "pod-2", IP: "10.0.0.2", Healthy: true},
		{Name: "pod-3", IP: "10.0.0.3", Healthy: true},
		{Name: "pod-4", IP: "10.0.0.4", Healthy: false},
		{Name: "pod-5", IP: "10.0.0.5", Healthy: false},
	}
	// 3/5 = 60%, threshold is 80%, 60% > 50 so Amber
	m := NewMonitor("test", "default", "my-svc", 8080, "/healthz", 80, 5000, "", 0, NewMockProvider(pods2, nil))
	result := m.Check(t.Context())
	assert.Equal(t, monitors.RAGAmber, result.Status)
	assert.Equal(t, "3/5 healthy (60%)", result.Value)

	// Also verify the 2/4 = 50% case is Red
	m2 := NewMonitor("test2", "default", "my-svc", 8080, "/healthz", 80, 5000, "", 0, NewMockProvider(pods, nil))
	result2 := m2.Check(t.Context())
	assert.Equal(t, monitors.RAGRed, result2.Status)
}

func TestRedAtOrBelow50Percent(t *testing.T) {
	pods := []PodEndpoint{
		{Name: "pod-1", IP: "10.0.0.1", Healthy: true},
		{Name: "pod-2", IP: "10.0.0.2", Healthy: false},
		{Name: "pod-3", IP: "10.0.0.3", Healthy: false},
		{Name: "pod-4", IP: "10.0.0.4", Healthy: false},
	}
	// 1/4 = 25%, should be red
	m := NewMonitor("test", "default", "my-svc", 8080, "/healthz", 80, 5000, "", 0, NewMockProvider(pods, nil))
	result := m.Check(t.Context())
	assert.Equal(t, monitors.RAGRed, result.Status)
	assert.Equal(t, "1/4 healthy (25%)", result.Value)
}

func TestRedZeroPods(t *testing.T) {
	m := NewMonitor("test", "default", "my-svc", 8080, "/healthz", 80, 5000, "", 0, NewMockProvider([]PodEndpoint{}, nil))
	result := m.Check(t.Context())
	assert.Equal(t, monitors.RAGRed, result.Status)
	assert.Equal(t, "0 pods", result.Value)
}

func TestRedNilPods(t *testing.T) {
	m := NewMonitor("test", "default", "my-svc", 8080, "/healthz", 80, 5000, "", 0, NewMockProvider(nil, nil))
	result := m.Check(t.Context())
	assert.Equal(t, monitors.RAGRed, result.Status)
	assert.Equal(t, "0 pods", result.Value)
}

func TestRedAllUnhealthy(t *testing.T) {
	pods := []PodEndpoint{
		{Name: "pod-1", IP: "10.0.0.1", Healthy: false},
		{Name: "pod-2", IP: "10.0.0.2", Healthy: false},
	}
	m := NewMonitor("test", "default", "my-svc", 8080, "/healthz", 80, 5000, "", 0, NewMockProvider(pods, nil))
	result := m.Check(t.Context())
	assert.Equal(t, monitors.RAGRed, result.Status)
	assert.Equal(t, "0/2 healthy (0%)", result.Value)
}

func TestErrorFromProvider(t *testing.T) {
	m := NewMonitor("test", "default", "my-svc", 8080, "/healthz", 80, 5000, "", 0, NewMockProvider(nil, errors.New("forbidden")))
	result := m.Check(t.Context())
	assert.Equal(t, monitors.RAGError, result.Status)
	assert.Contains(t, result.Value, "error")
	assert.Contains(t, result.Value, "forbidden")
}

func TestDefaultValues(t *testing.T) {
	m := NewMonitor("defaults", "ns", "svc", 80, "/health", 0, 0, "", 0, nil)
	assert.Equal(t, "k8sservice_defaults", m.Name())
	assert.Equal(t, "K8s Service: ns/svc", m.DisplayName())
	assert.Equal(t, Group, m.Group())
	assert.Equal(t, 1, m.AlertThreshold())
	assert.Equal(t, Icon, m.icon)
	assert.Equal(t, 80, m.threshold)
	assert.Equal(t, 5000, m.timeout)
	assert.NotNil(t, m.impl)
}

func TestCustomIcon(t *testing.T) {
	m := NewMonitor("test", "ns", "svc", 80, "/health", 80, 5000, "custom-icon", 0, NewMockProvider(nil, nil))
	assert.Equal(t, "custom-icon", m.icon)
}

func TestCustomAlertThreshold(t *testing.T) {
	m := NewMonitor("test", "ns", "svc", 80, "/health", 80, 5000, "", 4, NewMockProvider(nil, nil))
	assert.Equal(t, 4, m.AlertThreshold())
}

func TestSave(t *testing.T) {
	cfg := &config.Config{}
	m := NewMonitor("save-test", "prod", "api-gateway", 8443, "/healthz", 90, 3000, "globe", 2, NewMockProvider(nil, nil))
	m.Save(cfg)

	saved, ok := cfg.Monitoring.K8sService["save-test"]
	require.True(t, ok, "config should be saved")
	assert.Equal(t, "prod", saved.Namespace)
	assert.Equal(t, "api-gateway", saved.Service)
	assert.Equal(t, "/healthz", saved.HealthPath)
	assert.Equal(t, 8443, saved.Port)
	assert.Equal(t, 90, saved.Threshold)
	assert.Equal(t, 3000, saved.Timeout)
	assert.Equal(t, "globe", saved.Icon)
	assert.Equal(t, 2, saved.AlertThreshold)
}

func TestSaveWithExistingConfig(t *testing.T) {
	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			K8sService: map[string]config.K8sServiceConfig{
				"existing": {
					Namespace: "staging",
					Service:   "old-svc",
				},
			},
		},
	}
	m := NewMonitor("new", "prod", "new-svc", 80, "/health", 80, 5000, "globe", 1, NewMockProvider(nil, nil))
	m.Save(cfg)

	// Existing config preserved
	existing, ok := cfg.Monitoring.K8sService["existing"]
	require.True(t, ok)
	assert.Equal(t, "staging", existing.Namespace)

	// New config added
	newCfg, ok := cfg.Monitoring.K8sService["new"]
	require.True(t, ok)
	assert.Equal(t, "prod", newCfg.Namespace)
	assert.Equal(t, "new-svc", newCfg.Service)
}

func TestNoopProvider(t *testing.T) {
	p := &NoopProvider{}
	pods, err := p.GetPodHealth(t.Context(), "ns", "svc", 80, "/health", 5000)
	assert.NoError(t, err)
	assert.Nil(t, pods)
}

func TestDisplayName(t *testing.T) {
	m := NewMonitor("test", "my-namespace", "my-service", 80, "/health", 80, 5000, "", 0, nil)
	assert.Equal(t, "K8s Service: my-namespace/my-service", m.DisplayName())
}
