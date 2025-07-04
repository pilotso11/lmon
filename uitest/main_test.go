package uitest

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"lmon/config"
	"lmon/monitor"
	"lmon/web"
)

// MockMonitorService is a mock implementation of the monitor service
type MockMonitorService struct {
	mock.Mock
}

// GetItems is a mock implementation of the GetItems method
func (m *MockMonitorService) GetItems() []*monitor.Item {
	args := m.Called()
	return args.Get(0).([]*monitor.Item)
}

// GetItem is a mock implementation of the GetItem method
func (m *MockMonitorService) GetItem(id string) *monitor.Item {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*monitor.Item)
}

// UpdateItem is a mock implementation of the UpdateItem method
func (m *MockMonitorService) UpdateItem(item *monitor.Item) {
	m.Called(item)
}

// Start is a mock implementation of the Start method
func (m *MockMonitorService) Start() {
	m.Called()
}

// Stop is a mock implementation of the Stop method
func (m *MockMonitorService) Stop() {
	m.Called()
}

// TestMain is the entry point for all tests in this package
func TestMain(m *testing.M) {
	// Check for a flag that indicates we should skip browser tests
	// We can't use testing.Short() here because it's not initialized yet
	for _, arg := range os.Args {
		if arg == "-test.short=true" || arg == "-test.short" {
			log.Println("Skipping UI tests in short mode")
			os.Exit(0)
		}
	}

	// Create a context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a test configuration
	cfg := config.DefaultConfig()
	cfg.Web.Port = 8080 // Use the port expected by the UI tests

	// Create a mock monitor service
	mockService := new(MockMonitorService)

	// Set up mock expectations
	mockService.On("Start").Return()
	mockService.On("Stop").Return()
	mockService.On("GetItems").Return([]*monitor.Item{
		{
			ID:        "cpu",
			Name:      "CPU Usage",
			Type:      "cpu",
			Status:    monitor.StatusOK,
			Value:     10.5,
			Threshold: 80.0,
			Unit:      "%",
			Icon:      "speed",
			LastCheck: time.Now(),
			Message:   "CPU usage is OK",
		},
		{
			ID:        "memory",
			Name:      "Memory Usage",
			Type:      "memory",
			Status:    monitor.StatusOK,
			Value:     20.5,
			Threshold: 80.0,
			Unit:      "%",
			Icon:      "memory",
			LastCheck: time.Now(),
			Message:   "Memory usage is OK",
		},
		{
			ID:        "disk-root",
			Name:      "Disk Usage (/)",
			Type:      "disk",
			Status:    monitor.StatusOK,
			Value:     30.5,
			Threshold: 80.0,
			Unit:      "%",
			Icon:      "storage",
			LastCheck: time.Now(),
			Message:   "Disk usage is OK",
		},
	})
	mockService.On("GetItem", mock.Anything).Return(nil)

	// Create a web server with the mock service
	webServer := web.NewServerWithContext(ctx, cfg, mockService)

	// Start the web server in a goroutine
	go func() {
		if err := webServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Web server error: %v", err)
		}
	}()

	// Wait for the server to start
	if err := waitForServer(fmt.Sprintf("http://localhost:%d", cfg.Web.Port), 30*time.Second); err != nil {
		log.Fatalf("Server did not start: %v", err)
	}

	// Run the tests
	exitCode := m.Run()

	// Shut down the server
	cancel() // Cancel the context to trigger shutdown
	if err := webServer.Stop(); err != nil {
		log.Printf("Error shutting down server: %v", err)
	}

	// Exit with the test result
	os.Exit(exitCode)
}

// waitForServer waits for the server to be ready
func waitForServer(url string, timeout time.Duration) error {
	start := time.Now()
	for {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		if time.Since(start) > timeout {
			return fmt.Errorf("server did not respond within %s", timeout)
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// TestServerHealth tests that the server is healthy
func TestServerHealth(t *testing.T) {
	resp, err := http.Get("http://localhost:8080")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}
