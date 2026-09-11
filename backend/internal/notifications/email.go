package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
)

type EmailSender struct {
	apiKey  string
	from    string
	baseURL string
}

func NewEmailSender(apiKey, from string) *EmailSender {
	return &EmailSender{
		apiKey:  apiKey,
		from:    from,
		baseURL: "https://api.sendbyte.africa",
	}
}

func (e *EmailSender) Configured() bool {
	return e.apiKey != ""
}

func (e *EmailSender) Send(to, subject, htmlBody string) error {
	if !e.Configured() {
		return fmt.Errorf("email not configured: missing SENDBYTE_API_KEY")
	}

	payload := map[string]any{
		"from":    e.from,
		"to":      to,
		"subject": subject,
		"html":    htmlBody,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal email payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, e.baseURL+"/emails", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("create sendbyte request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("sendbyte request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("sendbyte returned status %d", resp.StatusCode)
	}

	return nil
}

var emailTemplate = template.Must(template.New("email").Parse(`
<!DOCTYPE html>
<html>
<body style="font-family:sans-serif;max-width:600px;margin:auto;padding:20px">
  <h2 style="color:#dc2626">Security Copilot Alert</h2>
  <h3>{{.Title}}</h3>
  <p>{{.Body}}</p>
  <div style="margin-top:24px">
    <a href="{{.ApproveURL}}" style="background:#16a34a;color:white;padding:12px 24px;text-decoration:none;border-radius:6px;margin-right:12px">
      Approve Fix
    </a>
    <a href="{{.DeclineURL}}" style="background:#dc2626;color:white;padding:12px 24px;text-decoration:none;border-radius:6px">
      Decline
    </a>
  </div>
  <p style="color:#6b7280;font-size:12px;margin-top:24px">
    These links expire in 24 hours. Only you can use them.
  </p>
</body>
</html>
`))

type emailData struct {
	Title      string
	Body       string
	ApproveURL string
	DeclineURL string
}

func BuildEmailHTML(title, body, approveURL, declineURL string) string {
	var buf bytes.Buffer
	if err := emailTemplate.Execute(&buf, emailData{
		Title:      title,
		Body:       body,
		ApproveURL: approveURL,
		DeclineURL: declineURL,
	}); err != nil {
		log.Printf("notifications: render email template: %v", err)
		return ""
	}
	return buf.String()
}
