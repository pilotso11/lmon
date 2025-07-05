package monitor

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

// defaultWebhookSender is the default implementation of WebhookSenderInterface
type defaultWebhookSender struct {
	client HTTPClientInterface
}

// newDefaultWebhookSender creates a new default webhook sender
func newDefaultWebhookSender() *defaultWebhookSender {
	return &defaultWebhookSender{
		client: &http.Client{
			Timeout: time.Second * 10,
		},
	}
}

// Send sends a notification to a webhook URL
func (s *defaultWebhookSender) Send(url string, item *Item) error {
	// Create webhook payload
	payload := WebhookPayload{
		Timestamp: time.Now().Format(time.RFC3339),
		Status:    string(item.Status),
		ItemID:    item.ID,
		ItemName:  item.Name,
		ItemType:  item.Type,
		Value:     fmt.Sprintf("%.2f%s", item.Value, item.Unit),
		Message:   item.Message,
	}

	// Marshal payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "lmon-monitoring-service")

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-success status code: %d", resp.StatusCode)
	}

	return nil
}

// --- TESTS ---

type mockHTTPClient struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}

func TestDefaultWebhookSender_Send_Errors(t *testing.T) {
	item := &Item{
		ID:        "id",
		Name:      "name",
		Type:      "type",
		Status:    StatusOK,
		Value:     1.23,
		Threshold: 10,
		Unit:      "%",
		Icon:      "icon",
		LastCheck: time.Now(),
		Message:   "msg",
	}

	// Request creation error (invalid URL)
	s := &defaultWebhookSender{client: &mockHTTPClient{}}
	err := s.Send(":", item)
	if err == nil {
		t.Error("expected request creation error, got nil")
	}

	// Network error
	s = &defaultWebhookSender{
		client: &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("network error")
			},
		},
	}
	err = s.Send("http://example.com", item)
	if err == nil || err.Error() != "failed to send webhook request: network error" {
		t.Errorf("expected network error, got %v", err)
	}

	// Non-2xx status code
	s = &defaultWebhookSender{
		client: &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 500,
					Body:       io.NopCloser(bytes.NewBufferString("fail")),
				}, nil
			},
		},
	}
	err = s.Send("http://example.com", item)
	if err == nil || err.Error() != "webhook returned non-success status code: 500" {
		t.Errorf("expected non-2xx error, got %v", err)
	}

	// Success
	s = &defaultWebhookSender{
		client: &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString("ok")),
				}, nil
			},
		},
	}
	err = s.Send("http://example.com", item)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}
