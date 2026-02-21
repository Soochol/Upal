package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/soochol/upal/internal/upal"
)

// SlackSender sends messages via a Slack incoming webhook URL.
type SlackSender struct {
	Client *http.Client
}

func (s *SlackSender) Type() upal.ConnectionType { return upal.ConnTypeSlack }

func (s *SlackSender) Send(ctx context.Context, conn *upal.Connection, message string) error {
	webhookURL, _ := conn.Extras["webhook_url"].(string)
	if webhookURL == "" {
		// Fall back to conn.Host as the webhook URL.
		webhookURL = conn.Host
	}
	if webhookURL == "" {
		return fmt.Errorf("slack connection %q missing webhook_url in extras", conn.ID)
	}

	channel, _ := conn.Extras["channel"].(string)

	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}

	payload := map[string]string{"text": message}
	if channel != "" {
		payload["channel"] = channel
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("slack send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack API returned %d", resp.StatusCode)
	}
	return nil
}
