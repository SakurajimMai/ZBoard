package store

import (
	"context"
	"time"
)

type Plan struct {
	ID           int64     `db:"id" json:"id"`
	Name         string    `db:"name" json:"name"`
	Price        string    `db:"price" json:"price"`
	DurationDays int       `db:"duration_days" json:"duration_days"`
	TrafficLimit int64     `db:"traffic_limit" json:"traffic_limit"`
	DeviceLimit  int       `db:"device_limit" json:"device_limit"`
	SpeedLimit   int       `db:"speed_limit" json:"speed_limit"`
	NodeGroupID  *int64    `db:"node_group_id" json:"node_group_id"`
	Status       string    `db:"status" json:"status"`
	Sort         int       `db:"sort" json:"sort"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type CreatePlanInput struct {
	Name         string
	Price        string
	DurationDays int
	TrafficLimit int64
	DeviceLimit  int
	SpeedLimit   int
	NodeGroupID  *int64
	Sort         int
}

func (s *Store) CreatePlan(ctx context.Context, in CreatePlanInput) (int64, error) {
	return s.InsertReturningID(ctx,
		`INSERT INTO plans(name, price, duration_days, traffic_limit, device_limit,
			speed_limit, node_group_id, sort) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		in.Name, in.Price, in.DurationDays, in.TrafficLimit, in.DeviceLimit,
		in.SpeedLimit, in.NodeGroupID, in.Sort,
	)
}

func (s *Store) ListActivePlans(ctx context.Context) ([]Plan, error) {
	q := `SELECT id, name, price, duration_days, traffic_limit, device_limit,
		speed_limit, node_group_id, status, sort, created_at, updated_at
		FROM plans WHERE status = 'active' ORDER BY sort ASC, id ASC`
	var rows []Plan
	if err := s.DB.SelectContext(ctx, &rows, q); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) ListAllPlans(ctx context.Context) ([]Plan, error) {
	q := `SELECT id, name, price, duration_days, traffic_limit, device_limit,
		speed_limit, node_group_id, status, sort, created_at, updated_at
		FROM plans ORDER BY sort ASC, id ASC`
	var rows []Plan
	if err := s.DB.SelectContext(ctx, &rows, q); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) FindPlanByID(ctx context.Context, id int64) (*Plan, error) {
	q := s.Rebind(`SELECT id, name, price, duration_days, traffic_limit, device_limit,
		speed_limit, node_group_id, status, sort, created_at, updated_at
		FROM plans WHERE id = ?`)
	var p Plan
	if err := s.DB.GetContext(ctx, &p, q, id); err != nil {
		return nil, err
	}
	return &p, nil
}
