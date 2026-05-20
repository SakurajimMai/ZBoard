package store

import (
	"context"
	"strings"
	"time"
)

type User struct {
	ID           int64      `db:"id"`
	Email        string     `db:"email"`
	PasswordHash string     `db:"password_hash"`
	Balance      string     `db:"balance"`
	PlanID       *int64     `db:"plan_id"`
	ExpiredAt    *time.Time `db:"expired_at"`
	TrafficLimit int64      `db:"traffic_limit"`
	TrafficUsed  int64      `db:"traffic_used"`
	Status       string     `db:"status"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
}

func (s *Store) CreateUser(ctx context.Context, email, passwordHash string) (int64, error) {
	return s.InsertReturningID(ctx,
		`INSERT INTO users(email, password_hash) VALUES (?, ?)`,
		email, passwordHash,
	)
}

type AdminCreateUserInput struct {
	Email        string
	PasswordHash string
	Balance      string
	PlanID       *int64
	ExpiredAt    *time.Time
	TrafficLimit int64
	TrafficUsed  int64
	Status       string
}

func (s *Store) AdminCreateUser(ctx context.Context, in AdminCreateUserInput) (int64, error) {
	if strings.TrimSpace(in.Balance) == "" {
		in.Balance = "0.00"
	}
	if strings.TrimSpace(in.Status) == "" {
		in.Status = "active"
	}
	return s.InsertReturningID(ctx,
		`INSERT INTO users(email, password_hash, balance, plan_id, expired_at,
			traffic_limit, traffic_used, status)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		in.Email, in.PasswordHash, in.Balance, in.PlanID, in.ExpiredAt,
		in.TrafficLimit, in.TrafficUsed, in.Status,
	)
}

func (s *Store) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	q := s.Rebind(`SELECT id, email, password_hash, balance, plan_id, expired_at,
		traffic_limit, traffic_used, status, created_at, updated_at
		FROM users WHERE email = ?`)
	var u User
	if err := s.DB.GetContext(ctx, &u, q, email); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) FindUserByID(ctx context.Context, id int64) (*User, error) {
	q := s.Rebind(`SELECT id, email, password_hash, balance, plan_id, expired_at,
		traffic_limit, traffic_used, status, created_at, updated_at
		FROM users WHERE id = ?`)
	var u User
	if err := s.DB.GetContext(ctx, &u, q, id); err != nil {
		return nil, err
	}
	return &u, nil
}

// ActivateUserPlan applies a plan: extends expiry by plan.DurationDays from
// max(now, current expiry) and resets traffic limit / used. The status is set
// to 'active'.
func (s *Store) ActivateUserPlan(ctx context.Context, userID int64, plan *Plan) error {
	u, err := s.FindUserByID(ctx, userID)
	if err != nil {
		return err
	}
	base := time.Now().UTC()
	if u.ExpiredAt != nil && u.ExpiredAt.After(base) {
		base = *u.ExpiredAt
	}
	newExpiry := base.AddDate(0, 0, plan.DurationDays)

	q := s.Rebind(`UPDATE users SET plan_id = ?, expired_at = ?, traffic_limit = ?,
		traffic_used = 0, status = 'active' WHERE id = ?`)
	_, err = s.DB.ExecContext(ctx, q, plan.ID, newExpiry, plan.TrafficLimit, userID)
	return err
}

// SetUserStatus changes the status field (e.g. 'disabled', 'active').
func (s *Store) SetUserStatus(ctx context.Context, userID int64, status string) error {
	q := s.Rebind(`UPDATE users SET status = ? WHERE id = ?`)
	_, err := s.DB.ExecContext(ctx, q, status, userID)
	return err
}

type AdminUpdateUserInput struct {
	Email        string
	Balance      string
	PlanID       *int64
	ExpiredAt    *time.Time
	TrafficLimit int64
	TrafficUsed  int64
	Status       string
}

func (s *Store) AdminUpdateUser(ctx context.Context, userID int64, in AdminUpdateUserInput) error {
	q := s.Rebind(`UPDATE users SET email = ?, balance = ?, plan_id = ?, expired_at = ?,
		traffic_limit = ?, traffic_used = ?, status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`)
	_, err := s.DB.ExecContext(ctx, q, in.Email, in.Balance, in.PlanID, in.ExpiredAt,
		in.TrafficLimit, in.TrafficUsed, in.Status, userID)
	return err
}

func (s *Store) ListUsers(ctx context.Context, limit int) ([]User, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := s.Rebind(`SELECT id, email, password_hash, balance, plan_id, expired_at,
		traffic_limit, traffic_used, status, created_at, updated_at
		FROM users ORDER BY id DESC LIMIT ?`)
	var rows []User
	if err := s.DB.SelectContext(ctx, &rows, q, limit); err != nil {
		return nil, err
	}
	return rows, nil
}
