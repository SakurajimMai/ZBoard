package store

import (
	"context"
	"fmt"

	"github.com/zboard/api-server/internal/config"
)

type TrafficDelta struct {
	UserID        int64
	NodeID        int64
	UploadDelta   int64
	DownloadDelta int64
}

func (s *Store) RecordTraffic(ctx context.Context, deltas []TrafficDelta) error {
	if len(deltas) == 0 {
		return nil
	}
	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := Now()
	insertLog := s.Rebind(`INSERT INTO traffic_logs(user_id, node_id, upload_delta, download_delta, total_delta, reported_at)
		VALUES (?, ?, ?, ?, ?, ?)`)
	upsertSnap := s.Rebind(upsertSnapshotSQL(s.Dialect))
	bumpUser := s.Rebind(`UPDATE users SET traffic_used = traffic_used + ? WHERE id = ?`)
	bumpNU := s.Rebind(`UPDATE node_users SET upload = upload + ?, download = download + ? WHERE user_id = ? AND node_id = ?`)

	for _, d := range deltas {
		total := d.UploadDelta + d.DownloadDelta
		if _, err := tx.ExecContext(ctx, insertLog, d.UserID, d.NodeID, d.UploadDelta, d.DownloadDelta, total, now); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, upsertSnap, d.UserID, d.UploadDelta, d.DownloadDelta, total, d.UploadDelta, d.DownloadDelta, total); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, bumpUser, total, d.UserID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, bumpNU, d.UploadDelta, d.DownloadDelta, d.UserID, d.NodeID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func upsertSnapshotSQL(dialect config.Dialect) string {
	switch dialect {
	case config.DialectMySQL:
		return `INSERT INTO user_traffic_snapshots(user_id, upload_total, download_total, total_used)
			VALUES (?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE upload_total = upload_total + ?, download_total = download_total + ?, total_used = total_used + ?`
	case config.DialectPostgres:
		return `INSERT INTO user_traffic_snapshots(user_id, upload_total, download_total, total_used)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (user_id) DO UPDATE
				SET upload_total = user_traffic_snapshots.upload_total + EXCLUDED.upload_total,
				    download_total = user_traffic_snapshots.download_total + EXCLUDED.download_total,
				    total_used = user_traffic_snapshots.total_used + EXCLUDED.total_used`
	default:
		return `INSERT INTO user_traffic_snapshots(user_id, upload_total, download_total, total_used)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(user_id) DO UPDATE SET
				upload_total = user_traffic_snapshots.upload_total + excluded.upload_total,
				download_total = user_traffic_snapshots.download_total + excluded.download_total,
				total_used = user_traffic_snapshots.total_used + excluded.total_used`
	}
}

type UserTrafficSnapshot struct {
	UserID        int64  `db:"user_id" json:"user_id"`
	UploadTotal   int64  `db:"upload_total" json:"upload_total"`
	DownloadTotal int64  `db:"download_total" json:"download_total"`
	TotalUsed     int64  `db:"total_used" json:"total_used"`
	TrafficLimit  int64  `db:"traffic_limit" json:"traffic_limit"`
	UpdatedAt     string `db:"updated_at" json:"updated_at"`
}

func (s *Store) ListTrafficSnapshots(ctx context.Context, limit int) ([]UserTrafficSnapshot, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := s.Rebind(`SELECT s.user_id, s.upload_total, s.download_total, u.traffic_used AS total_used,
			u.traffic_limit, s.updated_at
		FROM user_traffic_snapshots s
		JOIN users u ON u.id = s.user_id
		ORDER BY u.traffic_used DESC LIMIT ?`)
	var rows []UserTrafficSnapshot
	if err := s.DB.SelectContext(ctx, &rows, q, limit); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) FindTrafficSnapshotByUser(ctx context.Context, userID int64) (*UserTrafficSnapshot, error) {
	q := s.Rebind(`SELECT s.user_id, s.upload_total, s.download_total, u.traffic_used AS total_used,
			u.traffic_limit, s.updated_at
		FROM user_traffic_snapshots s
		JOIN users u ON u.id = s.user_id
		WHERE s.user_id = ?`)
	var row UserTrafficSnapshot
	if err := s.DB.GetContext(ctx, &row, q, userID); err != nil {
		return nil, err
	}
	return &row, nil
}

type TrafficLog struct {
	ID            int64  `db:"id" json:"id"`
	UserID        int64  `db:"user_id" json:"user_id"`
	NodeID        int64  `db:"node_id" json:"node_id"`
	UploadDelta   int64  `db:"upload_delta" json:"upload_delta"`
	DownloadDelta int64  `db:"download_delta" json:"download_delta"`
	TotalDelta    int64  `db:"total_delta" json:"total_delta"`
	ReportedAt    string `db:"reported_at" json:"reported_at"`
	CreatedAt     string `db:"created_at" json:"created_at"`
}

func (s *Store) ListTrafficLogs(ctx context.Context, limit int) ([]TrafficLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := s.Rebind(`SELECT id, user_id, node_id, upload_delta, download_delta, total_delta, reported_at, created_at
		FROM traffic_logs ORDER BY id DESC LIMIT ?`)
	var rows []TrafficLog
	if err := s.DB.SelectContext(ctx, &rows, q, limit); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) ListTrafficLogsByUser(ctx context.Context, userID int64, limit int) ([]TrafficLog, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	q := s.Rebind(`SELECT id, user_id, node_id, upload_delta, download_delta, total_delta, reported_at, created_at
		FROM traffic_logs WHERE user_id = ? ORDER BY id DESC LIMIT ?`)
	var rows []TrafficLog
	if err := s.DB.SelectContext(ctx, &rows, q, userID, limit); err != nil {
		return nil, err
	}
	return rows, nil
}

// DailyTrafficPoint aggregates a user's traffic by calendar day.
type DailyTrafficPoint struct {
	Day      string `db:"day" json:"day"`
	Upload   int64  `db:"upload" json:"upload"`
	Download int64  `db:"download" json:"download"`
	Total    int64  `db:"total" json:"total"`
}

// ListDailyTrafficByUser returns per-day upload/download/total bytes for the
// given user over the last `days` days (UTC).
func (s *Store) ListDailyTrafficByUser(ctx context.Context, userID int64, days int) ([]DailyTrafficPoint, error) {
	if days <= 0 || days > 365 {
		days = 30
	}
	var q string
	switch s.Dialect {
	case "mysql":
		q = `SELECT DATE_FORMAT(reported_at, '%Y-%m-%d') AS day,
			COALESCE(SUM(upload_delta), 0) AS upload,
			COALESCE(SUM(download_delta), 0) AS download,
			COALESCE(SUM(total_delta), 0) AS total
			FROM traffic_logs
			WHERE user_id = ? AND reported_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL ? DAY)
			GROUP BY day ORDER BY day ASC`
	case "postgres":
		q = `SELECT TO_CHAR(reported_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS day,
			COALESCE(SUM(upload_delta), 0) AS upload,
			COALESCE(SUM(download_delta), 0) AS download,
			COALESCE(SUM(total_delta), 0) AS total
			FROM traffic_logs
			WHERE user_id = $1 AND reported_at >= NOW() - ($2::int || ' days')::interval
			GROUP BY day ORDER BY day ASC`
	default:
		q = `SELECT strftime('%Y-%m-%d', reported_at) AS day,
			COALESCE(SUM(upload_delta), 0) AS upload,
			COALESCE(SUM(download_delta), 0) AS download,
			COALESCE(SUM(total_delta), 0) AS total
			FROM traffic_logs
			WHERE user_id = ? AND reported_at >= datetime('now', ?)
			GROUP BY day ORDER BY day ASC`
	}
	var rows []DailyTrafficPoint
	var err error
	switch s.Dialect {
	case "postgres":
		err = s.DB.SelectContext(ctx, &rows, q, userID, days)
	case "mysql":
		err = s.DB.SelectContext(ctx, &rows, q, userID, days)
	default:
		err = s.DB.SelectContext(ctx, &rows, q, userID, fmt.Sprintf("-%d days", days))
	}
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// ResetUserTraffic zeroes out the user's traffic counters across users,
// user_traffic_snapshots, and node_users. Idempotent.
func (s *Store) ResetUserTraffic(ctx context.Context, userID int64) error {
	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, s.Rebind(`UPDATE users SET traffic_used = 0 WHERE id = ?`), userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		s.Rebind(`UPDATE user_traffic_snapshots SET upload_total = 0, download_total = 0, total_used = 0 WHERE user_id = ?`),
		userID,
	); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		s.Rebind(`UPDATE node_users SET upload = 0, download = 0 WHERE user_id = ?`),
		userID,
	); err != nil {
		return err
	}
	return tx.Commit()
}

// RotateUserClientID assigns a fresh client_id to every node_users row of a
// user. The new id is the same value across all rows so the user's
// subscription stays a single coherent identity.
func (s *Store) RotateUserClientID(ctx context.Context, userID int64, newClientID string) error {
	q := s.Rebind(`UPDATE node_users SET client_id = ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = ?`)
	_, err := s.DB.ExecContext(ctx, q, newClientID, userID)
	return err
}
