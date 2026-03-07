package k8sevents

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"lmon/config"
	"lmon/monitors"
)

func TestGreenNoEvents(t *testing.T) {
	m := NewMonitor("test", "default", 5, 300, "", 0, NewMockProvider(nil, nil))
	result := m.Check(t.Context())
	assert.Equal(t, monitors.RAGGreen, result.Status)
	assert.Equal(t, "0 events", result.Value)
	assert.Equal(t, Group, result.Group)
	assert.Equal(t, "K8s Events: test", result.DisplayName)
}

func TestAmberBelowThreshold(t *testing.T) {
	events := []PodEvent{
		{Namespace: "default", Pod: "pod-1", Reason: "CrashLoopBackOff", Message: "Back-off restarting"},
		{Namespace: "default", Pod: "pod-2", Reason: "OOMKilled", Message: "Out of memory"},
	}
	m := NewMonitor("test", "default", 5, 300, "", 0, NewMockProvider(events, nil))
	result := m.Check(t.Context())
	assert.Equal(t, monitors.RAGAmber, result.Status)
	assert.Equal(t, "2 events", result.Value)
}

func TestRedAtThreshold(t *testing.T) {
	events := []PodEvent{
		{Namespace: "default", Pod: "pod-1", Reason: "CrashLoopBackOff", Message: "msg"},
		{Namespace: "default", Pod: "pod-2", Reason: "OOMKilled", Message: "msg"},
		{Namespace: "default", Pod: "pod-3", Reason: "CrashLoopBackOff", Message: "msg"},
		{Namespace: "default", Pod: "pod-4", Reason: "Evicted", Message: "msg"},
		{Namespace: "default", Pod: "pod-5", Reason: "Failed", Message: "msg"},
	}
	m := NewMonitor("test", "default", 5, 300, "", 0, NewMockProvider(events, nil))
	result := m.Check(t.Context())
	assert.Equal(t, monitors.RAGRed, result.Status)
	assert.Equal(t, "5 events", result.Value)
}

func TestRedAboveThreshold(t *testing.T) {
	events := []PodEvent{
		{Namespace: "default", Pod: "pod-1", Reason: "CrashLoopBackOff", Message: "msg"},
		{Namespace: "default", Pod: "pod-2", Reason: "OOMKilled", Message: "msg"},
		{Namespace: "default", Pod: "pod-3", Reason: "CrashLoopBackOff", Message: "msg"},
		{Namespace: "default", Pod: "pod-4", Reason: "Evicted", Message: "msg"},
		{Namespace: "default", Pod: "pod-5", Reason: "Failed", Message: "msg"},
		{Namespace: "default", Pod: "pod-6", Reason: "Error", Message: "msg"},
	}
	m := NewMonitor("test", "default", 5, 300, "", 0, NewMockProvider(events, nil))
	result := m.Check(t.Context())
	assert.Equal(t, monitors.RAGRed, result.Status)
	assert.Equal(t, "6 events", result.Value)
}

func TestErrorFromProvider(t *testing.T) {
	m := NewMonitor("test", "default", 5, 300, "", 0, NewMockProvider(nil, errors.New("connection refused")))
	result := m.Check(t.Context())
	assert.Equal(t, monitors.RAGError, result.Status)
	assert.Contains(t, result.Value, "error")
	assert.Contains(t, result.Value, "connection refused")
}

func TestDeduplication(t *testing.T) {
	events := []PodEvent{
		{Namespace: "default", Pod: "pod-1", Reason: "CrashLoopBackOff", Message: "first"},
		{Namespace: "default", Pod: "pod-1", Reason: "CrashLoopBackOff", Message: "second"},
		{Namespace: "default", Pod: "pod-1", Reason: "CrashLoopBackOff", Message: "third"},
		{Namespace: "default", Pod: "pod-2", Reason: "OOMKilled", Message: "msg"},
	}
	// 3 events for pod-1:CrashLoopBackOff collapse to 1, plus 1 for pod-2:OOMKilled = 2 unique
	m := NewMonitor("test", "default", 5, 300, "", 0, NewMockProvider(events, nil))
	result := m.Check(t.Context())
	assert.Equal(t, monitors.RAGAmber, result.Status)
	assert.Equal(t, "2 events", result.Value)
}

func TestDefaultValues(t *testing.T) {
	m := NewMonitor("defaults", "default", 0, 0, "", 0, nil)
	assert.Equal(t, "k8sevents_defaults", m.Name())
	assert.Equal(t, "K8s Events: defaults", m.DisplayName())
	assert.Equal(t, Group, m.Group())
	assert.Equal(t, 1, m.AlertThreshold())
	assert.Equal(t, Icon, m.icon)
	assert.Equal(t, 5, m.threshold)
	assert.Equal(t, 300, m.window)
	assert.NotNil(t, m.impl)
}

func TestCustomIcon(t *testing.T) {
	m := NewMonitor("test", "default", 5, 300, "custom-icon", 0, NewMockProvider(nil, nil))
	assert.Equal(t, "custom-icon", m.icon)
}

func TestCustomAlertThreshold(t *testing.T) {
	m := NewMonitor("test", "default", 5, 300, "", 3, NewMockProvider(nil, nil))
	assert.Equal(t, 3, m.AlertThreshold())
}

func TestSave(t *testing.T) {
	cfg := &config.Config{}
	m := NewMonitor("save-test", "kube-system", 10, 600, "lightning", 2, NewMockProvider(nil, nil))
	m.Save(cfg)

	saved, ok := cfg.Monitoring.K8sEvents["save-test"]
	require.True(t, ok, "config should be saved")
	assert.Equal(t, "kube-system", saved.Namespaces)
	assert.Equal(t, 10, saved.Threshold)
	assert.Equal(t, 600, saved.Window)
	assert.Equal(t, "lightning", saved.Icon)
	assert.Equal(t, 2, saved.AlertThreshold)
}

func TestSaveWithExistingConfig(t *testing.T) {
	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			K8sEvents: map[string]config.K8sEventsConfig{
				"existing": {
					Namespaces: "prod",
					Threshold:  3,
					Window:     120,
					Icon:       "old",
				},
			},
		},
	}
	m := NewMonitor("new", "staging", 7, 400, "new-icon", 1, NewMockProvider(nil, nil))
	m.Save(cfg)

	// Existing config preserved
	existing, ok := cfg.Monitoring.K8sEvents["existing"]
	require.True(t, ok)
	assert.Equal(t, "prod", existing.Namespaces)

	// New config added
	newCfg, ok := cfg.Monitoring.K8sEvents["new"]
	require.True(t, ok)
	assert.Equal(t, "staging", newCfg.Namespaces)
	assert.Equal(t, 7, newCfg.Threshold)
}

func TestNoopProvider(t *testing.T) {
	p := &NoopProvider{}
	events, err := p.GetFailureEvents(t.Context(), "default", 300)
	assert.NoError(t, err)
	assert.Nil(t, events)
}
