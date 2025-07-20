// Package ping provides the Ping monitor implementation for ICMP ping checks.
// It supports both production and mock/test usage providers.
//
// # Ping Monitor
//
// The Ping monitor checks the network reachability of an IP address or hostname using ICMP ping.
//
// ## How it works:
//   - Uses a PingProvider interface to abstract ping checks (default: system ping command).
//   - Configured with:
//     - name: Logical name for the ping monitor.
//     - address: IP address or hostname to ping.
//     - timeout: Timeout for the ping request (ms).
//     - icon: UI icon (optional).
//   - On each check:
//     - Performs an ICMP ping to the configured address.
//     - Status is:
//         - Green: Ping successful and response time < 100ms.
//         - Amber: Ping successful and response time >= 100ms.
//         - Red: Ping failed or timed out.
//   - Configuration is persisted back to the config struct for saving.

package ping

import (
	"context"
	"fmt"
	"time"

	probing "github.com/prometheus-community/pro-bing"

	"lmon/config"
	"lmon/monitors"
)

// Default values
const Icon = "wifi"    // Default icon for ping monitors
const Group = "health" // Group name for ping monitors

// PingProvider is an interface for performing ping checks.
type PingProvider interface {
	Ping(ctx context.Context, address string, timeoutMs int) (responseMs int, err error)
}

// DefaultPingProvider uses pro-bing for ICMP ping.
type DefaultPingProvider struct{}

func NewDefaultPingProvider() *DefaultPingProvider {
	return &DefaultPingProvider{}
}

// Ping performs an ICMP ping using pro-bing.
func (p *DefaultPingProvider) Ping(ctx context.Context, address string, timeoutMs int) (int, error) {
	pinger, err := probing.NewPinger(address)
	if err != nil {
		return 0, fmt.Errorf("ping setup failed: %v", err)
	}
	pinger.Count = 1
	pinger.Timeout = time.Duration(timeoutMs) * time.Millisecond

	// Run ping (blocking)
	err = pinger.Run()
	if err != nil {
		return 0, fmt.Errorf("ping failed: %v", err)
	}
	stats := pinger.Statistics()
	if stats.PacketsRecv == 0 {
		return 0, fmt.Errorf("no packets received")
	}
	// Return round-trip time in ms
	return int(stats.AvgRtt.Milliseconds()), nil
}

// PingMonitor represents an ICMP ping monitor.
type PingMonitor struct {
	Name           string
	Address        string
	Timeout        int
	Icon           string
	AmberThreshold int
	Impl           PingProvider
}

// NewPingMonitor constructs a new PingMonitor.
func NewPingMonitor(name, address string, timeout int, icon string, amberThreshold int, impl PingProvider) PingMonitor {
	if icon == "" {
		icon = Icon
	}
	if impl == nil {
		impl = NewDefaultPingProvider()
	}
	return PingMonitor{
		Name:           name,
		Address:        address,
		Timeout:        timeout,
		Icon:           icon,
		AmberThreshold: amberThreshold,
		Impl:           impl,
	}
}

// Check performs the ping and returns the monitor result.
func (pm PingMonitor) Check(ctx context.Context) monitors.Result {
	responseMs, err := pm.Impl.Ping(ctx, pm.Address, pm.Timeout)
	if err != nil {
		return monitors.Result{
			Status:      monitors.RAGRed,
			Value:       fmt.Sprintf("Ping error: %v", err),
			Group:       Group,
			DisplayName: pm.DisplayName(),
		}
	}
	status := monitors.RAGGreen
	if responseMs >= pm.AmberThreshold {
		status = monitors.RAGAmber
	}
	return monitors.Result{
		Status:      status,
		Value:       fmt.Sprintf("%d ms", responseMs),
		Group:       Group,
		DisplayName: pm.DisplayName(),
	}
}

func (pm PingMonitor) DisplayName() string {
	return fmt.Sprintf("Ping: %s", pm.Name)
}

func (pm PingMonitor) Group() string {
	return Group
}

// Save persists the ping monitor configuration to the provided config struct.
func (pm PingMonitor) Save(cfg *config.Config) {
	if cfg.Monitoring.Ping == nil {
		cfg.Monitoring.Ping = make(map[string]config.PingConfig)
	}
	cfg.Monitoring.Ping[pm.Name] = config.PingConfig{
		Address:        pm.Address,
		Timeout:        pm.Timeout,
		Icon:           pm.Icon,
		AmberThreshold: pm.AmberThreshold,
	}
}
