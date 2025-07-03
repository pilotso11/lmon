package monitor

import (
	"context"
	"log"
	"sync"
	"time"

	"lmon/config"
)

// Status represents the status of a monitored item
type Status string

const (
	// StatusOK indicates the monitored item is healthy
	StatusOK Status = "OK"
	// StatusWarning indicates the monitored item is in a warning state
	StatusWarning Status = "WARNING"
	// StatusCritical indicates the monitored item is in a critical state
	StatusCritical Status = "CRITICAL"
	// StatusUnknown indicates the status of the monitored item is unknown
	StatusUnknown Status = "UNKNOWN"
)

// Item represents a monitored item
type Item struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Status    Status    `json:"status"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
	Unit      string    `json:"unit"`
	Icon      string    `json:"icon"`
	LastCheck time.Time `json:"last_check"`
	Message   string    `json:"message"`
}

// Service represents the monitoring service
type Service struct {
	config        *config.Config
	items         map[string]*Item
	mutex         sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	diskMonitor   DiskMonitorInterface
	sysMonitor    SystemMonitorInterface
	healthMonitor HealthMonitorInterface
	webhookSender WebhookSenderInterface
}

// NewService creates a new monitoring service
func NewService(cfg *config.Config) *Service {
	return NewServiceWithContext(context.Background(), cfg)
}

// NewServiceWithContext creates a new monitoring service with the provided context
func NewServiceWithContext(parentCtx context.Context, cfg *config.Config) *Service {
	ctx, cancel := context.WithCancel(parentCtx)

	return &Service{
		config:        cfg,
		items:         make(map[string]*Item),
		ctx:           ctx,
		cancel:        cancel,
		diskMonitor:   NewDiskMonitor(cfg),
		sysMonitor:    NewSystemMonitor(cfg),
		healthMonitor: NewHealthMonitor(cfg),
		webhookSender: newDefaultWebhookSender(),
	}
}

// NewServiceWithMonitors creates a new monitoring service with custom monitors
func NewServiceWithMonitors(
	cfg *config.Config,
	diskMonitor DiskMonitorInterface,
	sysMonitor SystemMonitorInterface,
	healthMonitor HealthMonitorInterface,
	webhookSender WebhookSenderInterface,
) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	return &Service{
		config:        cfg,
		items:         make(map[string]*Item),
		ctx:           ctx,
		cancel:        cancel,
		diskMonitor:   diskMonitor,
		sysMonitor:    sysMonitor,
		healthMonitor: healthMonitor,
		webhookSender: webhookSender,
	}
}

// Start starts the monitoring service
func (s *Service) Start() {
	log.Println("Starting monitoring service")

	// Start disk monitoring
	go s.monitorDisk()

	// Start system monitoring
	go s.monitorSystem()

	// Start health check monitoring
	go s.monitorHealthChecks()
}

// Stop stops the monitoring service
func (s *Service) Stop() {
	log.Println("Stopping monitoring service")
	s.cancel()
}

// GetItems returns all monitored items
func (s *Service) GetItems() []*Item {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	items := make([]*Item, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}

	return items
}

// GetItem returns a monitored item by ID
func (s *Service) GetItem(id string) *Item {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.items[id]
}

// UpdateItem updates a monitored item
func (s *Service) UpdateItem(item *Item) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.items[item.ID] = item

	// Check if status changed to unhealthy
	if item.Status == StatusWarning || item.Status == StatusCritical {
		s.notifyUnhealthy(item)
	}
}

// notifyUnhealthy sends a notification for an unhealthy item
func (s *Service) notifyUnhealthy(item *Item) {
	if s.config.Webhook.Enabled && s.config.Webhook.URL != "" {
		go func() {
			if err := s.webhookSender.Send(s.config.Webhook.URL, item); err != nil {
				log.Printf("Failed to send webhook notification: %v", err)
			}
		}()
	}
}

// monitorDisk monitors disk space
func (s *Service) monitorDisk() {
	ticker := time.NewTicker(time.Duration(s.config.Monitoring.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			items, err := s.diskMonitor.Check()
			if err != nil {
				log.Printf("Disk monitoring error: %v", err)
				continue
			}

			for _, item := range items {
				s.UpdateItem(item)
			}
		}
	}
}

// monitorSystem monitors CPU and memory usage
func (s *Service) monitorSystem() {
	ticker := time.NewTicker(time.Duration(s.config.Monitoring.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			items, err := s.sysMonitor.Check()
			if err != nil {
				log.Printf("System monitoring error: %v", err)
				continue
			}

			for _, item := range items {
				s.UpdateItem(item)
			}
		}
	}
}

// monitorHealthChecks monitors health checks
func (s *Service) monitorHealthChecks() {
	ticker := time.NewTicker(time.Duration(s.config.Monitoring.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			items, err := s.healthMonitor.Check()
			if err != nil {
				log.Printf("Health check monitoring error: %v", err)
				continue
			}

			for _, item := range items {
				s.UpdateItem(item)
			}
		}
	}
}
