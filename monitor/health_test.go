package monitor

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"lmon/config"
)

// MockHTTPClient is a mock implementation of HTTPClient
type MockHTTPClient struct {
	mock.Mock
}

// Do is a mock implementation of the Do method
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

// MockResponseBody is a mock implementation of io.ReadCloser for response bodies
type MockResponseBody struct {
	io.Reader
}

// Close is a mock implementation of the Close method
func (m *MockResponseBody) Close() error {
	return nil
}

func TestHealthMonitor_Check(t *testing.T) {
	// Test cases
	tests := []struct {
		name           string
		healthConfigs  []config.HealthcheckConfig
		setupMock      func(*MockHTTPClient)
		expectedItems  int
		expectedStatus map[string]Status
		expectError    bool
	}{
		{
			name: "Single health check with OK status",
			healthConfigs: []config.HealthcheckConfig{
				{
					Name:     "Test API",
					URL:      "https://api.example.com/health",
					Interval: 60,
					Timeout:  10,
					Icon:     "cloud",
				},
			},
			setupMock: func(m *MockHTTPClient) {
				// Create a mock response
				resp := &http.Response{
					StatusCode: http.StatusOK,
					Body:       &MockResponseBody{Reader: strings.NewReader(`{"status":"ok"}`)},
				}
				// Expect a request to the health check URL
				m.On("Do", mock.MatchedBy(func(req *http.Request) bool {
					return req.URL.String() == "https://api.example.com/health"
				})).Return(resp, nil)
			},
			expectedItems: 1,
			expectedStatus: map[string]Status{
				"health-Test API": StatusOK,
			},
			expectError: false,
		},
		{
			name: "Single health check with non-OK status code",
			healthConfigs: []config.HealthcheckConfig{
				{
					Name:     "Test API",
					URL:      "https://api.example.com/health",
					Interval: 60,
					Timeout:  10,
					Icon:     "cloud",
				},
			},
			setupMock: func(m *MockHTTPClient) {
				// Create a mock response
				resp := &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       &MockResponseBody{Reader: strings.NewReader(`{"status":"error"}`)},
				}
				// Expect a request to the health check URL
				m.On("Do", mock.MatchedBy(func(req *http.Request) bool {
					return req.URL.String() == "https://api.example.com/health"
				})).Return(resp, nil)
			},
			expectedItems: 1,
			expectedStatus: map[string]Status{
				"health-Test API": StatusCritical,
			},
			expectError: false,
		},
		{
			name: "Single health check with non-OK status in response",
			healthConfigs: []config.HealthcheckConfig{
				{
					Name:     "Test API",
					URL:      "https://api.example.com/health",
					Interval: 60,
					Timeout:  10,
					Icon:     "cloud",
				},
			},
			setupMock: func(m *MockHTTPClient) {
				// Create a mock response
				resp := &http.Response{
					StatusCode: http.StatusOK,
					Body:       &MockResponseBody{Reader: strings.NewReader(`{"status":"error"}`)},
				}
				// Expect a request to the health check URL
				m.On("Do", mock.MatchedBy(func(req *http.Request) bool {
					return req.URL.String() == "https://api.example.com/health"
				})).Return(resp, nil)
			},
			expectedItems: 1,
			expectedStatus: map[string]Status{
				"health-Test API": StatusCritical,
			},
			expectError: false,
		},
		{
			name: "Single health check with connection error",
			healthConfigs: []config.HealthcheckConfig{
				{
					Name:     "Test API",
					URL:      "https://api.example.com/health",
					Interval: 60,
					Timeout:  10,
					Icon:     "cloud",
				},
			},
			setupMock: func(m *MockHTTPClient) {
				// Expect a request to the health check URL that fails
				m.On("Do", mock.MatchedBy(func(req *http.Request) bool {
					return req.URL.String() == "https://api.example.com/health"
				})).Return(nil, errors.New("connection refused"))
			},
			expectedItems: 1,
			expectedStatus: map[string]Status{
				"health-Test API": StatusCritical,
			},
			expectError: false,
		},
		{
			name: "Multiple health checks with different statuses",
			healthConfigs: []config.HealthcheckConfig{
				{
					Name:     "API 1",
					URL:      "https://api1.example.com/health",
					Interval: 60,
					Timeout:  10,
					Icon:     "cloud",
				},
				{
					Name:     "API 2",
					URL:      "https://api2.example.com/health",
					Interval: 60,
					Timeout:  10,
					Icon:     "cloud",
				},
			},
			setupMock: func(m *MockHTTPClient) {
				// Create mock responses
				resp1 := &http.Response{
					StatusCode: http.StatusOK,
					Body:       &MockResponseBody{Reader: strings.NewReader(`{"status":"ok"}`)},
				}
				resp2 := &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       &MockResponseBody{Reader: strings.NewReader(`{"status":"error"}`)},
				}
				// Expect requests to the health check URLs
				m.On("Do", mock.MatchedBy(func(req *http.Request) bool {
					return req.URL.String() == "https://api1.example.com/health"
				})).Return(resp1, nil)
				m.On("Do", mock.MatchedBy(func(req *http.Request) bool {
					return req.URL.String() == "https://api2.example.com/health"
				})).Return(resp2, nil)
			},
			expectedItems: 2,
			expectedStatus: map[string]Status{
				"health-API 1": StatusOK,
				"health-API 2": StatusCritical,
			},
			expectError: false,
		},
		{
			name: "Invalid URL",
			healthConfigs: []config.HealthcheckConfig{
				{
					Name:     "Invalid URL",
					URL:      "://invalid-url",
					Interval: 60,
					Timeout:  10,
					Icon:     "cloud",
				},
			},
			setupMock: func(m *MockHTTPClient) {
				// No mock expectations needed for invalid URL
			},
			expectedItems: 1,
			expectedStatus: map[string]Status{
				"health-Invalid URL": StatusUnknown,
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock HTTP client
			mockClient := new(MockHTTPClient)

			// Set up mock expectations
			tc.setupMock(mockClient)

			// Create test configuration
			cfg := config.DefaultConfig()
			cfg.Monitoring.Healthchecks = tc.healthConfigs

			// Create health monitor with mock client
			monitor := NewHealthMonitorWithClient(cfg, mockClient)

			// Check health endpoints
			items, err := monitor.Check()

			// Verify results
			if tc.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedItems, len(items))

			// Check status of items
			for _, item := range items {
				if expectedStatus, ok := tc.expectedStatus[item.ID]; ok {
					assert.Equal(t, expectedStatus, item.Status, "Item %s has wrong status", item.ID)
				}

				// Check common properties
				assert.NotEmpty(t, item.Name)
				assert.NotEmpty(t, item.Message)
				assert.NotEmpty(t, item.Icon)
				assert.NotZero(t, item.LastCheck)
			}

			// Verify all expectations were met
			mockClient.AssertExpectations(t)
		})
	}
}
