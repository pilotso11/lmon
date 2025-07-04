package monitor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"lmon/config"
)

// MockDiskMonitor is a mock implementation of DiskMonitorInterface
type MockDiskMonitor struct {
	mock.Mock
}

func (m *MockDiskMonitor) Check() ([]*Item, error) {
	args := m.Called()
	return args.Get(0).([]*Item), args.Error(1)
}

// MockSystemMonitor is a mock implementation of SystemMonitorInterface
type MockSystemMonitor struct {
	mock.Mock
}

func (m *MockSystemMonitor) Check() ([]*Item, error) {
	args := m.Called()
	return args.Get(0).([]*Item), args.Error(1)
}

// MockHealthMonitor is a mock implementation of HealthMonitorInterface
type MockHealthMonitor struct {
	mock.Mock
}

func (m *MockHealthMonitor) Check() ([]*Item, error) {
	args := m.Called()
	return args.Get(0).([]*Item), args.Error(1)
}

// MockWebhookSender is a mock implementation of WebhookSenderInterface
type MockWebhookSender struct {
	mock.Mock
}

func (m *MockWebhookSender) Send(url string, item *Item) error {
	args := m.Called(url, item)
	return args.Error(0)
}

// TestInitialPass tests that the monitoring service runs an initial check immediately after starting
func TestInitialPass(t *testing.T) {
	// Create mock monitors
	mockDiskMonitor := new(MockDiskMonitor)
	mockSystemMonitor := new(MockSystemMonitor)
	mockHealthMonitor := new(MockHealthMonitor)
	mockWebhookSender := new(MockWebhookSender)

	// Set up expectations for initial checks
	diskItems := []*Item{
		{ID: "disk:/", Name: "Disk /", Status: StatusOK},
	}
	mockDiskMonitor.On("Check").Return(diskItems, nil)

	sysItems := []*Item{
		{ID: "cpu", Name: "CPU", Status: StatusOK},
		{ID: "memory", Name: "Memory", Status: StatusOK},
	}
	mockSystemMonitor.On("Check").Return(sysItems, nil)

	healthItems := []*Item{
		{ID: "health:test", Name: "Test Health", Status: StatusOK},
	}
	mockHealthMonitor.On("Check").Return(healthItems, nil)

	// Create test configuration
	cfg := config.DefaultConfig()

	// Create monitoring service with mocks
	service := NewServiceWithMonitors(
		cfg,
		mockDiskMonitor,
		mockSystemMonitor,
		mockHealthMonitor,
		mockWebhookSender,
	)

	// Start the service
	service.Start()

	// Give a short time for the initial checks to complete
	time.Sleep(100 * time.Millisecond)

	// Stop the service
	service.Stop()

	// Verify that all initial checks were called
	mockDiskMonitor.AssertCalled(t, "Check")
	mockSystemMonitor.AssertCalled(t, "Check")
	mockHealthMonitor.AssertCalled(t, "Check")

	// Verify that items were added to the service
	items := service.GetItems()
	assert.Len(t, items, 4) // 1 disk + 2 system + 1 health

	// Verify specific items
	diskItem := service.GetItem("disk:/")
	assert.NotNil(t, diskItem)
	assert.Equal(t, "Disk /", diskItem.Name)

	cpuItem := service.GetItem("cpu")
	assert.NotNil(t, cpuItem)
	assert.Equal(t, "CPU", cpuItem.Name)

	memoryItem := service.GetItem("memory")
	assert.NotNil(t, memoryItem)
	assert.Equal(t, "Memory", memoryItem.Name)

	healthItem := service.GetItem("health:test")
	assert.NotNil(t, healthItem)
	assert.Equal(t, "Test Health", healthItem.Name)
}
