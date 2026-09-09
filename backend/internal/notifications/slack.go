package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func sendSlack(webhookURL, title, body, approveURL, declineURL string) error {
	payload := map[string]any{
		"blocks": []map[string]any{
			{
				"type": "header",
				"text": map[string]string{
					"type": "plain_text",
					"text": "Security Copilot Alert",
				},
			},
			{
				"type": "section",
				"text": map[string]string{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*%s*\n%s", title, body),
				},
			},
			{
				"type": "actions",
				"elements": []map[string]any{
					{
						"type": "button",
						"text": map[string]string{
							"type": "plain_text",
							"text": "Approve Fix",
						},
						"style": "primary",
						"url":   approveURL,
					},
					{
						"type": "button",
						"text": map[string]string{
							"type": "plain_text",
							"text": "Decline",
						},
						"style": "danger",
						"url":   declineURL,
					},
				},
			},
		},
	}

	b, _ := json.Marshal(payload)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("slack post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}
	return nil
}
