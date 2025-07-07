// Package mapper provides the Mapper type for constructing monitor implementations
// from configuration, supporting both production and mock/test providers.
package mapper

import (
	"context"

	"lmon/config"
	"lmon/monitors/disk"
	"lmon/monitors/healthcheck"
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
	Webhook WebhookCallbackFunc                  // Optional webhook callback for testing
}

// Mapper constructs monitor implementations from configuration and optional providers.
// It is used to abstract over production and test/mock implementations.
type Mapper struct {
	Impls Implementations
}

// NewMapper returns a Mapper with the given implementations.
// If impls is nil, all providers are set to nil (production defaults).
func NewMapper(impls *Implementations) Mapper {
	if impls == nil {
		impls = &Implementations{}
	}
	return Mapper{*impls}
}

// NewDisk constructs a disk monitor using the provided configuration and optional mock provider.
func (d Mapper) NewDisk(ctx context.Context, name string, cfg config.DiskConfig) (disk.Disk, error) {
	return disk.NewDisk(name, cfg.Path, cfg.Threshold, cfg.Icon, d.Impls.Disk), nil
}

// NewHealthcheck constructs a healthcheck monitor using the provided configuration and optional mock provider.
func (d Mapper) NewHealthcheck(ctx context.Context, name string, cfg config.HealthcheckConfig) (healthcheck.Healthcheck, error) {
	return healthcheck.NewHealthcheck(name, cfg.URL, cfg.Timeout, cfg.Icon, d.Impls.Health)
}

// NewCpu constructs a CPU monitor using the provided configuration and optional mock provider.
func (d Mapper) NewCpu(ctx context.Context, cfg config.SystemItem) (system.Cpu, error) {
	return system.NewCpu(cfg.Threshold, cfg.Icon, d.Impls.Cpu), nil
}

// NewMem constructs a memory monitor using the provided configuration and optional mock provider.
func (d Mapper) NewMem(ctx context.Context, cfg config.SystemItem) (system.Mem, error) {
	return system.NewMem(cfg.Threshold, cfg.Icon, d.Impls.Mem), nil
}
