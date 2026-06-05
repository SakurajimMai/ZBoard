package store

import (
	"context"
	"time"
)

type ActiveUserTraffic struct {
	UserID        int64   `db:"user_id" json:"user_id"`
	Email         string  `db:"email" json:"email"`
	Status        string  `db:"status" json:"status"`
	PlanID        *int64  `db:"plan_id" json:"plan_id"`
	UploadTotal   int64   `db:"upload_total" json:"upload_total"`
	DownloadTotal int64   `db:"download_total" json:"download_total"`
	TrafficTotal  int64   `db:"traffic_total" json:"traffic_total"`
	NodeCount     int64   `db:"node_count" json:"node_count"`
	FirstActiveAt string  `db:"first_active_at" json:"first_active_at"`
	LastActiveAt  string  `db:"last_active_at" json:"last_active_at"`
	PlanName      *string `db:"plan_name" json:"plan_name"`
}

type ActiveUsersSummary struct {
	ActiveUsers   int64 `db:"active_users" json:"active_users"`
	UploadTotal   int64 `db:"upload_total" json:"upload_total"`
	DownloadTotal int64 `db:"download_total" json:"download_total"`
	TrafficTotal  int64 `db:"traffic_total" json:"traffic_total"`
}

func (s *Store) ListActiveUsersByDay(ctx context.Context, day time.Time, p PageParams) ([]ActiveUserTraffic, int64, ActiveUsersSummary, error) {
	p = NormalizePage(p)
	start := dayStart(day.UTC())
	end := start.AddDate(0, 0, 1)

	activeSubquery := `SELECT user_id,
			COALESCE(SUM(upload_delta), 0) AS upload_total,
			COALESCE(SUM(download_delta), 0) AS download_total,
			COALESCE(SUM(` + trafficTotalExpr("") + `), 0) AS traffic_total,
			COUNT(DISTINCT node_id) AS node_count,
			MIN(reported_at) AS first_active_at,
			MAX(reported_at) AS last_active_at
		FROM traffic_logs
		WHERE reported_at >= ? AND reported_at < ? AND ` + trafficTotalExpr("") + ` > 0
		GROUP BY user_id`

	var total int64
	countQuery := s.Rebind(`SELECT COUNT(*) FROM (` + activeSubquery + `) active`)
	if err := s.DB.GetContext(ctx, &total, countQuery, start, end); err != nil {
		return nil, 0, ActiveUsersSummary{}, err
	}

	var summary ActiveUsersSummary
	summaryQuery := s.Rebind(`SELECT
			COALESCE(COUNT(*), 0) AS active_users,
			COALESCE(SUM(upload_total), 0) AS upload_total,
			COALESCE(SUM(download_total), 0) AS download_total,
			COALESCE(SUM(traffic_total), 0) AS traffic_total
		FROM (` + activeSubquery + `) active`)
	if err := s.DB.GetContext(ctx, &summary, summaryQuery, start, end); err != nil {
		return nil, 0, ActiveUsersSummary{}, err
	}

	query := s.Rebind(`SELECT active.user_id,
			u.email,
			u.status,
			u.plan_id,
			active.upload_total,
			active.download_total,
			active.traffic_total,
			active.node_count,
			active.first_active_at,
			active.last_active_at,
			p.name AS plan_name
		FROM (` + activeSubquery + `) active
		INNER JOIN users u ON u.id = active.user_id
		LEFT JOIN plans p ON p.id = u.plan_id
		ORDER BY active.traffic_total DESC, active.last_active_at DESC, active.user_id ASC
		LIMIT ? OFFSET ?`)
	var rows []ActiveUserTraffic
	if err := s.DB.SelectContext(ctx, &rows, query, start, end, p.PageSize, p.Offset()); err != nil {
		return nil, 0, ActiveUsersSummary{}, err
	}
	return rows, total, summary, nil
}
