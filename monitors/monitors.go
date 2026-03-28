// Package monitors provides the core monitoring service and abstractions for lmon.
// It defines monitor interfaces, result types, and the Service that manages monitor lifecycles.
package monitors

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/puzpuzpuz/xsync/v4"

	"lmon/common"
	"lmon/config"
)

// ErrNotFound is returned when a monitor is not found in the Service.
type ErrNotFound struct {
	Name string
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("monitor %s not found", e.Name)
}

// RAG represents the Red-Amber-Green status for a monitor.
type RAG int

const (
	RAGUnknown RAG = iota // Status is unknown
	RAGError              // Monitor encountered an error
	RAGRed                // Monitor is in a red (critical) state
	RAGAmber              // Monitor is in an amber (warning) state
	RAGGreen              // Monitor is in a green (healthy) state
)

// String returns the string representation of the RAG status.
func (r RAG) String() string {
	switch r {
	case RAGGreen:
		return "Green"
	case RAGAmber:
		return "Amber"
	case RAGRed:
		return "Red"
	case RAGError:
		return "Error"
	default:
		return "Unknown"
	}
}

// Result represents the outcome of a single monitor check.
type Result struct {
	Status      RAG    // The RAG status of the check
	Value       string // Human-readable value or message
	Value2      string // Optional second value for additional context
	Group       string // Group/category of the monitor
	DisplayName string // Display name for UI
}

// Monitor is the interface implemented by all monitor types.
// It defines methods for checking status, naming, grouping, and saving configuration.
type Monitor interface {
	Check(ctx context.Context) Result // Perform a check and return the result
	DisplayName() string              // Human-readable name for display
	Group() string                    // Group/category of the monitor
	Name() string                     // Unique name/ID of the monitor
	Save(cfg *config.Config)          // Save monitor configuration to the provided config
	AlertThreshold() int              // Number of consecutive failures before triggering alert (default: 1)
}

// PushFunc is a callback called when a monitor's result changes.
// It receives the monitor, previous result, and new result.
type PushFunc func(ctx context.Context, m Monitor, prev, result Result)

// Service manages the lifecycle of monitors, periodic checks, and result storage.
// It is safe for concurrent use.
type Service struct {
	period         *common.AtomicDuration
	timeout        *common.AtomicDuration
	monitors       *xsync.Map[string, Monitor]                // Map of monitor names to Monitor instances
	result         *xsync.Map[string, Result]
	failureCount   *xsync.MapOf[string, *atomic.Int64]        // Track consecutive failures per monitor (atomic to avoid races)
	maintenance    *xsync.Map[string, *config.MaintenanceConfig] // Optional maintenance windows per monitor
	cancel         context.CancelFunc
	push           PushFunc
	wg             sync.WaitGroup
}

// NewService creates a new monitoring Service.
// period: how often to check all monitors.
// timeout: maximum duration for each check.
// push: callback for result changes (may be nil).
func NewService(ctx context.Context, period time.Duration, timeout time.Duration, push PushFunc) *Service {
	s := Service{
		period:       common.NewAtomicDuration(period),
		timeout:      common.NewAtomicDuration(timeout),
		monitors:     xsync.NewMap[string, Monitor](),
		result:       xsync.NewMap[string, Result](),
		failureCount: xsync.NewMapOf[string, *atomic.Int64](),
		maintenance:  xsync.NewMap[string, *config.MaintenanceConfig](),
		push:         push,
	}
	s.startMonitors(ctx)
	return &s
}

// SetPush sets or clears the push callback function.
// If push is nil, no callback will be invoked on result changes.
// We don't lock anything here because the push function is only called once during setup in
// normal operation.
func (s *Service) SetPush(push PushFunc) {
	s.push = push
}

// Add adds a monitor to the service and performs an initial check asynchronously.
func (s *Service) Add(ctx context.Context, m Monitor) {
	s.monitors.Store(m.Name(), m)

	go func() {
		if s.isInMaintenance(m.Name()) {
			return
		}
		result := m.Check(ctx)
		s.checkStoreAndPush(ctx, m, result)
	}()
}

// AddWithMaintenance adds a monitor with an associated maintenance window configuration.
// During the maintenance window, the monitor's checks, alerts, and DB recording are skipped.
func (s *Service) AddWithMaintenance(ctx context.Context, m Monitor, mc *config.MaintenanceConfig) {
	if mc != nil {
		s.maintenance.Store(m.Name(), mc)
	} else {
		s.maintenance.Delete(m.Name())
	}
	s.Add(ctx, m)
}

// isInMaintenance checks whether the named monitor is currently in a maintenance window.
func (s *Service) isInMaintenance(name string) bool {
	mc, ok := s.maintenance.Load(name)
	if !ok {
		return false
	}
	return IsInMaintenanceWindow(mc, time.Now())
}

// Remove removes a monitor from the service by its name.
// Returns ErrNotFound if the monitor does not exist.
func (s *Service) Remove(m Monitor) error {
	name := m.Name()
	_, ok := s.monitors.Load(name)
	if !ok {
		return ErrNotFound{Name: name}
	}
	s.monitors.Delete(name)
	s.result.Delete(name)         // Remove any pending result immediately
	s.failureCount.Delete(name)   // Remove failure count
	s.maintenance.Delete(name)    // Remove maintenance window
	return nil
}

// Results return a clone of the current monitor results map.
// The returned map can be safely mutated by the caller.
func (s *Service) Results() map[string]Result {
	return xsync.ToPlainMap(s.result)
}

// SetPeriod changes the refresh period and timeout, and restarts the monitor checks.
func (s *Service) SetPeriod(ctx context.Context, period time.Duration, timeout time.Duration) {
	timeout = sanitizeTimeout(timeout, period)
	log.Printf("Setting new period %v and timeout %v", period, timeout)
	s.period.Store(period)
	s.timeout.Store(timeout)

	s.stopMonitors()
	s.startMonitors(ctx)
}

func sanitizeTimeout(timeout time.Duration, period time.Duration) time.Duration {
	if timeout >= period || timeout == 0 {
		timeout = time.Duration(float64(period) * 2 / 3)
	}
	return timeout
}

// stopMonitors stops all monitor routines and waits for them to finish.
func (s *Service) stopMonitors() {
	// Cancel the current monitors.
	log.Printf("Stopping monitors")
	s.cancel()

	// Wait for running monitors to stop.
	s.wg.Wait()
}

// startMonitors launches a goroutine to periodically check all monitors.
// Each check is run with a timeout context based on Service.timeout.
func (s *Service) startMonitors(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.wg.Add(1)
	go func(ctx context.Context) {
		defer s.wg.Done()
		// Validate timeout length
		log.Printf("Starting monitors with period %v and timeout %v", s.period.Load(), s.timeout.Load())
		ticker := time.NewTicker(s.period.Load())
		defer ticker.Stop()
		for {
			to := sanitizeTimeout(s.timeout.Load(), s.period.Load())

			if func() bool {
				toCtx, toCancel := context.WithTimeout(ctx, to)
				defer toCancel()
				s.checkMonitors(toCtx)

				// call cancel to release resources before we start the next tick or on end.
				select {
				case <-ctx.Done():
					return true
				case <-ticker.C: // wait
					return false
				}
			}() {
				return // cascade shutdown
			}
		}
	}(ctx)
}

// checkMonitors checks all monitors and updates the result map.
// Each check runs in its own goroutine in parallel.
// Monitors that are currently in a maintenance window are skipped entirely.
func (s *Service) checkMonitors(ctx context.Context) {
	now := time.Now()
	s.monitors.Range(func(key string, m Monitor) bool {
		if mc, ok := s.maintenance.Load(key); ok && IsInMaintenanceWindow(mc, now) {
			return true // skip this monitor — in maintenance window
		}
		go func(ctx context.Context, m Monitor) {
			result := m.Check(ctx)
			result.DisplayName = m.DisplayName()
			result.Group = m.Group()

			// Store and push result if changed or first non-green
			s.checkStoreAndPush(ctx, m, result)
		}(ctx, m)
		return true
	})
}

// checkStoreAndPush updates the result map and triggers the push callback if needed.
// It tracks consecutive failures and only triggers alerts when the alertThreshold is reached.
func (s *Service) checkStoreAndPush(ctx context.Context, m Monitor, result Result) {
	result.DisplayName = m.DisplayName()
	result.Group = m.Group()
	prev, ok := s.result.Load(m.Name()) // get previous result
	s.result.Store(m.Name(), result)
	
	// Determine if this is a failure (non-green status)
	isFailure := result.Status != RAGGreen
	
	// Get or create atomic counter for this monitor
	counter, _ := s.failureCount.LoadOrStore(m.Name(), &atomic.Int64{})
	
	// Get the alert threshold for this monitor
	threshold := m.AlertThreshold()
	
	// Check current count before updating
	prevCount := counter.Load()
	
	var currentCount int64
	if isFailure {
		// Atomically increment failure count
		currentCount = counter.Add(1)
	} else {
		// Reset failure count on success
		counter.Store(0)
		currentCount = 0
	}
	
	shouldAlert := false
	if s.push != nil {
		// Case 1: Recovery to Green. Alert if we had previously reached the alert threshold
		if ok && prev.Status != RAGGreen && result.Status == RAGGreen {
			if prevCount >= int64(threshold) {
				shouldAlert = true
			}
		}
		// Case 2: Failure threshold is met for the first time.
		if isFailure && currentCount == int64(threshold) {
			shouldAlert = true
		}
	}
	
	if shouldAlert {
		s.push(ctx, m, prev, result)
	}
}

// Size returns the number of monitors currently managed by the service.
func (s *Service) Size() int {
	return s.monitors.Size()
}

// Get retrieves a monitor by its name. Returns nil if the monitor is not found.
func (s *Service) Get(name string) Monitor {
	m, _ := s.monitors.Load(name)
	return m
}

// GetFailureCount retrieves the consecutive failure count for a monitor by its name.
func (s *Service) GetFailureCount(name string) int {
	counter, ok := s.failureCount.Load(name)
	if !ok {
		return 0
	}
	return int(counter.Load())
}

// Save persists the current monitor configuration to the provided config struct.
// It clears disk and healthcheck entries and saves all monitors' configs.
func (s *Service) Save(cfg *config.Config) error {
	// Remove all disks, healthchecks, ping, and docker monitors from config
	cfg.Monitoring.Disk = make(map[string]config.DiskConfig)
	cfg.Monitoring.Healthcheck = make(map[string]config.HealthcheckConfig)
	cfg.Monitoring.Ping = make(map[string]config.PingConfig)
	cfg.Monitoring.Docker = make(map[string]config.DockerConfig)

	// Save all the monitors
	s.monitors.Range(func(key string, m Monitor) bool {
		m.Save(cfg)
		return true
	})
	cfg.Monitoring.Interval = int(s.period.Load() / time.Second)

	// Apply maintenance windows from the service's maintenance map back to the saved config.
	s.maintenance.Range(func(monitorName string, mc *config.MaintenanceConfig) bool {
		if mc == nil {
			return true
		}
		applyMaintenanceToConfig(cfg, monitorName, *mc)
		return true
	})

	return nil
}

// applyMaintenanceToConfig sets the maintenance config on the appropriate config entry
// by parsing the monitor name prefix (e.g., "disk_root" → cfg.Monitoring.Disk["root"]).
func applyMaintenanceToConfig(cfg *config.Config, monitorName string, mc config.MaintenanceConfig) {
	switch {
	case monitorName == "system_cpu":
		cfg.Monitoring.System.CPU.Maintenance = mc
	case monitorName == "system_mem":
		cfg.Monitoring.System.Memory.Maintenance = mc
	default:
		// Monitor names follow the pattern "group_name", e.g. "disk_root", "health_local"
		if name, ok := strings.CutPrefix(monitorName, "disk_"); ok {
			if d, found := cfg.Monitoring.Disk[name]; found {
				d.Maintenance = mc
				cfg.Monitoring.Disk[name] = d
			}
		} else if name, ok := strings.CutPrefix(monitorName, "health_"); ok {
			if h, found := cfg.Monitoring.Healthcheck[name]; found {
				h.Maintenance = mc
				cfg.Monitoring.Healthcheck[name] = h
			}
		} else if name, ok := strings.CutPrefix(monitorName, "ping_"); ok {
			if p, found := cfg.Monitoring.Ping[name]; found {
				p.Maintenance = mc
				cfg.Monitoring.Ping[name] = p
			}
		} else if name, ok := strings.CutPrefix(monitorName, "docker_"); ok {
			if d, found := cfg.Monitoring.Docker[name]; found {
				d.Maintenance = mc
				cfg.Monitoring.Docker[name] = d
			}
		}
	}
}
