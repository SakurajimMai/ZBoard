package store

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type User struct {
	ID           int64      `db:"id"`
	Email        string     `db:"email"`
	PasswordHash string     `db:"password_hash"`
	Balance      string     `db:"balance"`
	PlanID       *int64     `db:"plan_id"`
	PlanPeriod   string     `db:"plan_period"`
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
	PlanPeriod   string
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
	if strings.TrimSpace(in.PlanPeriod) == "" {
		in.PlanPeriod = BillingPeriodMonthly
	}
	return s.InsertReturningID(ctx,
		`INSERT INTO users(email, password_hash, balance, plan_id, plan_period, expired_at,
			traffic_limit, traffic_used, status)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		in.Email, in.PasswordHash, in.Balance, in.PlanID, in.PlanPeriod, in.ExpiredAt,
		in.TrafficLimit, in.TrafficUsed, in.Status,
	)
}

func (s *Store) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	q := s.Rebind(`SELECT id, email, password_hash, balance, plan_id, plan_period, expired_at,
		traffic_limit, traffic_used, status, created_at, updated_at
		FROM users WHERE email = ?`)
	var u User
	if err := s.DB.GetContext(ctx, &u, q, email); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) FindUserByID(ctx context.Context, id int64) (*User, error) {
	q := s.Rebind(`SELECT id, email, password_hash, balance, plan_id, plan_period, expired_at,
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
	return s.ActivateUserPlanPeriod(ctx, userID, plan, BillingPeriodMonthly)
}

func (s *Store) ActivateUserPlanPeriod(ctx context.Context, userID int64, plan *Plan, period string) error {
	u, err := s.FindUserByID(ctx, userID)
	if err != nil {
		return err
	}
	period = NormalizeBillingPeriod(period)
	base := time.Now().UTC()
	if u.PlanID != nil && *u.PlanID == plan.ID && u.ExpiredAt != nil && u.ExpiredAt.After(base) {
		base = *u.ExpiredAt
	}
	newExpiry := base.AddDate(0, 0, PlanDurationDays(plan, period))
	trafficLimit := PlanTrafficLimit(plan, period)

	q := s.Rebind(`UPDATE users SET plan_id = ?, plan_period = ?, expired_at = ?, traffic_limit = ?,
		traffic_used = 0, status = 'active' WHERE id = ?`)
	if _, err = s.DB.ExecContext(ctx, q, plan.ID, period, newExpiry, trafficLimit, userID); err != nil {
		return err
	}
	return s.ApplyPlanLimitsToNodeUsers(ctx, userID, plan)
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
	PlanPeriod   string
	ExpiredAt    *time.Time
	TrafficLimit int64
	TrafficUsed  int64
	Status       string
}

func (s *Store) AdminUpdateUser(ctx context.Context, userID int64, in AdminUpdateUserInput) error {
	if strings.TrimSpace(in.PlanPeriod) == "" {
		if existing, err := s.FindUserByID(ctx, userID); err == nil && strings.TrimSpace(existing.PlanPeriod) != "" {
			in.PlanPeriod = existing.PlanPeriod
		} else {
			in.PlanPeriod = BillingPeriodMonthly
		}
	}
	q := s.Rebind(`UPDATE users SET email = ?, balance = ?, plan_id = ?, expired_at = ?,
		plan_period = ?, traffic_limit = ?, traffic_used = ?, status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`)
	_, err := s.DB.ExecContext(ctx, q, in.Email, in.Balance, in.PlanID, in.ExpiredAt,
		in.PlanPeriod,
		in.TrafficLimit, in.TrafficUsed, in.Status, userID)
	return err
}

func (s *Store) ListUsers(ctx context.Context, limit int) ([]User, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, _, err := s.ListUsersPage(ctx, PageParams{Page: 1, PageSize: limit})
	return rows, err
}

type UserFilter struct {
	Email      string
	Status     string
	PlanID     *int64
	Expires    string
	TrafficMin *int64
	TrafficMax *int64
}

func (s *Store) ListUsersPage(ctx context.Context, p PageParams) ([]User, int64, error) {
	return s.ListUsersPageFiltered(ctx, p, UserFilter{})
}

func (s *Store) ListUsersPageFiltered(ctx context.Context, p PageParams, f UserFilter) ([]User, int64, error) {
	p = NormalizePage(p)
	where, args := userFilterWhere(f)
	var total int64
	if err := s.DB.GetContext(ctx, &total, s.Rebind(`SELECT COUNT(*) FROM users`+where), args...); err != nil {
		return nil, 0, err
	}
	args = append(args, p.PageSize, p.Offset())
	q := s.Rebind(`SELECT id, email, password_hash, balance, plan_id, plan_period, expired_at,
		traffic_limit, traffic_used, status, created_at, updated_at
		FROM users` + where + ` ORDER BY id DESC LIMIT ? OFFSET ?`)
	var rows []User
	if err := s.DB.SelectContext(ctx, &rows, q, args...); err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func userFilterWhere(f UserFilter) (string, []any) {
	parts := make([]string, 0, 6)
	args := make([]any, 0, 6)
	if email := strings.TrimSpace(strings.ToLower(f.Email)); email != "" {
		// LIKE wildcards (% _) inside the user-supplied email would otherwise
		// turn admin search into a full-table scan ("%") or pattern probe.
		// Escape with '!' (chosen because '\' is dialect-ambiguous between
		// MySQL string literals and SQL standard) and declare it explicitly.
		parts = append(parts, "LOWER(email) LIKE ? ESCAPE '!'")
		args = append(args, "%"+escapeLikePattern(email)+"%")
	}
	if status := strings.TrimSpace(f.Status); status != "" && status != "all" {
		parts = append(parts, "status = ?")
		args = append(args, status)
	}
	if f.PlanID != nil {
		parts = append(parts, "plan_id = ?")
		args = append(args, *f.PlanID)
	}
	switch strings.TrimSpace(f.Expires) {
	case "valid":
		parts = append(parts, "(expired_at IS NULL OR expired_at > ?)")
		args = append(args, Now())
	case "expired":
		parts = append(parts, "expired_at IS NOT NULL AND expired_at <= ?")
		args = append(args, Now())
	}
	if f.TrafficMin != nil {
		parts = append(parts, "traffic_used >= ?")
		args = append(args, *f.TrafficMin)
	}
	if f.TrafficMax != nil {
		parts = append(parts, "traffic_used <= ?")
		args = append(args, *f.TrafficMax)
	}
	if len(parts) == 0 {
		return "", args
	}
	return fmt.Sprintf(" WHERE %s", strings.Join(parts, " AND ")), args
}

// escapeLikePattern escapes the LIKE wildcards "%" and "_" plus the "!" escape
// character itself so user-supplied search input is treated as a literal. The
// SQL using this MUST declare ESCAPE '!' to match.
func escapeLikePattern(s string) string {
	r := strings.NewReplacer(
		"!", "!!",
		"%", "!%",
		"_", "!_",
	)
	return r.Replace(s)
}
