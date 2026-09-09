package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Settings holds a user's notification channel configuration.
type Settings struct {
	ID                string
	UserID            string
	Email             string
	SlackWebhookURL   string
	DiscordWebhookURL string
	TelegramBotToken  string
	TelegramChatID    string
	EmailEnabled      bool
	SlackEnabled      bool
	DiscordEnabled    bool
	TelegramEnabled   bool
}

// SettingsRepository handles notification settings persistence.
type SettingsRepository struct {
	pool *pgxpool.Pool
}

func NewSettingsRepository(pool *pgxpool.Pool) *SettingsRepository {
	return &SettingsRepository{pool: pool}
}

func (r *SettingsRepository) Upsert(ctx context.Context, s *Settings) error {
	query := `
		INSERT INTO notification_settings (
			user_id, email, slack_webhook_url, discord_webhook_url,
			telegram_bot_token, telegram_chat_id,
			email_enabled, slack_enabled, discord_enabled, telegram_enabled
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (user_id) DO UPDATE SET
			email               = EXCLUDED.email,
			slack_webhook_url   = EXCLUDED.slack_webhook_url,
			discord_webhook_url = EXCLUDED.discord_webhook_url,
			telegram_bot_token  = EXCLUDED.telegram_bot_token,
			telegram_chat_id    = EXCLUDED.telegram_chat_id,
			email_enabled       = EXCLUDED.email_enabled,
			slack_enabled       = EXCLUDED.slack_enabled,
			discord_enabled     = EXCLUDED.discord_enabled,
			telegram_enabled    = EXCLUDED.telegram_enabled,
			updated_at          = NOW()
	`
	_, err := r.pool.Exec(ctx, query,
		s.UserID, s.Email, s.SlackWebhookURL, s.DiscordWebhookURL,
		s.TelegramBotToken, s.TelegramChatID,
		s.EmailEnabled, s.SlackEnabled, s.DiscordEnabled, s.TelegramEnabled,
	)
	return err
}

func (r *SettingsRepository) GetByUserID(ctx context.Context, userID string) (*Settings, error) {
	query := `
		SELECT id, user_id, email, slack_webhook_url, discord_webhook_url,
			telegram_bot_token, telegram_chat_id,
			email_enabled, slack_enabled, discord_enabled, telegram_enabled
		FROM notification_settings
		WHERE user_id = $1
	`
	s := &Settings{}
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&s.ID, &s.UserID, &s.Email, &s.SlackWebhookURL, &s.DiscordWebhookURL,
		&s.TelegramBotToken, &s.TelegramChatID,
		&s.EmailEnabled, &s.SlackEnabled, &s.DiscordEnabled, &s.TelegramEnabled,
	)
	if err != nil {
		return nil, fmt.Errorf("get notification settings: %w", err)
	}
	return s, nil
}
