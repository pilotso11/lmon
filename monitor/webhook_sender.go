package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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