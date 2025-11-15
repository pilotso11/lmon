// Package mapper provides the Mapper type for constructing monitor implementations
// from configuration, supporting both production and mock/test providers.
package mapper

import (
	"context"

	"lmon/config"
	"lmon/monitors/disk"
	"lmon/monitors/docker"
	"lmon/monitors/healthcheck"
	"lmon/monitors/ping"
	"lmon/monitors/system"
)

// WebhookCallbackFunc is a function type for handling webhook notifications.
type WebhookCallbackFunc func(msg string)

// Implementations holds optional mock/test providers for each monitor type.
// If a field is nil, the default production implementation is used.
type Implementations struct {
	Disk    *disk.MockDiskProvider               // Optional mock disk provider
	Health  *healthcheck.MockHealthcheckProvider // Optional mock healthcheck provider
	Cpu     *system.MockCpuProvider              // Optional mock CPU provider
	Mem     *system.MockMemProvider              // Optional mock memory provider
	Ping    *ping.MockPingProvider               // Optional mock ping provider for tests; nil uses DefaultPingProvider
	Docker  *docker.MockDockerProvider           // Optional mock Docker provider
	Webhook WebhookCallbackFunc                  // Optional webhook callback for testing
}

// Mapper constructs monitor implementations from configuration and optional providers.
// It is used to abstract over production and test/mock implementations.
type Mapper struct {
	Impls                    Implementations
	AllowedRestartContainers string // Global whitelist of containers allowed for restart
}

// NewMapper returns a Mapper with the given implementations.
// If impls is nil, all providers are set to nil (production defaults).
func NewMapper(impls *Implementations) Mapper {
	if impls == nil {
		impls = &Implementations{}
	}
	return Mapper{
		Impls:                    *impls,
		AllowedRestartContainers: "", // Empty by default, will be set from config
	}
}

// NewDisk constructs a disk monitor using the provided configuration and optional mock provider.
func (d Mapper) NewDisk(_ context.Context, name string, cfg config.DiskConfig) (disk.Disk, error) {
	name, _ = config.SanitiseName(name)
	return disk.NewDisk(name, cfg.Path, cfg.Threshold, cfg.Icon, d.Impls.Disk), nil
}

// NewHealthcheck constructs a healthcheck monitor using the provided configuration and optional mock provider.
func (d Mapper) NewHealthcheck(_ context.Context, name string, cfg config.HealthcheckConfig) (healthcheck.Healthcheck, error) {
	name, _ = config.SanitiseName(name)
	return healthcheck.NewHealthcheck(name, cfg.URL, cfg.Timeout, cfg.RespCode, cfg.Icon, cfg.RestartContainers, d.AllowedRestartContainers, d.Impls.Health, d.Impls.Docker)
}

// NewCpu constructs a CPU monitor using the provided configuration and optional mock provider.
func (d Mapper) NewCpu(_ context.Context, cfg config.SystemItem) (system.Cpu, error) {
	return system.NewCpu(cfg.Threshold, cfg.Icon, d.Impls.Cpu), nil
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
	return ping.NewPingMonitor(name, cfg.Address, cfg.Timeout, cfg.Icon, cfg.AmberThreshold, provider), nil
}

// NewMem constructs a memory monitor using the provided configuration and optional mock provider.
func (d Mapper) NewMem(_ context.Context, cfg config.SystemItem) (system.Mem, error) {
	return system.NewMem(cfg.Threshold, cfg.Icon, d.Impls.Mem), nil
}

// NewDocker constructs a Docker monitor using the provided configuration and optional mock provider.
func (d Mapper) NewDocker(_ context.Context, name string, cfg config.DockerConfig) (docker.Monitor, error) {
	name, _ = config.SanitiseName(name)
	return docker.NewMonitor(name, cfg.Containers, cfg.Threshold, cfg.Icon, d.AllowedRestartContainers, d.Impls.Docker)
}
