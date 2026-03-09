// Package mapper provides the Mapper type for constructing monitor implementations
// from configuration, supporting both production and mock/test providers.
package mapper

import (
	"context"

	"lmon/common"
	"lmon/config"
	"lmon/monitors/disk"
	"lmon/monitors/docker"
	"lmon/monitors/healthcheck"
	"lmon/monitors/k8sevents"
	"lmon/monitors/k8snodes"
	"lmon/monitors/k8sservice"
	"lmon/monitors/ping"
	"lmon/monitors/system"
)

// WebhookCallbackFunc is a function type for handling webhook notifications.
type WebhookCallbackFunc func(msg string)

// Implementations holds optional mock/test providers for each monitor type.
// If a field is nil, the default production implementation is used.
type Implementations struct {
	Disk       *disk.MockDiskProvider               // Optional mock disk provider
	Health     *healthcheck.MockHealthcheckProvider // Optional mock healthcheck provider
	Cpu        *system.MockCpuProvider              // Optional mock CPU provider
	Mem        *system.MockMemProvider              // Optional mock memory provider
	Ping       *ping.MockPingProvider               // Optional mock ping provider for tests; nil uses DefaultPingProvider
	Docker     *docker.MockDockerProvider           // Optional mock Docker provider
	K8sEvents  k8sevents.Provider                   // Optional K8s events provider (mock or real)
	K8sNodes   k8snodes.Provider                    // Optional K8s nodes provider (mock or real)
	K8sService k8sservice.Provider                  // Optional K8s service provider (mock or real)
	Webhook    WebhookCallbackFunc                  // Optional webhook callback for testing
}

// Mapper constructs monitor implementations from configuration and optional providers.
// It is used to abstract over production and test/mock implementations.
type Mapper struct {
	Impls                    Implementations
	AllowedRestartContainers []string // Global whitelist of containers allowed for restart
}

// NewMapper returns a Mapper with the given implementations.
// If impls is nil, all providers are set to nil (production defaults).
func NewMapper(impls *Implementations) Mapper {
	if impls == nil {
		impls = &Implementations{}
	}
	return Mapper{
		Impls:                    *impls,
		AllowedRestartContainers: nil, // Empty by default, will be set from config
	}
}

// NewDisk constructs a disk monitor using the provided configuration and optional mock provider.
func (d Mapper) NewDisk(_ context.Context, name string, cfg config.DiskConfig) (disk.Disk, error) {
	name, _ = config.SanitiseName(name)
	return disk.NewDisk(name, cfg.Path, cfg.Threshold, cfg.Icon, cfg.AlertThreshold, d.Impls.Disk), nil
}

// NewHealthcheck constructs a healthcheck monitor using the provided configuration and optional mock provider.
func (d Mapper) NewHealthcheck(_ context.Context, name string, cfg config.HealthcheckConfig) (healthcheck.Healthcheck, error) {
	name, _ = config.SanitiseName(name)
	return healthcheck.NewHealthcheck(name, cfg.URL, cfg.Timeout, cfg.RespCode, cfg.Icon, cfg.RestartContainers, cfg.AlertThreshold, d.AllowedRestartContainers, d.Impls.Health, d.Impls.Docker)
}

// NewCpu constructs a CPU monitor using the provided configuration and optional mock provider.
func (d Mapper) NewCpu(_ context.Context, cfg config.SystemItem) (system.Cpu, error) {
	return system.NewCpu(cfg.Threshold, cfg.Icon, cfg.AlertThreshold, d.Impls.Cpu), nil
}

// NewPing constructs a ping monitor using the provided configuration and optional mock provider.
// AmberThreshold is required and must be > 0.
func (d Mapper) NewPing(_ context.Context, name string, cfg config.PingConfig) (ping.Monitor, error) {
	name, _ = config.SanitiseName(name)
	var provider ping.Provider
	if d.Impls.Ping != nil {
		provider = d.Impls.Ping
	} else {
		provider = ping.NewDefaultPingProvider()
	}
	return ping.NewPingMonitor(name, cfg.Address, cfg.Timeout, cfg.Icon, cfg.AmberThreshold, cfg.AlertThreshold, provider), nil
}

// NewMem constructs a memory monitor using the provided configuration and optional mock provider.
func (d Mapper) NewMem(_ context.Context, cfg config.SystemItem) (system.Mem, error) {
	return system.NewMem(cfg.Threshold, cfg.Icon, cfg.AlertThreshold, d.Impls.Mem), nil
}

// NewDocker constructs a Docker monitor using the provided configuration and optional mock provider.
func (d Mapper) NewDocker(_ context.Context, name string, cfg config.DockerConfig) (docker.Monitor, error) {
	name, _ = config.SanitiseName(name)
	var impl docker.Provider
	if !common.IsNil(d.Impls.Docker) {
		impl = d.Impls.Docker
	}
	return docker.NewMonitor(name, cfg.Containers, cfg.Threshold, cfg.Icon, cfg.AlertThreshold, d.AllowedRestartContainers, impl)
}

// NewK8sEvents constructs a K8s events monitor using the provided configuration and optional provider.
func (d Mapper) NewK8sEvents(_ context.Context, name string, cfg config.K8sEventsConfig) (k8sevents.Monitor, error) {
	name, _ = config.SanitiseName(name)
	return k8sevents.NewMonitor(name, cfg.Namespaces, cfg.Threshold, cfg.Window, cfg.Icon, cfg.AlertThreshold, d.Impls.K8sEvents), nil
}

// NewK8sNodes constructs a K8s nodes monitor using the provided configuration and optional provider.
func (d Mapper) NewK8sNodes(_ context.Context, name string, cfg config.K8sNodesConfig) (k8snodes.Monitor, error) {
	name, _ = config.SanitiseName(name)
	return k8snodes.NewMonitor(name, cfg.Icon, cfg.AlertThreshold, d.Impls.K8sNodes), nil
}

// NewK8sService constructs a K8s service monitor using the provided configuration and optional provider.
func (d Mapper) NewK8sService(_ context.Context, name string, cfg config.K8sServiceConfig) (k8sservice.Monitor, error) {
	name, _ = config.SanitiseName(name)
	return k8sservice.NewMonitor(name, cfg.Namespace, cfg.Service, cfg.Port, cfg.HealthPath, cfg.Threshold, cfg.Timeout, cfg.Icon, cfg.AlertThreshold, d.Impls.K8sService), nil
}
