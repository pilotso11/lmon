package web

import (
	"lmon/monitor"
)

// MonitorServiceInterface defines the interface for the monitor service
type MonitorServiceInterface interface {
	// GetItems returns all monitored items
	GetItems() []*monitor.Item
	
	// GetItem returns a monitored item by ID
	GetItem(id string) *monitor.Item
	
	// UpdateItem updates a monitored item
	UpdateItem(item *monitor.Item)
	
	// Start starts the monitoring service
	Start()
	
	// Stop stops the monitoring service
	Stop()
}