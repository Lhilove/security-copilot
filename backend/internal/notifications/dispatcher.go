package notifications

import (
	"context"
	"fmt"
	"log"

	"github.com/lhilove/security-copilot/internal/database"
)

// Dispatcher fans out notifications to all enabled channels.
type Dispatcher struct {
	settingsRepo *database.SettingsRepository
	notifRepo    *database.NotificationRepository
	emailSender  *EmailSender
	jwtSecret    []byte
	baseURL      string // e.g. https://84.12.68.222 or your domain
}

func NewDispatcher(
	settingsRepo *database.SettingsRepository,
	notifRepo *database.NotificationRepository,
	jwtSecret []byte,
	baseURL string,
	sendByteAPIKey string,
	emailFrom string,
) *Dispatcher {
	return &Dispatcher{
		settingsRepo: settingsRepo,
		notifRepo:    notifRepo,
		emailSender:  NewEmailSender(sendByteAPIKey, emailFrom),
		jwtSecret:    jwtSecret,
		baseURL:      baseURL,
	}
}

// FindingAlert dispatches a notification for a new security finding.
func (d *Dispatcher) FindingAlert(ctx context.Context, userID, findingID, title, body string) {
	// Always store in-app notification
	if err := d.notifRepo.Create(ctx, userID, findingID, title, body); err != nil {
		log.Printf("notifications: store in-app: %v", err)
	}

	// Get user's channel settings
	settings, err := d.settingsRepo.GetByUserID(ctx, userID)
	if err != nil {
		// User hasn't configured notifications yet — in-app only
		return
	}

	// Build signed approval URLs
	approveURL, err := d.buildApprovalURL(userID, findingID, ActionApprove, "link")
	if err != nil {
		log.Printf("notifications: build approve url: %v", err)
		return
	}

	declineURL, err := d.buildApprovalURL(userID, findingID, ActionDecline, "link")
	if err != nil {
		log.Printf("notifications: build decline url: %v", err)
		return
	}

	// Dispatch to each enabled channel concurrently
	if settings.EmailEnabled && settings.Email != "" {
		go func() {
			subject := fmt.Sprintf("Security Copilot: %s", title)
			html := BuildEmailHTML(title, body, approveURL, declineURL)
			if html == "" {
				return
			}
			if err := d.emailSender.Send(settings.Email, subject, html); err != nil {
				log.Printf("notifications: email: %v", err)
			}
		}()
	}

	if settings.SlackEnabled && settings.SlackWebhookURL != "" {
		go func() {
			if err := sendSlack(settings.SlackWebhookURL, title, body, approveURL, declineURL); err != nil {
				log.Printf("notifications: slack: %v", err)
			}
		}()
	}

	if settings.DiscordEnabled && settings.DiscordWebhookURL != "" {
		go func() {
			if err := sendDiscord(settings.DiscordWebhookURL, title, body, approveURL, declineURL); err != nil {
				log.Printf("notifications: discord: %v", err)
			}
		}()
	}

	if settings.TelegramEnabled && settings.TelegramBotToken != "" && settings.TelegramChatID != "" {
		go func() {
			if err := sendTelegram(settings.TelegramBotToken, settings.TelegramChatID, title, body, approveURL, declineURL); err != nil {
				log.Printf("notifications: telegram: %v", err)
			}
		}()
	}
}

func (d *Dispatcher) buildApprovalURL(userID, findingID string, action ApprovalAction, channel string) (string, error) {
	token, err := IssueApprovalToken(userID, findingID, action, channel, d.jwtSecret)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/api/v1/approve?token=%s", d.baseURL, token), nil
}
