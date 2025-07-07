package system

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v3/mem"

	"lmon/config"
	"lmon/monitors"
)

const MemIcon = "speedometer"

type MemProvider interface {
	Usage() (*mem.VirtualMemoryStat, error)
}

type defaultMemProvider struct {
}

func (d defaultMemProvider) Usage() (*mem.VirtualMemoryStat, error) {
	return mem.VirtualMemory()
}

type Mem struct {
	threshold int
	icon      string
	impl      MemProvider
}

func NewMem(threshold int, icon string, provider MemProvider) Mem {
	if icon == "" {
		icon = MemIcon
	}
	if provider == nil {
		provider = defaultMemProvider{}
	}
	return Mem{
		threshold: threshold,
		icon:      icon,
		impl:      provider,
	}
}

func (c Mem) DisplayName() string {
	return "mem"
}

func (c Mem) Group() string {
	return Group
}

func (c Mem) Name() string {
	return fmt.Sprintf("%s_mem", Group)
}

func (c Mem) Save(cfg *config.Config) {
	cfg.Monitoring.System.Memory.Threshold = c.threshold
	cfg.Monitoring.System.Memory.Icon = c.icon
}

func (c Mem) Check(_ context.Context) monitors.Result {
	usage, err := c.impl.Usage()
	if err != nil {
		return monitors.Result{
			Status: monitors.RAGError,
			Value:  fmt.Sprintf("error getting mem Current: %v", err),
		}
	}
	val := fmt.Sprintf("%.1f%%", usage.UsedPercent)
	status := monitors.RAGGreen
	switch {
	case usage.UsedPercent >= float64(c.threshold):
		status = monitors.RAGRed
	case usage.UsedPercent >= float64(c.threshold)*.9:
		status = monitors.RAGAmber
	}
	return monitors.Result{
		Status: status,
		Value:  val,
	}
}
