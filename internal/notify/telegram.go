package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/soochol/upal/internal/upal"
)

// TelegramSender sends messages via the Telegram Bot API.
type TelegramSender struct {
	Client *http.Client
}

func (s *TelegramSender) Type() upal.ConnectionType { return upal.ConnTypeTelegram }

func (s *TelegramSender) Send(ctx context.Context, conn *upal.Connection, message string) error {
	chatID, _ := conn.Extras["chat_id"].(string)
	if chatID == "" {
		return fmt.Errorf("telegram connection %q missing chat_id in extras", conn.ID)
	}

	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", conn.Token)
	body, _ := json.Marshal(map[string]string{
		"chat_id":    chatID,
		"text":       message,
		"parse_mode": "Markdown",
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("telegram API returned %d", resp.StatusCode)
	}
	return nil
}
