package store

import (
	"context"
	"fmt"
	"time"

	"github.com/zboard/api-server/internal/config"
)

type AdminOverview struct {
	Users        int64               `json:"users"`
	ActiveNodes  int64               `json:"active_nodes"`
	PaidOrders   int64               `json:"paid_orders"`
	Revenue      string              `json:"revenue"`
	RevenueTrend []RevenueTrendPoint `json:"revenue_trend"`
	TrafficTrend []TrafficTrendPoint `json:"traffic_trend"`
}

type RevenueTrendPoint struct {
	Month   string  `json:"month"`
	Label   string  `json:"label"`
	Revenue float64 `json:"revenue"`
}

type TrafficTrendPoint struct {
	Day   string  `json:"day"`
	Label string  `json:"label"`
	Total int64   `json:"total"`
	TB    float64 `json:"tb"`
}

func (s *Store) AdminOverview(ctx context.Context) (*AdminOverview, error) {
	var out AdminOverview
	if err := s.DB.GetContext(ctx, &out.Users, `SELECT COUNT(*) FROM users`); err != nil {
		return nil, err
	}
	if err := s.DB.GetContext(ctx, &out.ActiveNodes, `SELECT COUNT(*) FROM nodes WHERE status = 'active'`); err != nil {
		return nil, err
	}
	if err := s.DB.GetContext(ctx, &out.PaidOrders, `SELECT COUNT(*) FROM orders WHERE status = 'paid'`); err != nil {
		return nil, err
	}
	var revenue float64
	if err := s.DB.GetContext(ctx, &revenue, `SELECT COALESCE(SUM(amount), 0) FROM orders WHERE status = 'paid'`); err != nil {
		return nil, err
	}
	out.Revenue = fmt.Sprintf("%.2f", revenue)
	revenueTrend, err := s.AdminRevenueTrend(ctx, 6)
	if err != nil {
		return nil, err
	}
	out.RevenueTrend = revenueTrend
	trafficTrend, err := s.AdminTrafficTrend(ctx, 7)
	if err != nil {
		return nil, err
	}
	out.TrafficTrend = trafficTrend
	return &out, nil
}

func (s *Store) AdminRevenueTrend(ctx context.Context, months int) ([]RevenueTrendPoint, error) {
	if months <= 0 || months > 24 {
		months = 6
	}
	start := monthStart(time.Now().UTC()).AddDate(0, -months+1, 0)
	rows := make([]RevenueTrendPoint, 0)
	var query string
	var args []any
	switch s.Dialect {
	case config.DialectMySQL:
		query = `SELECT DATE_FORMAT(COALESCE(paid_at, updated_at, created_at), '%Y-%m') AS month,
			COALESCE(SUM(amount), 0) AS revenue
			FROM orders
			WHERE status = 'paid' AND COALESCE(paid_at, updated_at, created_at) >= ?
			GROUP BY month ORDER BY month ASC`
		args = []any{start}
	case config.DialectPostgres:
		query = `SELECT TO_CHAR(COALESCE(paid_at, updated_at, created_at) AT TIME ZONE 'UTC', 'YYYY-MM') AS month,
			COALESCE(SUM(amount::numeric), 0)::float8 AS revenue
			FROM orders
			WHERE status = 'paid' AND COALESCE(paid_at, updated_at, created_at) >= $1
			GROUP BY month ORDER BY month ASC`
		args = []any{start}
	default:
		query = `SELECT strftime('%Y-%m', COALESCE(paid_at, updated_at, created_at)) AS month,
			COALESCE(SUM(CAST(amount AS REAL)), 0) AS revenue
			FROM orders
			WHERE status = 'paid' AND COALESCE(paid_at, updated_at, created_at) >= ?
			GROUP BY month ORDER BY month ASC`
		args = []any{start.Format("2006-01-02 15:04:05")}
	}
	if err := s.DB.SelectContext(ctx, &rows, s.Rebind(query), args...); err != nil {
		return nil, err
	}
	byMonth := make(map[string]float64, len(rows))
	for _, row := range rows {
		byMonth[row.Month] = row.Revenue
	}
	out := make([]RevenueTrendPoint, 0, months)
	for i := 0; i < months; i++ {
		t := start.AddDate(0, i, 0)
		month := t.Format("2006-01")
		out = append(out, RevenueTrendPoint{
			Month:   month,
			Label:   fmt.Sprintf("%d月", int(t.Month())),
			Revenue: byMonth[month],
		})
	}
	return out, nil
}

func (s *Store) AdminTrafficTrend(ctx context.Context, days int) ([]TrafficTrendPoint, error) {
	if days <= 0 || days > 60 {
		days = 7
	}
	start := dayStart(time.Now().UTC()).AddDate(0, 0, -days+1)
	rows := make([]TrafficTrendPoint, 0)
	var query string
	var args []any
	switch s.Dialect {
	case config.DialectMySQL:
		query = `SELECT DATE_FORMAT(reported_at, '%Y-%m-%d') AS day,
			COALESCE(SUM(total_delta), 0) AS total
			FROM traffic_logs
			WHERE reported_at >= ?
			GROUP BY day ORDER BY day ASC`
		args = []any{start}
	case config.DialectPostgres:
		query = `SELECT TO_CHAR(reported_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS day,
			COALESCE(SUM(total_delta), 0) AS total
			FROM traffic_logs
			WHERE reported_at >= $1
			GROUP BY day ORDER BY day ASC`
		args = []any{start}
	default:
		query = `SELECT strftime('%Y-%m-%d', reported_at) AS day,
			COALESCE(SUM(total_delta), 0) AS total
			FROM traffic_logs
			WHERE reported_at >= ?
			GROUP BY day ORDER BY day ASC`
		args = []any{start.Format("2006-01-02 15:04:05")}
	}
	if err := s.DB.SelectContext(ctx, &rows, s.Rebind(query), args...); err != nil {
		return nil, err
	}
	byDay := make(map[string]int64, len(rows))
	for _, row := range rows {
		byDay[row.Day] = row.Total
	}
	out := make([]TrafficTrendPoint, 0, days)
	for i := 0; i < days; i++ {
		t := start.AddDate(0, 0, i)
		day := t.Format("2006-01-02")
		total := byDay[day]
		out = append(out, TrafficTrendPoint{
			Day:   day,
			Label: fmt.Sprintf("%d/%d", int(t.Month()), t.Day()),
			Total: total,
			TB:    float64(total) / 1099511627776,
		})
	}
	return out, nil
}

func monthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func dayStart(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
