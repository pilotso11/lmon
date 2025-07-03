package monitor

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockHTTPClientForWebhook is a mock implementation of HTTPClientInterface
type MockHTTPClientForWebhook struct {
	mock.Mock
}

// Do is a mock implementation of the Do method
func (m *MockHTTPClientForWebhook) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

func TestWebhookSender_Send(t *testing.T) {
	// Test cases
	tests := []struct {
		name        string
		url         string
		item        *Item
		setupMock   func(*MockHTTPClientForWebhook)
		expectError bool
	}{
		{
			name: "Successful webhook",
			url:  "https://webhook.example.com",
			item: &Item{
				ID:        "test-item",
				Name:      "Test Item",
				Type:      "test",
				Status:    StatusCritical,
				Value:     90.0,
				Threshold: 80.0,
				Unit:      "%",
				Icon:      "test",
				LastCheck: time.Now(),
				Message:   "Critical test message",
			},
			setupMock: func(m *MockHTTPClientForWebhook) {
				// Create a mock response
				resp := &http.Response{
					StatusCode: http.StatusOK,
					Body:       &MockResponseBody{Reader: nil}, // Empty body is fine
				}
				// Expect a POST request to the webhook URL with the correct payload
				m.On("Do", mock.MatchedBy(func(req *http.Request) bool {
					// Check method and URL
					if req.Method != "POST" || req.URL.String() != "https://webhook.example.com" {
						return false
					}
					
					// Check headers
					if req.Header.Get("Content-Type") != "application/json" {
						return false
					}
					
					// Check payload
					var payload WebhookPayload
					decoder := json.NewDecoder(req.Body)
					if err := decoder.Decode(&payload); err != nil {
						return false
					}
					
					// Verify payload fields
					return payload.ItemID == "test-item" &&
						payload.ItemName == "Test Item" &&
						payload.ItemType == "test" &&
						payload.Status == "CRITICAL" &&
						payload.Value == "90.00%" &&
						payload.Message == "Critical test message"
				})).Return(resp, nil)
			},
			expectError: false,
		},
		{
			name: "HTTP error",
			url:  "https://webhook.example.com",
			item: &Item{
				ID:        "test-item",
				Name:      "Test Item",
				Type:      "test",
				Status:    StatusCritical,
				Value:     90.0,
				Threshold: 80.0,
				Unit:      "%",
				Icon:      "test",
				LastCheck: time.Now(),
				Message:   "Critical test message",
			},
			setupMock: func(m *MockHTTPClientForWebhook) {
				// Expect a request that fails
				m.On("Do", mock.Anything).Return(nil, errors.New("connection refused"))
			},
			expectError: true,
		},
		{
			name: "Non-success status code",
			url:  "https://webhook.example.com",
			item: &Item{
				ID:        "test-item",
				Name:      "Test Item",
				Type:      "test",
				Status:    StatusCritical,
				Value:     90.0,
				Threshold: 80.0,
				Unit:      "%",
				Icon:      "test",
				LastCheck: time.Now(),
				Message:   "Critical test message",
			},
			setupMock: func(m *MockHTTPClientForWebhook) {
				// Create a mock response with non-success status code
				resp := &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       &MockResponseBody{Reader: nil}, // Empty body is fine
				}
				// Expect a request that returns a non-success status code
				m.On("Do", mock.Anything).Return(resp, nil)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock HTTP client
			mockClient := new(MockHTTPClientForWebhook)
			
			// Set up mock expectations
			tc.setupMock(mockClient)
			
			// Create webhook sender with mock client
			sender := &defaultWebhookSender{
				client: mockClient,
			}
			
			// Send webhook
			err := sender.Send(tc.url, tc.item)
			
			// Verify results
			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			
			// Verify all expectations were met
			mockClient.AssertExpectations(t)
		})
	}
}