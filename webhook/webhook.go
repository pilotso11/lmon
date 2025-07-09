package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Payload represents the JSON payload sent to a webhook endpoint.
type Payload struct {
	Text string `json:"text"`
}

// Send sends a message to the specified webhook URL.
// It marshals the message into JSON format and performs an HTTP POST request.
func Send(ctx context.Context, url string, msg string) error {
	var body Payload
	body.Text = msg
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("Webhook.Send (json): %v", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("Webhook.Send (newreq): %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Webhook.Send (req): %v", err)
	}

	// read the body to ensure the request is complete
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Webhook.Send (status): %v", resp.Status)
	}
	return nil
}
