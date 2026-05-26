package store

import (
	"context"
	"time"
)

// PaymentProvider is a DB-managed payment gateway configuration.
type PaymentProvider struct {
	ID           int64     `db:"id" json:"id"`
	Name         string    `db:"name" json:"name"`                   // unique slug: "epay", "stripe", "paypal"
	DisplayName  string    `db:"display_name" json:"display_name"`   // 显示名称
	ProviderType string    `db:"provider_type" json:"provider_type"` // "epay" | "stripe" | "paypal" | "nowpayments" | "creem"
	ConfigJSON   string    `db:"config_json" json:"config_json"`     // JSON blob with provider-specific keys
	Enabled      int       `db:"enabled" json:"enabled"`
	Sort         int       `db:"sort" json:"sort"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

func (s *Store) ListPaymentProviders(ctx context.Context) ([]PaymentProvider, error) {
	q := `SELECT id, name, display_name, provider_type, config_json, enabled, sort, created_at, updated_at
		FROM payment_providers ORDER BY sort ASC, id ASC`
	var rows []PaymentProvider
	if err := s.DB.SelectContext(ctx, &rows, q); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) ListEnabledPaymentProviders(ctx context.Context) ([]PaymentProvider, error) {
	q := `SELECT id, name, display_name, provider_type, config_json, enabled, sort, created_at, updated_at
		FROM payment_providers WHERE enabled = 1 ORDER BY sort ASC, id ASC`
	var rows []PaymentProvider
	if err := s.DB.SelectContext(ctx, &rows, q); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) FindPaymentProviderByName(ctx context.Context, name string) (*PaymentProvider, error) {
	q := s.Rebind(`SELECT id, name, display_name, provider_type, config_json, enabled, sort, created_at, updated_at
		FROM payment_providers WHERE name = ?`)
	var p PaymentProvider
	if err := s.DB.GetContext(ctx, &p, q, name); err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) FindPaymentProviderByID(ctx context.Context, id int64) (*PaymentProvider, error) {
	q := s.Rebind(`SELECT id, name, display_name, provider_type, config_json, enabled, sort, created_at, updated_at
		FROM payment_providers WHERE id = ?`)
	var p PaymentProvider
	if err := s.DB.GetContext(ctx, &p, q, id); err != nil {
		return nil, err
	}
	return &p, nil
}

type CreatePaymentProviderInput struct {
	Name         string
	DisplayName  string
	ProviderType string
	ConfigJSON   string
	Enabled      int
	Sort         int
}

func (s *Store) CreatePaymentProvider(ctx context.Context, in CreatePaymentProviderInput) (int64, error) {
	return s.InsertReturningID(ctx,
		`INSERT INTO payment_providers(name, display_name, provider_type, config_json, enabled, sort)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		in.Name, in.DisplayName, in.ProviderType, in.ConfigJSON, in.Enabled, in.Sort,
	)
}

func (s *Store) UpdatePaymentProvider(ctx context.Context, id int64, displayName, configJSON string, enabled, sort int) error {
	q := s.Rebind(`UPDATE payment_providers SET display_name = ?, config_json = ?, enabled = ?, sort = ? WHERE id = ?`)
	_, err := s.DB.ExecContext(ctx, q, displayName, configJSON, enabled, sort, id)
	return err
}

func (s *Store) DeletePaymentProvider(ctx context.Context, id int64) error {
	q := s.Rebind(`DELETE FROM payment_providers WHERE id = ?`)
	_, err := s.DB.ExecContext(ctx, q, id)
	return err
}
