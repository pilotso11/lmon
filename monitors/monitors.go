// Package monitors provides the core monitoring service and abstractions for lmon.
// It defines monitor interfaces, result types, and the Service that manages monitor lifecycles.
package monitors

import (
	"context"
	"fmt"
	"log"
	"maps"
	"sync"
	"time"

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
}

// PushFunc is a callback called when a monitor's result changes.
// It receives the monitor, previous result, and new result.
type PushFunc func(ctx context.Context, m Monitor, prev, result Result)

// Service manages the lifecycle of monitors, periodic checks, and result storage.
// It is safe for concurrent use.
type Service struct {
	mu       sync.Mutex
	period   time.Duration
	timeout  time.Duration
	monitors map[string]Monitor
	result   map[string]Result
	cancel   context.CancelFunc
	push     PushFunc
	wg       sync.WaitGroup
}

// NewService creates a new monitoring Service.
// period: how often to check all monitors.
// timeout: maximum duration for each check.
// push: callback for result changes (may be nil).
func NewService(ctx context.Context, period time.Duration, timeout time.Duration, push PushFunc) *Service {
	s := Service{
		period:   period,
		timeout:  timeout,
		monitors: make(map[string]Monitor),
		result:   make(map[string]Result),
		push:     push,
	}
	s.startMonitors(ctx)
	return &s
}

// SetPush sets or clears the push callback function.
// If push is nil, no callback will be invoked on result changes.
func (s *Service) SetPush(push PushFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.push = push
}

// Add adds a monitor to the service and performs an initial check.
// Returns an error if the initial check fails.
func (s *Service) Add(ctx context.Context, m Monitor) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.monitors[m.Name()] = m

	// Synchronously check so any error can be reported back to the user
	result := m.Check(ctx)
	s.checkStoreAndPush(ctx, m, result)
	if result.Status == RAGError {
		return fmt.Errorf("error adding monitor %s: %s", m.DisplayName(), result.Value)
	}
	return nil
}

// Remove removes a monitor from the service by its name.
// Returns ErrNotFound if the monitor does not exist.
func (s *Service) Remove(m Monitor) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.monitors[m.Name()]
	if !ok {
		return ErrNotFound{Name: m.Name()}
	}
	delete(s.monitors, m.Name())
	return nil
}

// Results return a clone of the current monitor results map.
// The returned map can be safely mutated by the caller.
func (s *Service) Results() map[string]Result {
	s.mu.Lock()
	defer s.mu.Unlock()

	return maps.Clone(s.result)
}

// SetPeriod changes the refresh period and timeout, and restarts the monitor checks.
func (s *Service) SetPeriod(ctx context.Context, period time.Duration, timeout time.Duration) {
	s.mu.Lock()
	s.period = period
	s.timeout = timeout
	s.mu.Unlock()

	s.stopMonitors()
	s.startMonitors(ctx)
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
		s.mu.Lock()
		// Validate timeout length
		if s.timeout > s.period/2 || s.timeout == 0 {
			s.timeout = s.period / 2
		}
		log.Printf("Starting monitors with period %v and timeout %v", s.period, s.timeout)
		ticker := time.NewTicker(s.period)
		s.mu.Unlock()
		defer ticker.Stop()
		for {
			// Safely clone the timeout
			s.mu.Lock()
			to := s.timeout
			s.mu.Unlock()
			timeout, toCancel := context.WithTimeout(ctx, to)
			s.checkMonitors(timeout)
			toCancel()

			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Wait until 1 timeout
				time.Sleep(to + -1*time.Millisecond)
			}
		}
	}(ctx)
}

// checkMonitors checks all monitors and updates the result map.
// Each check runs in its own goroutine in parallel.
func (s *Service) checkMonitors(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, m := range s.monitors {
		go func(ctx context.Context, m Monitor) {
			result := m.Check(ctx)
			result.DisplayName = m.DisplayName()
			result.Group = m.Group()

			// Store and push result if changed or first non-green
			s.mu.Lock()
			defer s.mu.Unlock()
			s.checkStoreAndPush(ctx, m, result)
		}(ctx, m)
	}
}

// checkStoreAndPush updates the result map and triggers the push callback if needed.
func (s *Service) checkStoreAndPush(ctx context.Context, m Monitor, result Result) {
	result.DisplayName = m.DisplayName()
	result.Group = m.Group()
	prev, ok := s.result[m.Name()] // get previous result
	s.result[m.Name()] = result
	switch {
	case ok && prev.Status != result.Status && s.push != nil,
		!ok && result.Status != RAGGreen && s.push != nil:
		s.push(ctx, m, prev, result)
	}
}

// Size returns the number of monitors currently managed by the service.
func (s *Service) Size() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.monitors)
}

// Save persists the current monitor configuration to the provided config struct.
// It clears disk and healthcheck entries and saves all monitors' configs.
func (s *Service) Save(cfg *config.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove all disks and healthchecks from config
	cfg.Monitoring.Disk = make(map[string]config.DiskConfig)
	cfg.Monitoring.Healthcheck = make(map[string]config.HealthcheckConfig)

	// Save all the monitors
	for _, m := range s.monitors {
		m.Save(cfg)
	}
	cfg.Monitoring.Interval = int(s.period / time.Second)
	return nil
}
