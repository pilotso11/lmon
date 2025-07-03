package monitor

import (
	"net/http"
)

// MonitorInterface defines the interface for all monitors
type MonitorInterface interface {
	Check() ([]*Item, error)
}

// DiskMonitorInterface defines the interface for disk monitors
type DiskMonitorInterface interface {
	MonitorInterface
}

// SystemMonitorInterface defines the interface for system monitors
type SystemMonitorInterface interface {
	MonitorInterface
}

// HealthMonitorInterface defines the interface for health check monitors
type HealthMonitorInterface interface {
	MonitorInterface
}

// HTTPClientInterface defines the interface for HTTP clients
type HTTPClientInterface interface {
	Do(req *http.Request) (*http.Response, error)
}

// WebhookSenderInterface defines the interface for webhook senders
type WebhookSenderInterface interface {
	Send(url string, item *Item) error
}