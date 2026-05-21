package store

import (
	"context"

	"github.com/zboard/api-server/internal/config"
)

func (s *Store) CountUserDevices(ctx context.Context, userID int64) (int, error) {
	q := s.Rebind(`SELECT COUNT(*) FROM user_devices WHERE user_id = ?`)
	var n int
	if err := s.DB.GetContext(ctx, &n, q, userID); err != nil {
		return 0, err
	}
	return n, nil
}

func (s *Store) HasUserDevice(ctx context.Context, userID int64, fingerprint string) (bool, error) {
	q := s.Rebind(`SELECT COUNT(*) FROM user_devices WHERE user_id = ? AND fingerprint = ?`)
	var n int
	if err := s.DB.GetContext(ctx, &n, q, userID, fingerprint); err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *Store) TouchUserDevice(ctx context.Context, userID int64, fingerprint, ip, ua string) error {
	switch s.Dialect {
	case config.DialectMySQL:
		q := s.Rebind(`INSERT INTO user_devices(user_id, fingerprint, ip, user_agent)
			VALUES (?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE ip = VALUES(ip), user_agent = VALUES(user_agent), last_seen_at = CURRENT_TIMESTAMP`)
		_, err := s.DB.ExecContext(ctx, q, userID, fingerprint, ip, ua)
		return err
	case config.DialectPostgres:
		q := s.Rebind(`INSERT INTO user_devices(user_id, fingerprint, ip, user_agent)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (user_id, fingerprint) DO UPDATE SET
				ip = EXCLUDED.ip, user_agent = EXCLUDED.user_agent, last_seen_at = NOW()`)
		_, err := s.DB.ExecContext(ctx, q, userID, fingerprint, ip, ua)
		return err
	default:
		q := s.Rebind(`INSERT INTO user_devices(user_id, fingerprint, ip, user_agent)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(user_id, fingerprint) DO UPDATE SET
				ip = excluded.ip, user_agent = excluded.user_agent, last_seen_at = CURRENT_TIMESTAMP`)
		_, err := s.DB.ExecContext(ctx, q, userID, fingerprint, ip, ua)
		return err
	}
}
