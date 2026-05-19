package store

import (
	"context"
	"time"
)

type AdminUser struct {
	ID               int64      `db:"id"`
	Email            string     `db:"email"`
	PasswordHash     string     `db:"password_hash"`
	Role             string     `db:"role"`
	TwoFactorEnabled int        `db:"two_factor_enabled"`
	TwoFactorSecret  *string    `db:"two_factor_secret"`
	Status           string     `db:"status"`
	LastLoginAt      *time.Time `db:"last_login_at"`
	CreatedAt        time.Time  `db:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at"`
}

func (s *Store) CountAdmins(ctx context.Context) (int64, error) {
	var n int64
	if err := s.DB.GetContext(ctx, &n, `SELECT COUNT(*) FROM admin_users`); err != nil {
		return 0, err
	}
	return n, nil
}

func (s *Store) CreateAdmin(ctx context.Context, email, passwordHash, role string) (int64, error) {
	return s.InsertReturningID(ctx,
		`INSERT INTO admin_users(email, password_hash, role) VALUES (?, ?, ?)`,
		email, passwordHash, role,
	)
}

func (s *Store) FindAdminByEmail(ctx context.Context, email string) (*AdminUser, error) {
	q := s.Rebind(`SELECT id, email, password_hash, role, two_factor_enabled,
		two_factor_secret, status, last_login_at, created_at, updated_at
		FROM admin_users WHERE email = ?`)
	var a AdminUser
	if err := s.DB.GetContext(ctx, &a, q, email); err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Store) FindAdminByID(ctx context.Context, id int64) (*AdminUser, error) {
	q := s.Rebind(`SELECT id, email, password_hash, role, two_factor_enabled,
		two_factor_secret, status, last_login_at, created_at, updated_at
		FROM admin_users WHERE id = ?`)
	var a AdminUser
	if err := s.DB.GetContext(ctx, &a, q, id); err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Store) TouchAdminLogin(ctx context.Context, id int64) error {
	q := s.Rebind(`UPDATE admin_users SET last_login_at = ? WHERE id = ?`)
	_, err := s.DB.ExecContext(ctx, q, Now(), id)
	return err
}
