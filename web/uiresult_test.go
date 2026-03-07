package web

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"lmon/config"
	"lmon/monitors"
	"lmon/monitors/disk"
	"lmon/monitors/docker"
	"lmon/monitors/healthcheck"
	"lmon/monitors/k8sevents"
	"lmon/monitors/k8snodes"
	"lmon/monitors/k8sservice"
	"lmon/monitors/ping"
	"lmon/monitors/system"
)

func TestNewUIResult_DiskGroup(t *testing.T) {
	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			Disk: map[string]config.DiskConfig{
				"root": {Path: "/", Threshold: 80, Icon: "hdd", AlertThreshold: 3},
			},
		},
	}
	result := monitors.Result{Group: disk.Group, Status: monitors.RAGGreen, Value: "42%", DisplayName: "Root Disk"}
	ui := newUIResult("disk_root", result, cfg, 0)
	assert.Equal(t, "hdd", ui.Icon)
	assert.Equal(t, 80, ui.Threshold)
	assert.Equal(t, 3, ui.AlertThreshold)
	assert.Equal(t, "Disk", ui.TypeLabel)
	assert.Equal(t, "status-ok", ui.StatusClass)
}

func TestNewUIResult_HealthcheckGroup(t *testing.T) {
	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			Healthcheck: map[string]config.HealthcheckConfig{
				"api": {URL: "http://localhost", Icon: "heart", AlertThreshold: 2, RestartContainers: "web,worker"},
			},
		},
	}
	result := monitors.Result{Group: healthcheck.Group, Status: monitors.RAGAmber, Value: "slow"}
	ui := newUIResult("health_api", result, cfg, 1)
	assert.Equal(t, "heart", ui.Icon)
	assert.Equal(t, 2, ui.AlertThreshold)
	assert.Equal(t, "Health", ui.TypeLabel)
	assert.True(t, ui.HasRestartContainers)
	assert.Equal(t, "status-warning", ui.StatusClass)
}

func TestNewUIResult_PingGroup(t *testing.T) {
	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			Ping: map[string]config.PingConfig{
				"dns": {Address: "8.8.8.8", Icon: "wifi", AmberThreshold: 100, AlertThreshold: 5},
			},
		},
	}
	result := monitors.Result{Group: ping.Group, Status: monitors.RAGRed, Value: "timeout"}
	ui := newUIResult("ping_dns", result, cfg, 3)
	assert.Equal(t, "wifi", ui.Icon)
	assert.Equal(t, 100, ui.Threshold)
	assert.Equal(t, 5, ui.AlertThreshold)
	assert.Equal(t, "Ping", ui.TypeLabel)
	assert.Equal(t, "status-error", ui.StatusClass)
}

func TestNewUIResult_SystemGroup(t *testing.T) {
	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			System: config.SystemConfig{
				CPU:    config.SystemItem{Threshold: 90, Icon: "cpu", AlertThreshold: 2},
				Memory: config.SystemItem{Threshold: 85, Icon: "memory", AlertThreshold: 3},
			},
		},
	}

	cpuResult := monitors.Result{Group: system.Group, Status: monitors.RAGGreen, Value: "50%", DisplayName: system.CPUDisplayName}
	ui := newUIResult("system_cpu", cpuResult, cfg, 0)
	assert.Equal(t, "cpu", ui.Icon)
	assert.Equal(t, 90, ui.Threshold)
	assert.Equal(t, 2, ui.AlertThreshold)
	assert.Equal(t, "System", ui.TypeLabel)

	memResult := monitors.Result{Group: system.Group, Status: monitors.RAGAmber, Value: "87%", DisplayName: system.MemDisplayName}
	ui = newUIResult("system_mem", memResult, cfg, 1)
	assert.Equal(t, "memory", ui.Icon)
	assert.Equal(t, 85, ui.Threshold)
	assert.Equal(t, 3, ui.AlertThreshold)
}

func TestNewUIResult_DockerGroup(t *testing.T) {
	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			Docker: map[string]config.DockerConfig{
				"web": {Containers: "nginx", Threshold: 1, Icon: "box", AlertThreshold: 2},
			},
		},
	}
	result := monitors.Result{Group: docker.Group, Status: monitors.RAGGreen, Value: "1/1 running"}
	ui := newUIResult("docker_web", result, cfg, 0)
	assert.Equal(t, "box", ui.Icon)
	assert.Equal(t, 1, ui.Threshold)
	assert.Equal(t, 2, ui.AlertThreshold)
	assert.Equal(t, "Docker", ui.TypeLabel)
}

func TestNewUIResult_K8sEventsGroup(t *testing.T) {
	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			K8sEvents: map[string]config.K8sEventsConfig{
				"pod-events": {Namespaces: "default", Threshold: 5, Icon: "zap", AlertThreshold: 3},
			},
		},
	}
	result := monitors.Result{Group: k8sevents.Group, Status: monitors.RAGAmber, Value: "3 events"}
	ui := newUIResult("k8sevents_pod-events", result, cfg, 1)
	assert.Equal(t, "zap", ui.Icon)
	assert.Equal(t, 5, ui.Threshold)
	assert.Equal(t, 3, ui.AlertThreshold)
	assert.Equal(t, "K8s Events", ui.TypeLabel)
	assert.Equal(t, "status-warning", ui.StatusClass)
}

func TestNewUIResult_K8sNodesGroup(t *testing.T) {
	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			K8sNodes: map[string]config.K8sNodesConfig{
				"cluster": {Icon: "server", AlertThreshold: 2},
			},
		},
	}
	result := monitors.Result{Group: k8snodes.Group, Status: monitors.RAGGreen, Value: "3/3 Ready"}
	ui := newUIResult("k8snodes_cluster", result, cfg, 0)
	assert.Equal(t, "server", ui.Icon)
	assert.Equal(t, 2, ui.AlertThreshold)
	assert.Equal(t, "K8s Nodes", ui.TypeLabel)
	assert.Equal(t, "status-ok", ui.StatusClass)
}

func TestNewUIResult_K8sServiceGroup(t *testing.T) {
	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			K8sService: map[string]config.K8sServiceConfig{
				"api": {Service: "api-svc", Namespace: "default", Threshold: 80, Icon: "globe", AlertThreshold: 4},
			},
		},
	}
	result := monitors.Result{Group: k8sservice.Group, Status: monitors.RAGRed, Value: "1/4 healthy"}
	ui := newUIResult("k8sservice_api", result, cfg, 2)
	assert.Equal(t, "globe", ui.Icon)
	assert.Equal(t, 80, ui.Threshold)
	assert.Equal(t, 4, ui.AlertThreshold)
	assert.Equal(t, "K8s Service", ui.TypeLabel)
	assert.Equal(t, "status-error", ui.StatusClass)
}

func TestNewUIResult_ErrorStatus(t *testing.T) {
	cfg := &config.Config{}
	result := monitors.Result{Group: disk.Group, Status: monitors.RAGError, Value: "error"}
	ui := newUIResult("disk_test", result, cfg, 0)
	assert.Equal(t, "status-critical", ui.StatusClass)
}

func TestNewUIResult_UnknownGroup(t *testing.T) {
	cfg := &config.Config{}
	result := monitors.Result{Group: "unknown", Status: monitors.RAGGreen, Value: "ok"}
	ui := newUIResult("unknown_test", result, cfg, 0)
	assert.Equal(t, "folder", ui.Icon) // default icon
	assert.Equal(t, 1, ui.AlertThreshold)
	assert.Equal(t, "", ui.TypeLabel)
}

func TestNewUIResult_DefaultAlertThreshold(t *testing.T) {
	cfg := &config.Config{
		Monitoring: config.MonitoringConfig{
			Disk: map[string]config.DiskConfig{
				"test": {Path: "/", Threshold: 80, AlertThreshold: 0}, // zero should default to 1
			},
		},
	}
	result := monitors.Result{Group: disk.Group, Status: monitors.RAGGreen, Value: "50%"}
	ui := newUIResult("disk_test", result, cfg, 0)
	assert.Equal(t, 1, ui.AlertThreshold) // should be defaulted to 1
}
