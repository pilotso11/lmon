package mapper

import (
	"context"

	"lmon/config"
	"lmon/monitors/disk"
	"lmon/monitors/healthcheck"
	"lmon/monitors/system"
)

type Implementations struct {
	Disk   *disk.MockDiskProvider
	Health *healthcheck.MockHealthcheckProvider
	Cpu    *system.MockCpuProvider
	Mem    *system.MockMemProvider
}

// Mapper handles applying config changes..
type Mapper struct {
	Impls Implementations
}

// NewBuilder returns an Mapper struct with all providers set to nil unless mocks are supplied.
// This is used when no testing or custom overrides are required.
func NewBuilder(impls *Implementations) Mapper {
	if impls == nil {
		impls = &Implementations{}
	}
	return Mapper{*impls}
}

func (d Mapper) NewDisk(ctx context.Context, name string, cfg config.DiskConfig) (disk.Disk, error) {
	return disk.NewDisk(name, cfg.Path, cfg.Threshold, cfg.Icon, d.Impls.Disk), nil
}

func (d Mapper) NewHealthcheck(ctx context.Context, name string, cfg config.HealthcheckConfig) (healthcheck.Healthcheck, error) {
	return healthcheck.NewHealthcheck(name, cfg.URL, cfg.Timeout, cfg.Icon, d.Impls.Health)
}

func (d Mapper) NewCpu(ctx context.Context, cfg config.SystemItem) (system.Cpu, error) {
	return system.NewCpu(cfg.Threshold, cfg.Icon, d.Impls.Cpu), nil
}

func (d Mapper) NewMem(ctx context.Context, cfg config.SystemItem) (system.Mem, error) {
	return system.NewMem(cfg.Threshold, cfg.Icon, d.Impls.Mem), nil
}
