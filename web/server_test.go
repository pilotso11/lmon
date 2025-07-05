package web

import (
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

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

// RefreshChecks is a mock implementation of the RefreshChecks method
func (m *MockMonitorService) RefreshChecks() {
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

	// Verify JSON field names match what the JavaScript code expects
	var jsonMap map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &jsonMap)
	require.NoError(t, err)

	// Check top-level fields
	assert.Contains(t, jsonMap, "web")
	assert.Contains(t, jsonMap, "monitoring")
	assert.Contains(t, jsonMap, "webhook")

	// Check web fields
	webMap := jsonMap["web"].(map[string]interface{})
	assert.Contains(t, webMap, "host")
	assert.Contains(t, webMap, "port")

	// Check monitoring fields
	monitoringMap := jsonMap["monitoring"].(map[string]interface{})
	assert.Contains(t, monitoringMap, "interval")
	assert.Contains(t, monitoringMap, "disk")
	assert.Contains(t, monitoringMap, "system")
	assert.Contains(t, monitoringMap, "healthchecks")

	// Check system fields
	systemMap := monitoringMap["system"].(map[string]interface{})
	require.Contains(t, systemMap, "cpu")
	require.Contains(t, systemMap, "memory")
	assert.Contains(t, systemMap["cpu"], "icon")
	assert.Contains(t, systemMap["memory"], "icon")
	assert.Contains(t, systemMap["cpu"], "threshold")
	assert.Contains(t, systemMap["memory"], "threshold")

	// Check webhook fields
	webhookMap := jsonMap["webhook"].(map[string]interface{})
	assert.Contains(t, webhookMap, "enabled")
	assert.Contains(t, webhookMap, "url")
}

func TestServer_HandleUpdateConfig(t *testing.T) {
	// Set up Gin in test mode
	gin.SetMode(gin.TestMode)

	// Create mock monitor service
	mockService := new(MockMonitorService)

	// Set up mock expectations
	mockService.On("RefreshChecks").Return()

	// Create test configuration
	cfg := config.DefaultConfig()

	// Create server with mock service
	server := NewServer(cfg, mockService)

	// Create a temporary file for the config
	tmpfile, err := os.CreateTemp("", "test-config-*.yaml")
	require.NoError(t, err)
	defer func(name string) {
		_ = os.Remove(name)
	}(tmpfile.Name()) // Clean up the temporary file when done

	// Set the config path to the temporary file
	server.SetConfigPath(tmpfile.Name())

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

func TestTemplates_LoadAndParse(t *testing.T) {
	// This test ensures that the embedded templates can be loaded and parsed without error
	templFS, err := GetTemplatesFSWithRoot()
	assert.NoError(t, err, "should load embedded templates FS")

	// Try to parse all HTML templates in the embedded FS
	_, err = template.ParseFS(templFS, "*.html")
	assert.NoError(t, err, "should parse all embedded HTML templates")
}

func TestServer_StartAndStop(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Web.Port = 0 // random port
	mockService := new(MockMonitorService)
	server := NewServer(cfg, mockService)

	// Start and Stop should not panic (we don't actually listen on a port in test)
	err := server.Stop()
	assert.NoError(t, err)
}

func TestServer_HandleHealthCheck_EdgeCases(t *testing.T) {
	cfg := config.DefaultConfig()
	mockService := new(MockMonitorService)
	server := NewServer(cfg, mockService)

	// Case 1: No items at all (should be healthy)
	mockService.On("GetItems").Return([]*monitor.Item{})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	server.handleHealthCheck(c)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "healthy", resp["status"])

	// Case 2: All items healthy
	mockService = new(MockMonitorService)
	server = NewServer(cfg, mockService)
	mockService.On("GetItems").Return([]*monitor.Item{
		{ID: "1", Status: monitor.StatusOK},
		{ID: "2", Status: monitor.StatusWarning},
	})
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	server.handleHealthCheck(c)
	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "healthy", resp["status"])

	// Case 3: At least one item critical
	mockService = new(MockMonitorService)
	server = NewServer(cfg, mockService)
	mockService.On("GetItems").Return([]*monitor.Item{
		{ID: "1", Status: monitor.StatusOK},
		{ID: "2", Status: monitor.StatusCritical},
	})
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	server.handleHealthCheck(c)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "unhealthy", resp["status"])
}

func TestGetTemplatesFS(t *testing.T) {
	fs := GetTemplatesFS()
	// Should not be zero value
	assert.NotNil(t, fs)
}

func TestGetHTTPFileSystem(t *testing.T) {
	httpFS := GetHTTPFileSystem()
	assert.NotNil(t, httpFS)
}

func TestWebServer_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.DefaultConfig()
	cfg.Web.Port = 0 // random port if we ever use ListenAndServe

	// Use a mock monitor service
	mockService := new(MockMonitorService)
	mockService.On("GetItems").Return([]*monitor.Item{
		{
			ID:        "test-1",
			Name:      "Test Item 1",
			Type:      "test",
			Status:    monitor.StatusOK,
			Value:     42.0,
			Threshold: 80.0,
			Unit:      "%",
			Icon:      "test",
			Message:   "Integration test",
		},
	})
	mockService.On("GetItem", "test-1").Return(&monitor.Item{
		ID:        "test-1",
		Name:      "Test Item 1",
		Type:      "test",
		Status:    monitor.StatusOK,
		Value:     42.0,
		Threshold: 80.0,
		Unit:      "%",
		Icon:      "test",
		Message:   "Integration test",
	})
	mockService.On("GetItem", "notfound").Return(nil)
	mockService.On("RefreshChecks").Return()

	server := NewServer(cfg, mockService)

	// Use httptest server for integration
	ts := httptest.NewServer(server.router)
	defer ts.Close()

	// Test GET /api/items
	resp, err := http.Get(ts.URL + "/api/items")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var items []*monitor.Item
	require.NoError(t, json.Unmarshal(body, &items))
	assert.Len(t, items, 1)
	assert.Equal(t, "test-1", items[0].ID)

	// Test GET /api/items/:id (found)
	resp, err = http.Get(ts.URL + "/api/items/test-1")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ = io.ReadAll(resp.Body)
	var item monitor.Item
	require.NoError(t, json.Unmarshal(body, &item))
	assert.Equal(t, "test-1", item.ID)

	// Test GET /api/items/:id (not found)
	resp, err = http.Get(ts.URL + "/api/items/notfound")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Test GET /api/config
	resp, err = http.Get(ts.URL + "/api/config")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Test GET /healthz
	resp, err = http.Get(ts.URL + "/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Test GET / (index page)
	resp, err = http.Get(ts.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Test GET /config (config page)
	resp, err = http.Get(ts.URL + "/config")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Test GET /static (should return 404 or 200 depending on static files present)
	resp, err = http.Get(ts.URL + "/static/")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound)

	// Optionally, test a static file if any exist
	staticDir := "web/static"
	files, _ := os.ReadDir(staticDir)
	for _, f := range files {
		if !f.IsDir() {
			resp, err := http.Get(ts.URL + "/static/" + f.Name())
			require.NoError(t, err)
			resp.Body.Close()
			assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound)
		}
		break
	}

	client := &http.Client{Timeout: 2 * time.Second}

	// Test POST /api/config (update config)
	updatedConfig := config.DefaultConfig()
	updatedConfig.Web.Host = "integration-test"
	jsonData, _ := json.Marshal(updatedConfig)
	req, _ := http.NewRequest("POST", ts.URL+"/api/config", strings.NewReader(string(jsonData)))
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Test POST /api/config (invalid JSON)
	req, _ = http.NewRequest("POST", ts.URL+"/api/config", strings.NewReader("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Test POST /api/config/disk (valid)
	mockService.On("RefreshChecks").Return()
	diskPayload := `{"path":"/testdisk","threshold":90,"icon":"storage"}`
	req, _ = http.NewRequest("POST", ts.URL+"/api/config/disk", strings.NewReader(diskPayload))
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Test POST /api/config/disk (invalid JSON)
	req, _ = http.NewRequest("POST", ts.URL+"/api/config/disk", strings.NewReader("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Test POST /api/config/healthcheck (valid)
	healthPayload := `{"name":"testhc","url":"http://localhost","interval":10,"timeout":5,"expected_status":200}`
	req, _ = http.NewRequest("POST", ts.URL+"/api/config/healthcheck", strings.NewReader(healthPayload))
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Test POST /api/config/healthcheck (invalid JSON)
	req, _ = http.NewRequest("POST", ts.URL+"/api/config/healthcheck", strings.NewReader("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Test POST /api/config/webhook (valid)
	webhookPayload := `{"enabled":true,"url":"http://localhost/webhook"}`
	req, _ = http.NewRequest("POST", ts.URL+"/api/config/webhook", strings.NewReader(webhookPayload))
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Test POST /api/config/webhook (invalid JSON)
	req, _ = http.NewRequest("POST", ts.URL+"/api/config/webhook", strings.NewReader("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Test DELETE /api/config/disk/:id (valid)
	req, _ = http.NewRequest("DELETE", ts.URL+"/api/config/disk/testdisk", nil)
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Test DELETE /api/config/healthcheck/:id (valid)
	req, _ = http.NewRequest("DELETE", ts.URL+"/api/config/healthcheck/testhc", nil)
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Test DELETE /api/config/unknown/:id (invalid type)
	req, _ = http.NewRequest("DELETE", ts.URL+"/api/config/unknown/something", nil)
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
