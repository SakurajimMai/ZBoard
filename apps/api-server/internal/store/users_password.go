package store

import "context"

// UpdateUserPasswordHash sets the bcrypt hash for a user, used by password reset.
func (s *Store) UpdateUserPasswordHash(ctx context.Context, userID int64, hash string) error {
	q := s.Rebind(`UPDATE users SET password_hash = ? WHERE id = ?`)
	_, err := s.DB.ExecContext(ctx, q, hash, userID)
	return err
}
