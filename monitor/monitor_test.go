package monitor

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"lmon/config"
)

// mockMonitor is a simple mock implementation of MonitorInterface
type mockMonitor struct {
	mock.Mock
}

func (m *mockMonitor) Check() ([]*Item, error) {
	args := m.Called()
	return args.Get(0).([]*Item), args.Error(1)
}

// mockWebhookSender is a simple mock implementation of WebhookSenderInterface
type mockWebhookSender struct {
	mock.Mock
}

func (m *mockWebhookSender) Send(url string, item *Item) error {
	args := m.Called(url, item)
	return args.Error(0)
}

func TestServiceGetItems(t *testing.T) {
	// Create a test configuration
	cfg := config.DefaultConfig()

	// Create mock monitors
	diskMonitor := new(mockMonitor)
	sysMonitor := new(mockMonitor)
	healthMonitor := new(mockMonitor)
	webhookSender := new(mockWebhookSender)

	// Create service with mock monitors
	service := NewServiceWithMonitors(
		cfg,
		diskMonitor,
		sysMonitor,
		healthMonitor,
		webhookSender,
	)

	// Add some test items
	item1 := &Item{
		ID:        "test-1",
		Name:      "Test Item 1",
		Type:      "test",
		Status:    StatusOK,
		Value:     50.0,
		Threshold: 80.0,
		Unit:      "%",
		Icon:      "test",
		LastCheck: time.Now(),
		Message:   "Test message 1",
	}

	item2 := &Item{
		ID:        "test-2",
		Name:      "Test Item 2",
		Type:      "test",
		Status:    StatusWarning,
		Value:     75.0,
		Threshold: 80.0,
		Unit:      "%",
		Icon:      "test",
		LastCheck: time.Now(),
		Message:   "Test message 2",
	}

	// Update items
	service.UpdateItem(item1)
	service.UpdateItem(item2)

	// Test GetItems
	items := service.GetItems()
	assert.Equal(t, 2, len(items))

	// Test GetItem
	item := service.GetItem("test-1")
	assert.NotNil(t, item)
	assert.Equal(t, "Test Item 1", item.Name)

	item = service.GetItem("test-2")
	assert.NotNil(t, item)
	assert.Equal(t, "Test Item 2", item.Name)

	item = service.GetItem("non-existent")
	assert.Nil(t, item)
}

func TestServiceMonitoring(t *testing.T) {
	// Create a test configuration with short interval
	cfg := config.DefaultConfig()
	cfg.Monitoring.Interval = 1 // 1 second

	// Create mock monitors
	diskMonitor := new(mockMonitor)
	sysMonitor := new(mockMonitor)
	healthMonitor := new(mockMonitor)
	webhookSender := new(mockWebhookSender)

	// Set up expectations
	diskItem := &Item{
		ID:        "disk-test",
		Name:      "Disk Test",
		Type:      "disk",
		Status:    StatusOK,
		Value:     50.0,
		Threshold: 80.0,
		Unit:      "%",
		Icon:      "storage",
		LastCheck: time.Now(),
		Message:   "50.0% used",
	}

	sysItems := []*Item{
		{
			ID:        "cpu",
			Name:      "CPU Usage",
			Type:      "cpu",
			Status:    StatusOK,
			Value:     30.0,
			Threshold: 80.0,
			Unit:      "%",
			Icon:      "speed",
			LastCheck: time.Now(),
			Message:   "30.0% used",
		},
		{
			ID:        "memory",
			Name:      "Memory Usage",
			Type:      "memory",
			Status:    StatusOK,
			Value:     40.0,
			Threshold: 80.0,
			Unit:      "%",
			Icon:      "memory",
			LastCheck: time.Now(),
			Message:   "40.0% used",
		},
	}

	healthItem := &Item{
		ID:        "health-test",
		Name:      "Health Test",
		Type:      "health",
		Status:    StatusOK,
		Value:     200.0,
		Threshold: 200.0,
		Unit:      "status",
		Icon:      "health_and_safety",
		LastCheck: time.Now(),
		Message:   "Service is healthy",
	}

	// Set up expectations
	diskMonitor.On("Check").Return([]*Item{diskItem}, nil)
	sysMonitor.On("Check").Return(sysItems, nil)
	healthMonitor.On("Check").Return([]*Item{healthItem}, nil)

	// Create service with mock monitors
	service := NewServiceWithMonitors(
		cfg,
		diskMonitor,
		sysMonitor,
		healthMonitor,
		webhookSender,
	)

	// Start monitoring
	service.Start()

	// Wait for monitoring to run
	time.Sleep(time.Second * 2)

	// Stop monitoring
	service.Stop()

	// Verify items were updated
	items := service.GetItems()
	require.GreaterOrEqual(t, len(items), 4) // At least 4 items (1 disk, 2 system, 1 health)

	// Check for specific items
	diskTestItem := service.GetItem("disk-test")
	assert.NotNil(t, diskTestItem)
	assert.Equal(t, "Disk Test", diskTestItem.Name)

	cpuItem := service.GetItem("cpu")
	assert.NotNil(t, cpuItem)
	assert.Equal(t, "CPU Usage", cpuItem.Name)

	memoryItem := service.GetItem("memory")
	assert.NotNil(t, memoryItem)
	assert.Equal(t, "Memory Usage", memoryItem.Name)

	healthTestItem := service.GetItem("health-test")
	assert.NotNil(t, healthTestItem)
	assert.Equal(t, "Health Test", healthTestItem.Name)
}

func TestNotifyUnhealthy(t *testing.T) {
	// Create a test configuration with webhook disabled
	cfg := config.DefaultConfig()
	cfg.Webhook.Enabled = false

	// Create mock monitors
	diskMonitor := new(mockMonitor)
	sysMonitor := new(mockMonitor)
	healthMonitor := new(mockMonitor)
	webhookSender := new(mockWebhookSender)

	// Create service with mock monitors
	service := NewServiceWithMonitors(
		cfg,
		diskMonitor,
		sysMonitor,
		healthMonitor,
		webhookSender,
	)

	// Create a critical item
	criticalItem := &Item{
		ID:        "test-critical",
		Name:      "Test Critical",
		Type:      "test",
		Status:    StatusCritical,
		Value:     90.0,
		Threshold: 80.0,
		Unit:      "%",
		Icon:      "test",
		LastCheck: time.Now(),
		Message:   "Critical test message",
	}

	// Call notifyUnhealthy with webhook disabled
	service.notifyUnhealthy(criticalItem)

	// Verify webhook was not called
	webhookSender.AssertNotCalled(t, "Send")

	// Enable webhook
	service.config.Webhook.Enabled = true
	service.config.Webhook.URL = "https://example.com/webhook"

	// Expect webhook to be called
	webhookSender.On("Send", service.config.Webhook.URL, criticalItem).Return(nil)

	// Call notifyUnhealthy with webhook enabled
	service.notifyUnhealthy(criticalItem)

	// Wait for goroutine to complete
	time.Sleep(time.Millisecond * 100)

	// Verify webhook was called
	webhookSender.AssertExpectations(t)
}

func TestServiceWebhookNotification(t *testing.T) {
	// Create a test configuration with webhook enabled
	cfg := config.DefaultConfig()
	cfg.Webhook.Enabled = true
	cfg.Webhook.URL = "https://example.com/webhook"

	// Create mock monitors
	diskMonitor := new(mockMonitor)
	sysMonitor := new(mockMonitor)
	healthMonitor := new(mockMonitor)
	webhookSender := new(mockWebhookSender)

	// Create service with mock monitors
	service := NewServiceWithMonitors(
		cfg,
		diskMonitor,
		sysMonitor,
		healthMonitor,
		webhookSender,
	)

	// Create a critical item
	criticalItem := &Item{
		ID:        "test-critical",
		Name:      "Test Critical",
		Type:      "test",
		Status:    StatusCritical,
		Value:     90.0,
		Threshold: 80.0,
		Unit:      "%",
		Icon:      "test",
		LastCheck: time.Now(),
		Message:   "Critical test message",
	}

	// Expect webhook to be called and succeed
	webhookSender.On("Send", cfg.Webhook.URL, criticalItem).Return(nil)

	// Update item (should trigger webhook)
	service.UpdateItem(criticalItem)

	// Wait for goroutine to complete
	time.Sleep(time.Millisecond * 100)

	// Verify webhook was called
	webhookSender.AssertCalled(t, "Send", cfg.Webhook.URL, criticalItem)

	// Create a warning item
	warningItem := &Item{
		ID:        "test-warning",
		Name:      "Test Warning",
		Type:      "test",
		Status:    StatusWarning,
		Value:     75.0,
		Threshold: 80.0,
		Unit:      "%",
		Icon:      "test",
		LastCheck: time.Now(),
		Message:   "Warning test message",
	}

	// Expect webhook to be called and fail (simulate error)
	webhookSender.On("Send", cfg.Webhook.URL, warningItem).Return(fmt.Errorf("webhook error"))

	// Update item (should trigger webhook, but with error)
	service.UpdateItem(warningItem)

	// Wait for goroutine to complete
	time.Sleep(time.Millisecond * 100)

	// Verify webhook was called even though it errored
	webhookSender.AssertCalled(t, "Send", cfg.Webhook.URL, warningItem)

	// Create an OK item
	okItem := &Item{
		ID:        "test-ok",
		Name:      "Test OK",
		Type:      "test",
		Status:    StatusOK,
		Value:     50.0,
		Threshold: 80.0,
		Unit:      "%",
		Icon:      "test",
		LastCheck: time.Now(),
		Message:   "OK test message",
	}

	// Update item (should NOT trigger webhook)
	service.UpdateItem(okItem)

	// Wait for goroutine to complete
	time.Sleep(time.Millisecond * 100)

	// Verify webhook was not called for OK item
	webhookSender.AssertNotCalled(t, "Send", cfg.Webhook.URL, okItem)

	// Disable webhook and ensure nothing is sent
	service.config.Webhook.Enabled = false
	criticalItem2 := &Item{
		ID:        "test-critical-2",
		Name:      "Test Critical 2",
		Type:      "test",
		Status:    StatusCritical,
		Value:     95.0,
		Threshold: 80.0,
		Unit:      "%",
		Icon:      "test",
		LastCheck: time.Now(),
		Message:   "Critical test message 2",
	}
	service.UpdateItem(criticalItem2)
	time.Sleep(time.Millisecond * 100)
	webhookSender.AssertNotCalled(t, "Send", cfg.Webhook.URL, criticalItem2)

	// Enable webhook but empty URL, should not send
	service.config.Webhook.Enabled = true
	service.config.Webhook.URL = ""
	criticalItem3 := &Item{
		ID:        "test-critical-3",
		Name:      "Test Critical 3",
		Type:      "test",
		Status:    StatusCritical,
		Value:     99.0,
		Threshold: 80.0,
		Unit:      "%",
		Icon:      "test",
		LastCheck: time.Now(),
		Message:   "Critical test message 3",
	}
	service.UpdateItem(criticalItem3)
	time.Sleep(time.Millisecond * 100)
	webhookSender.AssertNotCalled(t, "Send", "", criticalItem3)
}

func TestWebhookOnStartup(t *testing.T) {
	// Create a test configuration with webhook enabled
	cfg := config.DefaultConfig()
	cfg.Webhook.Enabled = true
	cfg.Webhook.URL = "https://example.com/webhook"

	// Create mock monitors
	diskMonitor := new(mockMonitor)
	sysMonitor := new(mockMonitor)
	healthMonitor := new(mockMonitor)
	webhookSender := new(mockWebhookSender)

	// Create critical items that will be returned by the monitors
	criticalDiskItem := &Item{
		ID:        "disk-critical",
		Name:      "Disk Critical",
		Type:      "disk",
		Status:    StatusCritical,
		Value:     95.0,
		Threshold: 80.0,
		Unit:      "%",
		Icon:      "storage",
		LastCheck: time.Now(),
		Message:   "Disk usage critical",
	}

	criticalMemoryItem := &Item{
		ID:        "memory",
		Name:      "Memory Usage",
		Type:      "memory",
		Status:    StatusCritical,
		Value:     95.0,
		Threshold: 90.0,
		Unit:      "%",
		Icon:      "memory",
		LastCheck: time.Now(),
		Message:   "Memory usage critical",
	}

	// Set up expectations for the initial checks
	diskMonitor.On("Check").Return([]*Item{criticalDiskItem}, nil)
	sysMonitor.On("Check").Return([]*Item{criticalMemoryItem}, nil)
	healthMonitor.On("Check").Return([]*Item{}, nil)

	// Expect webhook to be called for both critical items
	webhookSender.On("Send", cfg.Webhook.URL, criticalDiskItem).Return(nil)
	webhookSender.On("Send", cfg.Webhook.URL, criticalMemoryItem).Return(nil)

	// Create service with mock monitors
	service := NewServiceWithMonitors(
		cfg,
		diskMonitor,
		sysMonitor,
		healthMonitor,
		webhookSender,
	)

	// Start the service (should trigger initial checks and webhooks)
	service.Start()

	// Wait for goroutines to complete
	time.Sleep(time.Millisecond * 500)

	// Stop the service
	service.Stop()

	// Verify webhook was called for critical items
	webhookSender.AssertExpectations(t)
}

func TestRefreshChecksHandlesErrors(t *testing.T) {
	// This test verifies that RefreshChecks handles errors from monitors gracefully

	cfg := config.DefaultConfig()

	// Use error monitors for all
	errorMonitorInst := &errorMonitor{}
	service := NewServiceWithMonitors(
		cfg,
		errorMonitorInst,
		errorMonitorInst,
		errorMonitorInst,
		new(mockWebhookSender),
	)

	// Should not panic even though all checks fail
	service.RefreshChecks()
}

func TestRefreshChecksUpdatesMonitors(t *testing.T) {
	// Create a test configuration
	cfg := config.DefaultConfig()
	cfg.Monitoring.System.Memory.Threshold = 80

	// Create a service with the initial configuration
	service := NewService(cfg)

	// Modify the configuration
	cfg.Monitoring.System.Memory.Threshold = 90

	// Call RefreshChecks to apply the new configuration
	service.RefreshChecks()

	// Get the memory item
	items, err := service.sysMonitor.Check()
	require.NoError(t, err)

	// Find the memory item
	var memoryItem *Item
	for _, item := range items {
		if item.ID == "memory" {
			memoryItem = item
			break
		}
	}

	// Verify the memory threshold was updated
	require.NotNil(t, memoryItem)
	assert.Equal(t, float64(90), memoryItem.Threshold)
}

func TestService_ConcurrentUpdateAndGetItems(t *testing.T) {
	cfg := config.DefaultConfig()
	service := NewService(cfg)

	numItems := 100
	numGoroutines := 10

	done := make(chan struct{})
	for i := 0; i < numGoroutines; i++ {
		go func(gid int) {
			for j := 0; j < numItems; j++ {
				item := &Item{
					ID:        fmt.Sprintf("item-%d-%d", gid, j),
					Name:      fmt.Sprintf("Item %d-%d", gid, j),
					Type:      "test",
					Status:    StatusOK,
					Value:     float64(j),
					Threshold: 100.0,
					Unit:      "%",
					Icon:      "test",
					LastCheck: time.Now(),
					Message:   "Concurrent test",
				}
				service.UpdateItem(item)
			}
			done <- struct{}{}
		}(i)
	}

	// Concurrently read items
	readDone := make(chan struct{})
	go func() {
		for i := 0; i < numItems*numGoroutines; i++ {
			_ = service.GetItems()
		}
		readDone <- struct{}{}
	}()

	// Wait for all writers
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
	<-readDone

	// Final check: all items should be present
	items := service.GetItems()
	assert.GreaterOrEqual(t, len(items), numItems*numGoroutines)
}

// errorMonitor implements all monitor interfaces and always returns an error.
type errorMonitor struct{}

func (e *errorMonitor) Check() ([]*Item, error) {
	return nil, fmt.Errorf("simulated error")
}

func TestPeriodicMonitorGoroutines_HandleErrors(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Monitoring.Interval = 1 // fast tick

	// Use error monitors for all
	errorMonitorInst := &errorMonitor{}
	service := NewServiceWithMonitors(
		cfg,
		errorMonitorInst,
		errorMonitorInst,
		errorMonitorInst,
		new(mockWebhookSender),
	)

	// Start the periodic goroutines (should not panic or deadlock)
	go service.monitorDisk()
	go service.monitorSystem()
	go service.monitorHealthChecks()

	// Let them run for a short while
	time.Sleep(2 * time.Second)

	// Stop the service to end goroutines
	service.Stop()

	// If we got here without panic or deadlock, test passes
}
