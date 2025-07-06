package system

import (
	"context"
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"

	"lmon/config"
	"lmon/monitors"
)

const CpuIcon = "cpu"
const Group = "system"

type CpuProvider interface {
	Usage() (float64, error)
}

type defaultCpuProvider struct {
	prevCPUTimes cpu.TimesStat
	lastCPUCheck time.Time
}

func newDefaultCpuProvider() *defaultCpuProvider {
	d := defaultCpuProvider{}
	_, _ = d.Usage()
	return &d
}

func (d *defaultCpuProvider) Usage() (float64, error) {
	times, err := cpu.Times(false)
	if err != nil {
		return 0, err
	}

	usage := calculateCPUPercentage(times[0], d.prevCPUTimes)

	d.prevCPUTimes = times[0]
	d.lastCPUCheck = time.Now()
	return usage, nil
}

// calculateCPUPercentage calculates the CPU usage percentage based on the difference between current and previous CPU times
func calculateCPUPercentage(current, previous cpu.TimesStat) float64 {
	// Calculate the total time spent by the CPU
	prevTotal := previous.User + previous.System + previous.Idle + previous.Nice + previous.Iowait + previous.Irq + previous.Softirq + previous.Steal
	currTotal := current.User + current.System + current.Idle + current.Nice + current.Iowait + current.Irq + current.Softirq + current.Steal

	// Calculate the total time spent by the CPU doing work
	prevBusy := previous.User + previous.System + previous.Nice + previous.Irq + previous.Softirq + previous.Steal
	currBusy := current.User + current.System + current.Nice + current.Irq + current.Softirq + current.Steal

	// Calculate the difference in total and busy time
	totalDiff := currTotal - prevTotal
	busyDiff := currBusy - prevBusy

	// Calculate the percentage of time spent by the CPU doing work
	if totalDiff > 0 {
		return (busyDiff / totalDiff) * 100.0
	}
	return 0.0
}

type Cpu struct {
	threshold int
	icon      string
	impl      CpuProvider
}

func NewCpu(threshold int, icon string, provider CpuProvider) *Cpu {
	if icon == "" {
		icon = CpuIcon
	}
	if provider == nil {
		provider = newDefaultCpuProvider()
	}
	return &Cpu{
		threshold: threshold,
		icon:      icon,
		impl:      provider,
	}
}

func (c Cpu) DisplayName() string {
	return "cpu"
}

func (c Cpu) Group() string {
	return Group
}

func (c Cpu) Name() string {
	return fmt.Sprintf("%s_cpu", Group)
}

func (c Cpu) Save(cfg *config.Config) {
	cfg.Monitoring.System.CPU.Threshold = c.threshold
	cfg.Monitoring.System.CPU.Icon = c.icon
}

func (c Cpu) Check(ctx context.Context) monitors.Result {
	usage, err := c.impl.Usage()
	if err != nil {
		return monitors.Result{
			Status: monitors.RAGError,
			Value:  fmt.Sprintf("error getting CPU usage: %v", err),
		}
	}
	val := fmt.Sprintf("%.1f%%", usage)
	status := monitors.RAGGreen
	switch {
	case usage >= float64(c.threshold):
		status = monitors.RAGRed
	case usage >= float64(c.threshold)*.9:
		status = monitors.RAGAmber
	}
	return monitors.Result{
		Status: status,
		Value:  val,
	}
}
