package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WebhookConfig holds optional notification webhook settings.
type WebhookConfig struct {
	URL     string // Slack/Discord webhook URL or generic endpoint
	Enabled bool
}

// Webhook is the global webhook configuration.
var Webhook WebhookConfig

// SendWebhook posts a notification to the configured webhook.
// Slack format: {"text": "title: body"} — works for both Slack and Discord webhooks.
func SendWebhook(title, body string) error {
	if !Webhook.Enabled || Webhook.URL == "" {
		return nil
	}

	payload := map[string]string{
		"text": fmt.Sprintf("*%s*: %s", title, body),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(Webhook.URL, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// Notify sends both desktop and webhook notifications.
func Notify(title, body string) {
	_ = Send(title, body)
	_ = SendWebhook(title, body)
}
