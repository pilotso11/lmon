package k8snodes

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"lmon/config"
	"lmon/monitors"
)

func TestGreenAllReady(t *testing.T) {
	nodes := []NodeStatus{
		{Name: "node-1", Ready: true},
		{Name: "node-2", Ready: true},
		{Name: "node-3", Ready: true},
	}
	m := NewMonitor("cluster", "", 0, NewMockProvider(nodes, nil))
	result := m.Check(t.Context())
	assert.Equal(t, monitors.RAGGreen, result.Status)
	assert.Equal(t, "3 nodes", result.Value)
	assert.Empty(t, result.Value2)
}

func TestGreenNoNodes(t *testing.T) {
	m := NewMonitor("empty", "", 0, NewMockProvider(nil, nil))
	result := m.Check(t.Context())
	assert.Equal(t, monitors.RAGGreen, result.Status)
	assert.Equal(t, "0 nodes", result.Value)
}

func TestRedNotReady(t *testing.T) {
	nodes := []NodeStatus{
		{Name: "node-1", Ready: true},
		{Name: "node-2", Ready: false},
	}
	m := NewMonitor("cluster", "", 0, NewMockProvider(nodes, nil))
	result := m.Check(t.Context())
	assert.Equal(t, monitors.RAGRed, result.Status)
	assert.Equal(t, "2 nodes", result.Value)
	assert.Contains(t, result.Value2, "node-2: NotReady")
}

func TestAmberCordoned(t *testing.T) {
	nodes := []NodeStatus{
		{Name: "node-1", Ready: true},
		{Name: "node-2", Ready: true, Cordoned: true},
	}
	m := NewMonitor("cluster", "", 0, NewMockProvider(nodes, nil))
	result := m.Check(t.Context())
	assert.Equal(t, monitors.RAGAmber, result.Status)
	assert.Contains(t, result.Value2, "node-2: Cordoned")
}

func TestAmberMemoryPressure(t *testing.T) {
	nodes := []NodeStatus{
		{Name: "node-1", Ready: true, MemoryPressure: true},
	}
	m := NewMonitor("cluster", "", 0, NewMockProvider(nodes, nil))
	result := m.Check(t.Context())
	assert.Equal(t, monitors.RAGAmber, result.Status)
	assert.Contains(t, result.Value2, "node-1: MemoryPressure")
}

func TestAmberDiskPressure(t *testing.T) {
	nodes := []NodeStatus{
		{Name: "node-1", Ready: true, DiskPressure: true},
	}
	m := NewMonitor("cluster", "", 0, NewMockProvider(nodes, nil))
	result := m.Check(t.Context())
	assert.Equal(t, monitors.RAGAmber, result.Status)
	assert.Contains(t, result.Value2, "node-1: DiskPressure")
}

func TestAmberPIDPressure(t *testing.T) {
	nodes := []NodeStatus{
		{Name: "node-1", Ready: true, PIDPressure: true},
	}
	m := NewMonitor("cluster", "", 0, NewMockProvider(nodes, nil))
	result := m.Check(t.Context())
	assert.Equal(t, monitors.RAGAmber, result.Status)
	assert.Contains(t, result.Value2, "node-1: PIDPressure")
}

func TestRedOverridesAmber(t *testing.T) {
	nodes := []NodeStatus{
		{Name: "node-1", Ready: true, MemoryPressure: true},
		{Name: "node-2", Ready: false},
	}
	m := NewMonitor("cluster", "", 0, NewMockProvider(nodes, nil))
	result := m.Check(t.Context())
	assert.Equal(t, monitors.RAGRed, result.Status)
	assert.Contains(t, result.Value2, "node-2: NotReady")
	assert.Contains(t, result.Value2, "node-1: MemoryPressure")
}

func TestMultiplePressureConditions(t *testing.T) {
	nodes := []NodeStatus{
		{Name: "node-1", Ready: true, MemoryPressure: true, DiskPressure: true, PIDPressure: true},
	}
	m := NewMonitor("cluster", "", 0, NewMockProvider(nodes, nil))
	result := m.Check(t.Context())
	assert.Equal(t, monitors.RAGAmber, result.Status)
	assert.Contains(t, result.Value2, "MemoryPressure")
	assert.Contains(t, result.Value2, "DiskPressure")
	assert.Contains(t, result.Value2, "PIDPressure")
}

func TestErrorFromProvider(t *testing.T) {
	m := NewMonitor("cluster", "", 0, NewMockProvider(nil, errors.New("unauthorized")))
	result := m.Check(t.Context())
	assert.Equal(t, monitors.RAGError, result.Status)
	assert.Contains(t, result.Value, "error")
	assert.Contains(t, result.Value, "unauthorized")
}

func TestDefaultValues(t *testing.T) {
	m := NewMonitor("defaults", "", 0, nil)
	assert.Equal(t, "k8snodes_defaults", m.Name())
	assert.Equal(t, "K8s Nodes: defaults", m.DisplayName())
	assert.Equal(t, Group, m.Group())
	assert.Equal(t, 1, m.AlertThreshold())
	assert.Equal(t, Icon, m.icon)
	assert.NotNil(t, m.impl)
}

func TestCustomIcon(t *testing.T) {
	m := NewMonitor("test", "custom-icon", 0, NewMockProvider(nil, nil))
	assert.Equal(t, "custom-icon", m.icon)
}

func TestCustomAlertThreshold(t *testing.T) {
	m := NewMonitor("test", "", 5, NewMockProvider(nil, nil))
	assert.Equal(t, 5, m.AlertThreshold())
}

func TestSave(t *testing.T) {
	cfg := &config.Config{}
	m := NewMonitor("save-test", "hdd-rack", 3, NewMockProvider(nil, nil))
	m.Save(cfg)

	saved, ok := cfg.Monitoring.K8sNodes["save-test"]
	require.True(t, ok, "config should be saved")
	assert.Equal(t, "hdd-rack", saved.Icon)
	assert.Equal(t, 3, saved.AlertThreshold)
}

func TestSaveWithExistingConfig(t *testing.T) {
	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			K8sNodes: map[string]config.K8sNodesConfig{
				"existing": {
					Icon:           "old-icon",
					AlertThreshold: 2,
				},
			},
		},
	}
	m := NewMonitor("new", "new-icon", 1, NewMockProvider(nil, nil))
	m.Save(cfg)

	// Existing config preserved
	existing, ok := cfg.Monitoring.K8sNodes["existing"]
	require.True(t, ok)
	assert.Equal(t, "old-icon", existing.Icon)

	// New config added
	newCfg, ok := cfg.Monitoring.K8sNodes["new"]
	require.True(t, ok)
	assert.Equal(t, "new-icon", newCfg.Icon)
}

func TestNoopProvider(t *testing.T) {
	p := &NoopProvider{}
	nodes, err := p.GetNodeStatuses(t.Context())
	assert.NoError(t, err)
	assert.Nil(t, nodes)
}
