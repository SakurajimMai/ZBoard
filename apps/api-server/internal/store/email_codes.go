package store

import (
	"context"
	"time"
)

const EmailCodeMaxAttempts = 5

type EmailCode struct {
	ID             int64      `db:"id"`
	Email          string     `db:"email"`
	Code           string     `db:"code"`
	Purpose        string     `db:"purpose"`
	Used           int        `db:"used"`
	FailedAttempts int        `db:"failed_attempts"`
	LockedAt       *time.Time `db:"locked_at"`
	ExpiresAt      time.Time  `db:"expires_at"`
	LastSentAt     time.Time  `db:"last_sent_at"`
	CreatedAt      time.Time  `db:"created_at"`
}

// FindLatestEmailCode returns the most recent unused code for (email, purpose).
func (s *Store) FindLatestEmailCode(ctx context.Context, email, purpose string) (*EmailCode, error) {
	q := s.Rebind(`SELECT id, email, code, purpose, used, failed_attempts, locked_at,
		expires_at, last_sent_at, created_at
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

// VerifyEmailCode consumes a matching unexpired code and counts wrong attempts
// against the latest outstanding code for the same email and purpose.
func (s *Store) VerifyEmailCode(ctx context.Context, email, code, purpose string) (bool, error) {
	now := time.Now().UTC()
	q := s.Rebind(`SELECT id, code, failed_attempts, locked_at
		FROM email_codes
		WHERE email = ? AND purpose = ? AND used = 0 AND expires_at > ?
		ORDER BY id DESC LIMIT 1`)
	var row struct {
		ID             int64      `db:"id"`
		Code           string     `db:"code"`
		FailedAttempts int        `db:"failed_attempts"`
		LockedAt       *time.Time `db:"locked_at"`
	}
	if err := s.DB.GetContext(ctx, &row, q, email, purpose, now); err != nil {
		if IsNoRows(err) {
			return false, nil
		}
		return false, err
	}
	if row.LockedAt != nil || row.FailedAttempts >= EmailCodeMaxAttempts {
		return false, nil
	}
	if row.Code == code {
		upd := s.Rebind(`UPDATE email_codes SET used = 1
			WHERE id = ? AND used = 0 AND locked_at IS NULL AND failed_attempts < ?`)
		res, err := s.DB.ExecContext(ctx, upd, row.ID, EmailCodeMaxAttempts)
		if err != nil {
			return false, err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return false, err
		}
		return affected > 0, nil
	}

	upd := s.Rebind(`UPDATE email_codes
		SET failed_attempts = failed_attempts + 1,
			locked_at = CASE WHEN failed_attempts + 1 >= ? THEN ? ELSE locked_at END
		WHERE id = ? AND used = 0 AND locked_at IS NULL AND failed_attempts < ?`)
	if _, err := s.DB.ExecContext(ctx, upd, EmailCodeMaxAttempts, now, row.ID, EmailCodeMaxAttempts); err != nil {
		return false, err
	}
	return false, nil
}
