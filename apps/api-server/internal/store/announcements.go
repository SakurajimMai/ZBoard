package store

import (
	"context"
	"strings"
	"time"
)

type Announcement struct {
	ID        int64      `db:"id" json:"id"`
	Title     string     `db:"title" json:"title"`
	Content   string     `db:"content" json:"content"`
	Popup     bool       `db:"popup" json:"popup"`
	Priority  int        `db:"priority" json:"priority"`
	Status    string     `db:"status" json:"status"`
	StartsAt  *time.Time `db:"starts_at" json:"starts_at"`
	EndsAt    *time.Time `db:"ends_at" json:"ends_at"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
}

type AnnouncementInput struct {
	Title    string
	Content  string
	Popup    bool
	Priority int
	Status   string
	StartsAt *time.Time
	EndsAt   *time.Time
}

func (s *Store) CreateAnnouncement(ctx context.Context, in AnnouncementInput) (int64, error) {
	if strings.TrimSpace(in.Status) == "" {
		in.Status = "active"
	}
	return s.InsertReturningID(ctx,
		`INSERT INTO announcements(title, content, popup, priority, status, starts_at, ends_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		in.Title, in.Content, in.Popup, in.Priority, in.Status, in.StartsAt, in.EndsAt,
	)
}

func (s *Store) UpdateAnnouncement(ctx context.Context, id int64, in AnnouncementInput) error {
	if strings.TrimSpace(in.Status) == "" {
		in.Status = "active"
	}
	q := s.Rebind(`UPDATE announcements SET title = ?, content = ?, popup = ?, priority = ?,
		status = ?, starts_at = ?, ends_at = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`)
	_, err := s.DB.ExecContext(ctx, q, in.Title, in.Content, in.Popup, in.Priority, in.Status, in.StartsAt, in.EndsAt, id)
	return err
}

func (s *Store) DeleteAnnouncement(ctx context.Context, id int64) error {
	q := s.Rebind(`DELETE FROM announcements WHERE id = ?`)
	_, err := s.DB.ExecContext(ctx, q, id)
	return err
}

func (s *Store) ListAnnouncementsPage(ctx context.Context, p PageParams) ([]Announcement, int64, error) {
	p = NormalizePage(p)
	var total int64
	if err := s.DB.GetContext(ctx, &total, `SELECT COUNT(*) FROM announcements`); err != nil {
		return nil, 0, err
	}
	q := s.Rebind(`SELECT id, title, content, popup, priority, status, starts_at, ends_at, created_at, updated_at
		FROM announcements ORDER BY priority DESC, id DESC LIMIT ? OFFSET ?`)
	var rows []Announcement
	if err := s.DB.SelectContext(ctx, &rows, q, p.PageSize, p.Offset()); err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (s *Store) ListActiveAnnouncements(ctx context.Context, limit int) ([]Announcement, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	now := Now()
	q := s.Rebind(`SELECT id, title, content, popup, priority, status, starts_at, ends_at, created_at, updated_at
		FROM announcements
		WHERE status = 'active'
		  AND (starts_at IS NULL OR starts_at <= ?)
		  AND (ends_at IS NULL OR ends_at >= ?)
		ORDER BY priority DESC, id DESC LIMIT ?`)
	var rows []Announcement
	if err := s.DB.SelectContext(ctx, &rows, q, now, now, limit); err != nil {
		return nil, err
	}
	return rows, nil
}
