package monitors

import (
	"context"
	"fmt"
	"maps"
	"sync"
	"time"

	"lmon/config"
)

// ErrNotFound is returned when a monitor is not found.
type ErrNotFound struct {
	Name string
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("monitor %s not found", e.Name)
}

// RAG status for a monitor.
type RAG int

const (
	UNKNOWN RAG = iota
	GREEN
	YELLOW
	RED
	ERROR
)

func (r RAG) String() string {
	switch r {
	case GREEN:
		return "Green"
	case YELLOW:
		return "Yellow"
	case RED:
		return "Red"
	case ERROR:
		return "Error"
	default:
		return "Unknown"
	}
}

// Result of a single monitor check.
type Result struct {
	Key    Monitor
	Status RAG
	Value  string
}

// Monitor interface implemented by all monitors.
type Monitor interface {
	Check(ctx context.Context) Result
	DisplayName() string
	Group() string
	Name() string
	Save(cfg *config.Config)
}

// Service controls the monitoring service.
type Service struct {
	mu       sync.Mutex
	period   time.Duration
	Timeout  time.Duration
	monitors map[string]Monitor
	result   map[string]Result
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewService creates a new monitoring service.
func NewService(ctx context.Context, period time.Duration, timeout time.Duration) *Service {
	s := Service{
		period:   period,
		Timeout:  timeout,
		monitors: make(map[string]Monitor),
		result:   make(map[string]Result),
	}
	s.startMonitors(ctx)
	return &s
}

// Add a monitor to the service.
func (s *Service) Add(m Monitor) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.monitors[m.Name()] = m
}

// Remove a monitor from the service.
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

// Results return a clone of the result map.
func (s *Service) Results() map[string]Result {
	s.mu.Lock()
	defer s.mu.Unlock()

	return maps.Clone(s.result)
}

// SetPeriod changes the refresh period and restarts the monitors with the new period.
func (s *Service) SetPeriod(ctx context.Context, period time.Duration) {
	s.mu.Lock()
	s.period = period
	s.mu.Unlock()

	s.stopMonitors()
	s.startMonitors(ctx)
}

// stopMonitors stops the monitors and waits for them to stop.
func (s *Service) stopMonitors() {
	// cancel the current monitors.
	s.cancel()

	// wait for running monitors to stop.
	s.wg.Wait()
}

// startMonitors starts the monitors in a go routine.
// Monitors are checked Service.period using a ticker.
// Each check is run with a timeout context based on Service.Timeout.
func (s *Service) startMonitors(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.wg.Add(1)
	go func(ctx context.Context) {
		defer s.wg.Done()
		s.mu.Lock()
		ticker := time.NewTicker(s.period)
		s.mu.Unlock()
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Safely clone the timeout
				s.mu.Lock()
				to := s.Timeout
				s.mu.Unlock()
				timeout, _ := context.WithTimeout(ctx, to)
				s.checkMonitors(timeout)
			}
		}
	}(ctx)
}

// checkMonitors checks all monitors and updates the result map.
// each check runs in its own go routine in parallel.
func (s *Service) checkMonitors(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, m := range s.monitors {
		go func(ctx context.Context, m Monitor) {
			result := m.Check(ctx)
			result.Key = m
			s.mu.Lock()
			s.result[m.Name()] = result
			s.mu.Unlock()
		}(ctx, m)
	}
}
