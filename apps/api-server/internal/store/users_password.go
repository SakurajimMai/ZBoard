package store

import "context"

// UpdateUserPasswordHash sets the bcrypt hash and revokes existing user sessions.
func (s *Store) UpdateUserPasswordHash(ctx context.Context, userID int64, hash string) error {
	return s.updateUserPasswordHash(ctx, userID, hash, "")
}

func (s *Store) UpdateUserPasswordHashKeepingSession(ctx context.Context, userID int64, hash, keepTokenHash string) error {
	return s.updateUserPasswordHash(ctx, userID, hash, keepTokenHash)
}

func (s *Store) updateUserPasswordHash(ctx context.Context, userID int64, hash, keepTokenHash string) error {
	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := s.Rebind(`UPDATE users SET password_hash = ? WHERE id = ?`)
	if _, err := tx.ExecContext(ctx, q, hash, userID); err != nil {
		return err
	}
	if keepTokenHash == "" {
		q = s.Rebind(`DELETE FROM user_sessions WHERE user_id = ?`)
		if _, err := tx.ExecContext(ctx, q, userID); err != nil {
			return err
		}
	} else {
		q = s.Rebind(`DELETE FROM user_sessions WHERE user_id = ? AND token_hash <> ?`)
		if _, err := tx.ExecContext(ctx, q, userID, keepTokenHash); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) DeleteUserSessions(ctx context.Context, userID int64) error {
	q := s.Rebind(`DELETE FROM user_sessions WHERE user_id = ?`)
	_, err := s.DB.ExecContext(ctx, q, userID)
	return err
}
