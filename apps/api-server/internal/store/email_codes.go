package store

import (
	"context"
	"time"
)

type EmailCode struct {
	ID         int64     `db:"id"`
	Email      string    `db:"email"`
	Code       string    `db:"code"`
	Purpose    string    `db:"purpose"`
	Used       int       `db:"used"`
	ExpiresAt  time.Time `db:"expires_at"`
	LastSentAt time.Time `db:"last_sent_at"`
	CreatedAt  time.Time `db:"created_at"`
}

// FindLatestEmailCode returns the most recent unused code for (email, purpose).
func (s *Store) FindLatestEmailCode(ctx context.Context, email, purpose string) (*EmailCode, error) {
	q := s.Rebind(`SELECT id, email, code, purpose, used, expires_at, last_sent_at, created_at
		FROM email_codes WHERE email = ? AND purpose = ? AND used = 0
		ORDER BY id DESC LIMIT 1`)
	var c EmailCode
	if err := s.DB.GetContext(ctx, &c, q, email, purpose); err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) CreateEmailCode(ctx context.Context, email, code, purpose string, ttl time.Duration) error {
	q := s.Rebind(`INSERT INTO email_codes(email, code, purpose, expires_at) VALUES (?, ?, ?, ?)`)
	_, err := s.DB.ExecContext(ctx, q, email, code, purpose, time.Now().UTC().Add(ttl))
	return err
}

// VerifyEmailCode marks the code as used if it matches and is not expired.
func (s *Store) VerifyEmailCode(ctx context.Context, email, code, purpose string) (bool, error) {
	q := s.Rebind(`SELECT id FROM email_codes
		WHERE email = ? AND code = ? AND purpose = ? AND used = 0 AND expires_at > ?
		ORDER BY id DESC LIMIT 1`)
	var id int64
	if err := s.DB.GetContext(ctx, &id, q, email, code, purpose, time.Now().UTC()); err != nil {
		if IsNoRows(err) {
			return false, nil
		}
		return false, err
	}
	upd := s.Rebind(`UPDATE email_codes SET used = 1 WHERE id = ?`)
	if _, err := s.DB.ExecContext(ctx, upd, id); err != nil {
		return false, err
	}
	return true, nil
}
