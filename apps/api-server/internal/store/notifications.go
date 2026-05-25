package store

import "context"

type Notification struct {
	ID        int64  `db:"id" json:"id"`
	UserID    int64  `db:"user_id" json:"user_id"`
	Type      string `db:"type" json:"type"` // ticket_reply, payment_success, plan_expiring, plan_expired, system
	Title     string `db:"title" json:"title"`
	Content   string `db:"content" json:"content"`
	IsRead    int    `db:"is_read" json:"is_read"`
	Link      string `db:"link" json:"link"`
	CreatedAt string `db:"created_at" json:"created_at"`
}

func (s *Store) CreateNotification(ctx context.Context, userID int64, ntype, title, content, link string) error {
	q := s.Rebind(`INSERT INTO notifications(user_id, type, title, content, link) VALUES (?, ?, ?, ?, ?)`)
	_, err := s.DB.ExecContext(ctx, q, userID, ntype, title, content, link)
	return err
}

func (s *Store) ListNotifications(ctx context.Context, userID int64, limit int) ([]Notification, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	q := s.Rebind(`SELECT id, user_id, type, title, content, is_read, link, created_at
		FROM notifications WHERE user_id = ? ORDER BY id DESC LIMIT ?`)
	var rows []Notification
	if err := s.DB.SelectContext(ctx, &rows, q, userID, limit); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) CountUnreadNotifications(ctx context.Context, userID int64) (int, error) {
	q := s.Rebind(`SELECT COUNT(*) FROM notifications WHERE user_id = ? AND is_read = 0`)
	var n int
	if err := s.DB.GetContext(ctx, &n, q, userID); err != nil {
		return 0, err
	}
	return n, nil
}

func (s *Store) HasUnreadNotificationType(ctx context.Context, userID int64, ntype string) (bool, error) {
	q := s.Rebind(`SELECT COUNT(*) FROM notifications WHERE user_id = ? AND type = ? AND is_read = 0`)
	var n int
	if err := s.DB.GetContext(ctx, &n, q, userID, ntype); err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *Store) MarkNotificationRead(ctx context.Context, id, userID int64) error {
	q := s.Rebind(`UPDATE notifications SET is_read = 1 WHERE id = ? AND user_id = ?`)
	_, err := s.DB.ExecContext(ctx, q, id, userID)
	return err
}

func (s *Store) MarkAllNotificationsRead(ctx context.Context, userID int64) error {
	q := s.Rebind(`UPDATE notifications SET is_read = 1 WHERE user_id = ? AND is_read = 0`)
	_, err := s.DB.ExecContext(ctx, q, userID)
	return err
}

// NotifyUser is a convenience wrapper used by other services to send notifications.
func (s *Store) NotifyUser(ctx context.Context, userID int64, ntype, title, content, link string) {
	_ = s.CreateNotification(ctx, userID, ntype, title, content, link)
}
