package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Notification is an in-app notification record.
type Notification struct {
	ID        string
	UserID    string
	FindingID *string
	Title     string
	Body      string
	Read      bool
	CreatedAt string
}

// NotificationRepository handles in-app notification persistence.
type NotificationRepository struct {
	pool *pgxpool.Pool
}

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{pool: pool}
}

func (r *NotificationRepository) Create(ctx context.Context, userID, findingID, title, body string) error {
	var fid *string
	if findingID != "" {
		fid = &findingID
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO notifications (user_id, finding_id, title, body) VALUES ($1,$2,$3,$4)`,
		userID, fid, title, body,
	)
	return err
}

func (r *NotificationRepository) ListUnread(ctx context.Context, userID string) ([]Notification, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, finding_id, title, body, read, created_at
		 FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	var out []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.FindingID, &n.Title, &n.Body, &n.Read, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

func (r *NotificationRepository) MarkRead(ctx context.Context, notificationID, userID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET read = TRUE WHERE id = $1 AND user_id = $2`,
		notificationID, userID,
	)
	return err
}

func (r *NotificationRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read = FALSE`,
		userID,
	).Scan(&count)
	return count, err
}
