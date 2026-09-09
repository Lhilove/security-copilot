package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func sendTelegram(botToken, chatID, title, body, approveURL, declineURL string) error {
	text := fmt.Sprintf("*Security Copilot Alert*\n\n*%s*\n\n%s", title, body)

	payload := map[string]any{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
		"reply_markup": map[string]any{
			"inline_keyboard": [][]map[string]string{
				{
					{"text": "Approve Fix", "url": approveURL},
					{"text": "Decline", "url": declineURL},
				},
			},
		},
	}

	b, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("telegram post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram returned status %d", resp.StatusCode)
	}
	return nil
}
