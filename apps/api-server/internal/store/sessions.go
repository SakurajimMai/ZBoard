package store

import (
	"context"
	"time"
)

func (s *Store) CreateAdminSession(ctx context.Context, adminID int64, tokenHash string, expires time.Time) error {
	q := s.Rebind(`INSERT INTO admin_sessions(admin_id, token_hash, expires_at) VALUES (?, ?, ?)`)
	_, err := s.DB.ExecContext(ctx, q, adminID, tokenHash, expires)
	return err
}

// FindAdminSession returns the admin id for an active session token hash.
func (s *Store) FindAdminSession(ctx context.Context, tokenHash string) (int64, error) {
	id, _, err := s.FindAdminSessionWithExpiry(ctx, tokenHash)
	return id, err
}

// FindAdminSessionWithExpiry returns the admin id and the session's current
// expires_at so callers can decide whether a refresh write is needed.
func (s *Store) FindAdminSessionWithExpiry(ctx context.Context, tokenHash string) (int64, time.Time, error) {
	q := s.Rebind(`SELECT admin_id, expires_at FROM admin_sessions WHERE token_hash = ? AND expires_at > ?`)
	var row struct {
		ID        int64     `db:"admin_id"`
		ExpiresAt time.Time `db:"expires_at"`
	}
	if err := s.DB.GetContext(ctx, &row, q, tokenHash, Now()); err != nil {
		return 0, time.Time{}, err
	}
	return row.ID, row.ExpiresAt, nil
}

func (s *Store) RefreshAdminSession(ctx context.Context, tokenHash string, expires time.Time) error {
	q := s.Rebind(`UPDATE admin_sessions SET expires_at = ? WHERE token_hash = ? AND expires_at > ?`)
	_, err := s.DB.ExecContext(ctx, q, expires, tokenHash, Now())
	return err
}

func (s *Store) DeleteAdminSession(ctx context.Context, tokenHash string) error {
	q := s.Rebind(`DELETE FROM admin_sessions WHERE token_hash = ?`)
	_, err := s.DB.ExecContext(ctx, q, tokenHash)
	return err
}

func (s *Store) CreateUserSession(ctx context.Context, userID int64, tokenHash string, expires time.Time) error {
	q := s.Rebind(`INSERT INTO user_sessions(user_id, token_hash, expires_at) VALUES (?, ?, ?)`)
	_, err := s.DB.ExecContext(ctx, q, userID, tokenHash, expires)
	return err
}

func (s *Store) FindUserSession(ctx context.Context, tokenHash string) (int64, error) {
	id, _, err := s.FindUserSessionWithExpiry(ctx, tokenHash)
	return id, err
}

// FindUserSessionWithExpiry returns the user id and the session's current
// expires_at so callers can decide whether a refresh write is needed.
func (s *Store) FindUserSessionWithExpiry(ctx context.Context, tokenHash string) (int64, time.Time, error) {
	q := s.Rebind(`SELECT user_id, expires_at FROM user_sessions WHERE token_hash = ? AND expires_at > ?`)
	var row struct {
		ID        int64     `db:"user_id"`
		ExpiresAt time.Time `db:"expires_at"`
	}
	if err := s.DB.GetContext(ctx, &row, q, tokenHash, Now()); err != nil {
		return 0, time.Time{}, err
	}
	return row.ID, row.ExpiresAt, nil
}

func (s *Store) RefreshUserSession(ctx context.Context, tokenHash string, expires time.Time) error {
	q := s.Rebind(`UPDATE user_sessions SET expires_at = ? WHERE token_hash = ? AND expires_at > ?`)
	_, err := s.DB.ExecContext(ctx, q, expires, tokenHash, Now())
	return err
}

func (s *Store) DeleteUserSession(ctx context.Context, tokenHash string) error {
	q := s.Rebind(`DELETE FROM user_sessions WHERE token_hash = ?`)
	_, err := s.DB.ExecContext(ctx, q, tokenHash)
	return err
}
