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
	RAGUnknown RAG = iota
	RAGGreen
	RAGAmber
	RAGRed
	RAGError
)

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

// PushFunc is called when a monitor result changes.
type PushFunc func(ctx context.Context, m Monitor, prev, result Result)

// Service controls the monitoring service.
type Service struct {
	mu       sync.Mutex
	period   time.Duration
	Timeout  time.Duration
	Monitors map[string]Monitor
	result   map[string]Result
	cancel   context.CancelFunc
	push     PushFunc
	wg       sync.WaitGroup
}

// NewService creates a new monitoring service.
func NewService(ctx context.Context, period time.Duration, timeout time.Duration, push PushFunc) *Service {
	s := Service{
		period:   period,
		Timeout:  timeout,
		Monitors: make(map[string]Monitor),
		result:   make(map[string]Result),
		push:     push,
	}
	s.startMonitors(ctx)
	return &s
}

// SetPush sets or clears the push function if nil is passed.
func (s *Service) SetPush(push PushFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.push = push
}

// Add a monitor to the service.
func (s *Service) Add(ctx context.Context, m Monitor) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Monitors[m.Name()] = m

	// synchronously so any error can be reported back to the user
	result := m.Check(ctx)
	s.checkStoreAndPush(ctx, m, result)
	if result.Status == RAGError {
		return fmt.Errorf("error adding monitor %s: %s", m.DisplayName(), result.Value)
	}
	return nil
}

// Remove a monitor from the service.
func (s *Service) Remove(m Monitor) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.Monitors[m.Name()]
	if !ok {
		return ErrNotFound{Name: m.Name()}
	}
	delete(s.Monitors, m.Name())
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

// stopMonitors stops the Monitors and waits for them to stop.
func (s *Service) stopMonitors() {
	// cancel the current monitors.
	s.cancel()

	// wait for running monitors to stop.
	s.wg.Wait()
}

// startMonitors starts the Monitors in a go routine.
// Monitors are checked Service.period using a ticker.
// Each check is run with a timeout context based on Service.Timeout.
func (s *Service) startMonitors(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.wg.Add(1)
	go func(ctx context.Context) {
		defer s.wg.Done()
		s.mu.Lock()
		// validate timeout length
		if s.Timeout > s.period/2 || s.Timeout == 0 {
			s.Timeout = s.period / 2
		}
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
				timeout, toCancel := context.WithTimeout(ctx, to)
				s.checkMonitors(timeout)

				// wait until 1 timeout
				time.Sleep(to + -1*time.Millisecond)
				toCancel()
			}
		}
	}(ctx)
}

// checkMonitors checks all Monitors and updates the result map.
// each check runs in its own go routine in parallel.
func (s *Service) checkMonitors(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, m := range s.Monitors {
		go func(ctx context.Context, m Monitor) {
			result := m.Check(ctx)
			result.Key = m

			// check the result
			// if status is changed push
			// or first result and not green
			s.mu.Lock()
			defer s.mu.Unlock()
			s.checkStoreAndPush(ctx, m, result)
		}(ctx, m)
	}
}

func (s *Service) checkStoreAndPush(ctx context.Context, m Monitor, result Result) {
	result.Key = m
	prev, ok := s.result[m.Name()] // get previous result
	s.result[m.Name()] = result
	switch {
	case ok && prev.Status != result.Status && s.push != nil,
		!ok && result.Status != RAGGreen && s.push != nil:
		s.push(ctx, m, prev, result)
	}
}
