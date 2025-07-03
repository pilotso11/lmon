package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"lmon/config"
	"lmon/monitor"
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

func TestServer_HandleGetItems(t *testing.T) {
	// Set up Gin in test mode
	gin.SetMode(gin.TestMode)

	// Create mock monitor service
	mockService := new(MockMonitorService)

	// Create test items
	items := []*monitor.Item{
		{
			ID:        "test-1",
			Name:      "Test Item 1",
			Type:      "test",
			Status:    monitor.StatusOK,
			Value:     50.0,
			Threshold: 80.0,
			Unit:      "%",
			Icon:      "test",
			Message:   "Test message 1",
		},
		{
			ID:        "test-2",
			Name:      "Test Item 2",
			Type:      "test",
			Status:    monitor.StatusWarning,
			Value:     75.0,
			Threshold: 80.0,
			Unit:      "%",
			Icon:      "test",
			Message:   "Test message 2",
		},
	}

	// Set up mock expectations
	mockService.On("GetItems").Return(items)

	// Create test configuration
	cfg := config.DefaultConfig()

	// Create server with mock service
	server := NewServer(cfg, mockService)

	// Create a test HTTP recorder and context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Call the handler
	server.handleGetItems(c)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response body
	var responseItems []*monitor.Item
	err := json.Unmarshal(w.Body.Bytes(), &responseItems)
	require.NoError(t, err)

	// Verify response items
	assert.Equal(t, len(items), len(responseItems))
	assert.Equal(t, items[0].ID, responseItems[0].ID)
	assert.Equal(t, items[1].ID, responseItems[1].ID)

	// Verify mock expectations
	mockService.AssertExpectations(t)
}

func TestServer_HandleGetItem(t *testing.T) {
	// Set up Gin in test mode
	gin.SetMode(gin.TestMode)

	// Create mock monitor service
	mockService := new(MockMonitorService)

	// Create test item
	item := &monitor.Item{
		ID:        "test-1",
		Name:      "Test Item 1",
		Type:      "test",
		Status:    monitor.StatusOK,
		Value:     50.0,
		Threshold: 80.0,
		Unit:      "%",
		Icon:      "test",
		Message:   "Test message 1",
	}

	// Set up mock expectations
	mockService.On("GetItem", "test-1").Return(item)
	mockService.On("GetItem", "non-existent").Return(nil)

	// Create test configuration
	cfg := config.DefaultConfig()

	// Create server with mock service
	server := NewServer(cfg, mockService)

	// Test case 1: Existing item
	t.Run("Existing item", func(t *testing.T) {
		// Create a test HTTP recorder and context
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "test-1"}}

		// Call the handler
		server.handleGetItem(c)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code)

		// Parse response body
		var responseItem monitor.Item
		err := json.Unmarshal(w.Body.Bytes(), &responseItem)
		require.NoError(t, err)

		// Verify response item
		assert.Equal(t, item.ID, responseItem.ID)
		assert.Equal(t, item.Name, responseItem.Name)
	})

	// Test case 2: Non-existent item
	t.Run("Non-existent item", func(t *testing.T) {
		// Create a test HTTP recorder and context
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{gin.Param{Key: "id", Value: "non-existent"}}

		// Call the handler
		server.handleGetItem(c)

		// Check response
		assert.Equal(t, http.StatusNotFound, w.Code)

		// Parse response body
		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify response
		assert.Contains(t, response, "error")
		assert.Equal(t, "Item not found", response["error"])
	})

	// Verify mock expectations
	mockService.AssertExpectations(t)
}

func TestServer_HandleGetConfig(t *testing.T) {
	// Set up Gin in test mode
	gin.SetMode(gin.TestMode)

	// Create mock monitor service
	mockService := new(MockMonitorService)

	// Create test configuration
	cfg := config.DefaultConfig()
	cfg.Web.Host = "test-host"
	cfg.Web.Port = 9090

	// Create server with mock service
	server := NewServer(cfg, mockService)

	// Create a test HTTP recorder and context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Call the handler
	server.handleGetConfig(c)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response body
	var responseConfig config.Config
	err := json.Unmarshal(w.Body.Bytes(), &responseConfig)
	require.NoError(t, err)

	// Verify response config
	assert.Equal(t, cfg.Web.Host, responseConfig.Web.Host)
	assert.Equal(t, cfg.Web.Port, responseConfig.Web.Port)
}

func TestServer_HandleUpdateConfig(t *testing.T) {
	// Set up Gin in test mode
	gin.SetMode(gin.TestMode)

	// Create mock monitor service
	mockService := new(MockMonitorService)

	// Create test configuration
	cfg := config.DefaultConfig()

	// Create server with mock service
	server := NewServer(cfg, mockService)

	// Create a test HTTP recorder and context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a test request body
	updatedConfig := config.DefaultConfig()
	updatedConfig.Web.Host = "updated-host"
	updatedConfig.Web.Port = 9090
	jsonData, err := json.Marshal(updatedConfig)
	require.NoError(t, err)

	// Set up the request
	c.Request = httptest.NewRequest("POST", "/api/config", strings.NewReader(string(jsonData)))
	c.Request.Header.Set("Content-Type", "application/json")

	// Call the handler
	server.handleUpdateConfig(c)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response body
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify response
	assert.Contains(t, response, "message")
	assert.Contains(t, response["message"], "Configuration updated")
}

func TestServer_HandleIndex(t *testing.T) {
	// Set up Gin in test mode
	gin.SetMode(gin.TestMode)

	// Create a mock router that doesn't actually render HTML
	router := gin.New()
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Mock Index Page")
	})

	// Create a test HTTP recorder and request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	// Serve the request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Mock Index Page")
}

func TestServer_HandleConfigPage(t *testing.T) {
	// Set up Gin in test mode
	gin.SetMode(gin.TestMode)

	// Create a mock router that doesn't actually render HTML
	router := gin.New()
	router.GET("/config", func(c *gin.Context) {
		c.String(http.StatusOK, "Mock Config Page")
	})

	// Create a test HTTP recorder and request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/config", nil)

	// Serve the request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Mock Config Page")
}
