package monitor

import (
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

	// Expect webhook to be called
	webhookSender.On("Send", cfg.Webhook.URL, criticalItem).Return(nil)

	// Update item (should trigger webhook)
	service.UpdateItem(criticalItem)

	// Wait for goroutine to complete
	time.Sleep(time.Millisecond * 100)

	// Verify webhook was called
	webhookSender.AssertExpectations(t)

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

	// Expect webhook to be called
	webhookSender.On("Send", cfg.Webhook.URL, warningItem).Return(nil)

	// Update item (should trigger webhook)
	service.UpdateItem(warningItem)

	// Wait for goroutine to complete
	time.Sleep(time.Millisecond * 100)

	// Verify webhook was called
	webhookSender.AssertExpectations(t)

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
}
