package store

import (
	"context"
	"time"
)

type SubscriptionToken struct {
	ID                  int64      `db:"id" json:"id"`
	UserID              int64      `db:"user_id" json:"user_id"`
	Token               string     `db:"token" json:"token"`
	TokenHash           string     `db:"token_hash" json:"-"`
	Status              string     `db:"status" json:"status"`
	LastAccessIP        *string    `db:"last_access_ip" json:"last_access_ip"`
	LastAccessUserAgent *string    `db:"last_access_user_agent" json:"last_access_user_agent"`
	LastAccessAt        *time.Time `db:"last_access_at" json:"last_access_at"`
	CreatedAt           time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at" json:"updated_at"`
}

func (s *Store) FindActiveSubTokenByUser(ctx context.Context, userID int64) (*SubscriptionToken, error) {
	q := s.Rebind(`SELECT id, user_id, token, token_hash, status, last_access_ip,
		last_access_user_agent, last_access_at, created_at, updated_at
		FROM subscription_tokens WHERE user_id = ? AND status = 'active' ORDER BY id DESC LIMIT 1`)
	var t SubscriptionToken
	if err := s.DB.GetContext(ctx, &t, q, userID); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) FindActiveSubTokenByHash(ctx context.Context, hash string) (*SubscriptionToken, error) {
	q := s.Rebind(`SELECT id, user_id, token, token_hash, status, last_access_ip,
		last_access_user_agent, last_access_at, created_at, updated_at
		FROM subscription_tokens WHERE token_hash = ? AND status = 'active' LIMIT 1`)
	var t SubscriptionToken
	if err := s.DB.GetContext(ctx, &t, q, hash); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) CreateSubToken(ctx context.Context, userID int64, token, hash string) (int64, error) {
	return s.InsertReturningID(ctx,
		`INSERT INTO subscription_tokens(user_id, token, token_hash) VALUES (?, ?, ?)`,
		userID, token, hash,
	)
}

// RotateSubToken disables all active tokens for a user and inserts a new one.
func (s *Store) RotateSubToken(ctx context.Context, userID int64, token, hash string) error {
	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, s.Rebind(
		`UPDATE subscription_tokens SET status = 'revoked' WHERE user_id = ? AND status = 'active'`),
		userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, s.Rebind(
		`INSERT INTO subscription_tokens(user_id, token, token_hash) VALUES (?, ?, ?)`),
		userID, token, hash); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) TouchSubTokenAccess(ctx context.Context, id int64, ip, ua string) error {
	q := s.Rebind(`UPDATE subscription_tokens SET last_access_ip = ?, last_access_user_agent = ?,
		last_access_at = ? WHERE id = ?`)
	_, err := s.DB.ExecContext(ctx, q, ip, ua, Now(), id)
	return err
}

// LogSubAccess records a subscription access result row (allow/deny).
func (s *Store) LogSubAccess(ctx context.Context, userID *int64, tokenHash, target, ip, ua, result, reason string) error {
	q := s.Rebind(`INSERT INTO subscription_access_logs(user_id, token_hash, target, ip,
		user_agent, result, reason) VALUES (?, ?, ?, ?, ?, ?, ?)`)
	_, err := s.DB.ExecContext(ctx, q, userID, tokenHash, target, ip, ua, result, reason)
	return err
}
