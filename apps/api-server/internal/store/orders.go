package store

import (
	"context"
	"time"
)

type Order struct {
	ID          int64      `db:"id" json:"id"`
	OrderNo     string     `db:"order_no" json:"order_no"`
	UserID      int64      `db:"user_id" json:"user_id"`
	PlanID      int64      `db:"plan_id" json:"plan_id"`
	Kind        string     `db:"kind" json:"kind"`
	Amount      string     `db:"amount" json:"amount"`
	Currency    string     `db:"currency" json:"currency"`
	Status      string     `db:"status" json:"status"`
	PaidAt      *time.Time `db:"paid_at" json:"paid_at"`
	CancelledAt *time.Time `db:"cancelled_at" json:"cancelled_at"`
	ExpiredAt   *time.Time `db:"expired_at" json:"expired_at"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

const (
	OrderKindPlan         = "plan"
	OrderKindTrafficReset = "traffic_reset"
)

func (s *Store) CreateOrder(ctx context.Context, o *Order) (int64, error) {
	if o.Kind == "" {
		o.Kind = OrderKindPlan
	}
	return s.InsertReturningID(ctx,
		`INSERT INTO orders(order_no, user_id, plan_id, kind, amount, currency, status, expired_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		o.OrderNo, o.UserID, o.PlanID, o.Kind, o.Amount, o.Currency, o.Status, o.ExpiredAt,
	)
}

func (s *Store) FindOrderByNo(ctx context.Context, orderNo string) (*Order, error) {
	q := s.Rebind(`SELECT id, order_no, user_id, plan_id, kind, amount, currency, status,
		paid_at, cancelled_at, expired_at, created_at, updated_at
		FROM orders WHERE order_no = ?`)
	var o Order
	if err := s.DB.GetContext(ctx, &o, q, orderNo); err != nil {
		return nil, err
	}
	return &o, nil
}

func (s *Store) MarkOrderPaid(ctx context.Context, orderNo string, paidAt time.Time) error {
	q := s.Rebind(`UPDATE orders SET status = 'paid', paid_at = ? WHERE order_no = ? AND status <> 'paid'`)
	_, err := s.DB.ExecContext(ctx, q, paidAt, orderNo)
	return err
}

func (s *Store) ListOrdersByUser(ctx context.Context, userID int64, limit int) ([]Order, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := s.Rebind(`SELECT id, order_no, user_id, plan_id, kind, amount, currency, status,
		paid_at, cancelled_at, expired_at, created_at, updated_at
		FROM orders WHERE user_id = ? ORDER BY id DESC LIMIT ?`)
	var rows []Order
	if err := s.DB.SelectContext(ctx, &rows, q, userID, limit); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) ListAllOrders(ctx context.Context, limit int) ([]Order, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, _, err := s.ListAllOrdersPage(ctx, PageParams{Page: 1, PageSize: limit})
	return rows, err
}

func (s *Store) ListAllOrdersPage(ctx context.Context, p PageParams) ([]Order, int64, error) {
	p = NormalizePage(p)
	var total int64
	if err := s.DB.GetContext(ctx, &total, `SELECT COUNT(*) FROM orders`); err != nil {
		return nil, 0, err
	}
	q := s.Rebind(`SELECT id, order_no, user_id, plan_id, kind, amount, currency, status,
		paid_at, cancelled_at, expired_at, created_at, updated_at
		FROM orders ORDER BY id DESC LIMIT ? OFFSET ?`)
	var rows []Order
	if err := s.DB.SelectContext(ctx, &rows, q, p.PageSize, p.Offset()); err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
