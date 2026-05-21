package store

import (
	"context"
	"fmt"
)

type AdminOverview struct {
	Users       int64  `json:"users"`
	ActiveNodes int64  `json:"active_nodes"`
	PaidOrders  int64  `json:"paid_orders"`
	Revenue     string `json:"revenue"`
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
	return &out, nil
}
