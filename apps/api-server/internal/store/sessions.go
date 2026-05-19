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
	q := s.Rebind(`SELECT admin_id FROM admin_sessions WHERE token_hash = ? AND expires_at > ?`)
	var id int64
	if err := s.DB.GetContext(ctx, &id, q, tokenHash, Now()); err != nil {
		return 0, err
	}
	return id, nil
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
	q := s.Rebind(`SELECT user_id FROM user_sessions WHERE token_hash = ? AND expires_at > ?`)
	var id int64
	if err := s.DB.GetContext(ctx, &id, q, tokenHash, Now()); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Store) DeleteUserSession(ctx context.Context, tokenHash string) error {
	q := s.Rebind(`DELETE FROM user_sessions WHERE token_hash = ?`)
	_, err := s.DB.ExecContext(ctx, q, tokenHash)
	return err
}
