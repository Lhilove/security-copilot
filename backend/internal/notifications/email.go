package notifications

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
	"os"
	"strings"
)

// EmailSender sends notification emails via SMTP.
type EmailSender struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewEmailSender() *EmailSender {
	return &EmailSender{
		host:     os.Getenv("SMTP_HOST"),
		port:     os.Getenv("SMTP_PORT"),
		username: os.Getenv("SMTP_USERNAME"),
		password: os.Getenv("SMTP_PASSWORD"),
		from:     os.Getenv("SMTP_FROM"),
	}
}

func (e *EmailSender) Configured() bool {
	return e.host != "" && e.username != "" && e.password != ""
}

func (e *EmailSender) Send(to, subject, htmlBody string) error {
	if !e.Configured() {
		return fmt.Errorf("email not configured")
	}
	port := e.port
	if port == "" {
		port = "587"
	}
	auth := smtp.PlainAuth("", e.username, e.password, e.host)
	msg := strings.Join([]string{
		"From: " + e.from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
		"",
		htmlBody,
	}, "\r\n")
	return smtp.SendMail(e.host+":"+port, auth, e.from, []string{to}, []byte(msg))
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
