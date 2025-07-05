package monitor

// WebhookPayload represents the payload sent to a webhook
type WebhookPayload struct {
	Timestamp string `json:"timestamp"`
	Status    string `json:"status"`
	ItemID    string `json:"item_id"`
	ItemName  string `json:"item_name"`
	ItemType  string `json:"item_type"`
	Value     string `json:"value"`
	Message   string `json:"message"`
}
