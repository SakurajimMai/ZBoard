package store

import (
	"context"

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
