package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func sendDiscord(webhookURL, title, body, approveURL, declineURL string) error {
	payload := map[string]any{
		"embeds": []map[string]any{
			{
				"title":       "Security Copilot Alert",
				"description": fmt.Sprintf("**%s**\n\n%s", title, body),
				"color":       15158332, // red
				"fields": []map[string]string{
					{
						"name":  "Approve Fix",
						"value": fmt.Sprintf("[Click to approve](%s)", approveURL),
					},
					{
						"name":  "Decline",
						"value": fmt.Sprintf("[Click to decline](%s)", declineURL),
					},
				},
			},
		},
	}

	b, _ := json.Marshal(payload)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("discord post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != 204 {
		return fmt.Errorf("discord returned status %d", resp.StatusCode)
	}
	return nil
}
