package store

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type StringList []string

func (s StringList) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	b, err := json.Marshal([]string(s))
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (s *StringList) Scan(value any) error {
	if value == nil {
		*s = nil
		return nil
	}
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("scan StringList from %T", value)
	}
	if len(raw) == 0 {
		*s = nil
		return nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return err
	}
	*s = out
	return nil
}

type Plan struct {
	ID                int64      `db:"id" json:"id"`
	Name              string     `db:"name" json:"name"`
	Price             string     `db:"price" json:"price"`
	QuarterlyPrice    string     `db:"quarterly_price" json:"quarterly_price"`
	YearlyPrice       string     `db:"yearly_price" json:"yearly_price"`
	ResetTrafficPrice string     `db:"reset_traffic_price" json:"reset_traffic_price"`
	DurationDays      int        `db:"duration_days" json:"duration_days"`
	TrafficLimit      int64      `db:"traffic_limit" json:"traffic_limit"`
	DeviceLimit       int        `db:"device_limit" json:"device_limit"`
	SpeedLimit        int        `db:"speed_limit" json:"speed_limit"`
	Features          StringList `db:"features_json" json:"features"`
	NodeGroupID       *int64     `db:"node_group_id" json:"node_group_id"`
	Status            string     `db:"status" json:"status"`
	Sort              int        `db:"sort" json:"sort"`
	CreatedAt         time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at" json:"updated_at"`
}

type CreatePlanInput struct {
	Name              string
	Price             string
	QuarterlyPrice    string
	YearlyPrice       string
	ResetTrafficPrice string
	DurationDays      int
	TrafficLimit      int64
	DeviceLimit       int
	SpeedLimit        int
	Features          []string
	NodeGroupID       *int64
	Sort              int
}

type UpdatePlanInput struct {
	Name              string
	Price             string
	QuarterlyPrice    string
	YearlyPrice       string
	ResetTrafficPrice string
	DurationDays      int
	TrafficLimit      int64
	DeviceLimit       int
	SpeedLimit        int
	Features          []string
	NodeGroupID       *int64
	Status            string
	Sort              int
}

func (s *Store) CreatePlan(ctx context.Context, in CreatePlanInput) (int64, error) {
	normalizePlanPrices(&in.Price, &in.QuarterlyPrice, &in.YearlyPrice, &in.ResetTrafficPrice)
	return s.InsertReturningID(ctx,
		`INSERT INTO plans(name, price, quarterly_price, yearly_price, reset_traffic_price, duration_days, traffic_limit, device_limit,
			speed_limit, features_json, node_group_id, sort) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		in.Name, in.Price, in.QuarterlyPrice, in.YearlyPrice, in.ResetTrafficPrice, in.DurationDays, in.TrafficLimit, in.DeviceLimit,
		0, StringList(in.Features), in.NodeGroupID, in.Sort,
	)
}

func normalizePlanPrices(price, quarterlyPrice, yearlyPrice, resetTrafficPrice *string) {
	*price = normalizeMoneyText(*price)
	*quarterlyPrice = normalizeMoneyText(*quarterlyPrice)
	*yearlyPrice = normalizeMoneyText(*yearlyPrice)
	*resetTrafficPrice = normalizeMoneyText(*resetTrafficPrice)
}

func normalizeMoneyText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "0.00"
	}
	return value
}

func (s *Store) UpdatePlan(ctx context.Context, id int64, in UpdatePlanInput) error {
	normalizePlanPrices(&in.Price, &in.QuarterlyPrice, &in.YearlyPrice, &in.ResetTrafficPrice)
	q := s.Rebind(`UPDATE plans SET name = ?, price = ?, quarterly_price = ?, yearly_price = ?, reset_traffic_price = ?, duration_days = ?,
		traffic_limit = ?, device_limit = ?, speed_limit = ?, features_json = ?,
		node_group_id = ?, status = ?, sort = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`)
	if _, err := s.DB.ExecContext(ctx, q, in.Name, in.Price, in.QuarterlyPrice, in.YearlyPrice, in.ResetTrafficPrice, in.DurationDays,
		in.TrafficLimit, in.DeviceLimit, 0, StringList(in.Features),
		in.NodeGroupID, in.Status, in.Sort, id); err != nil {
		return err
	}
	return s.ApplyPlanLimitsToUsersByPlanID(ctx, id, in.DeviceLimit)
}

func (s *Store) ListActivePlans(ctx context.Context) ([]Plan, error) {
	q := `SELECT id, name, price, quarterly_price, yearly_price, reset_traffic_price, duration_days, traffic_limit, device_limit,
		speed_limit, features_json, node_group_id, status, sort, created_at, updated_at
		FROM plans WHERE status = 'active' ORDER BY sort ASC, id ASC`
	var rows []Plan
	if err := s.DB.SelectContext(ctx, &rows, q); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) ListAllPlans(ctx context.Context) ([]Plan, error) {
	rows, _, err := s.ListAllPlansPage(ctx, PageParams{Page: 1, PageSize: 500})
	return rows, err
}

func (s *Store) ListAllPlansPage(ctx context.Context, p PageParams) ([]Plan, int64, error) {
	p = NormalizePage(p)
	var total int64
	if err := s.DB.GetContext(ctx, &total, `SELECT COUNT(*) FROM plans`); err != nil {
		return nil, 0, err
	}
	q := s.Rebind(`SELECT id, name, price, quarterly_price, yearly_price, reset_traffic_price, duration_days, traffic_limit, device_limit,
		speed_limit, features_json, node_group_id, status, sort, created_at, updated_at
		FROM plans ORDER BY sort ASC, id ASC LIMIT ? OFFSET ?`)
	var rows []Plan
	if err := s.DB.SelectContext(ctx, &rows, q, p.PageSize, p.Offset()); err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (s *Store) FindPlanByID(ctx context.Context, id int64) (*Plan, error) {
	q := s.Rebind(`SELECT id, name, price, quarterly_price, yearly_price, reset_traffic_price, duration_days, traffic_limit, device_limit,
		speed_limit, features_json, node_group_id, status, sort, created_at, updated_at
		FROM plans WHERE id = ?`)
	var p Plan
	if err := s.DB.GetContext(ctx, &p, q, id); err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) ApplyPlanLimitsToUsersByPlanID(ctx context.Context, planID int64, deviceLimit int) error {
	q := s.Rebind(`UPDATE node_users SET speed_limit = ?, device_limit = ?,
		updated_at = CURRENT_TIMESTAMP
		WHERE user_id IN (SELECT id FROM users WHERE plan_id = ?)`)
	_, err := s.DB.ExecContext(ctx, q, 0, deviceLimit, planID)
	return err
}
