package store

import (
	"context"
	"time"
)

type Payment struct {
	ID              int64      `db:"id" json:"id"`
	PaymentNo       string     `db:"payment_no" json:"payment_no"`
	OrderID         int64     `db:"order_id" json:"order_id"`
	UserID          int64      `db:"user_id" json:"user_id"`
	Provider        string     `db:"provider" json:"provider"`
	ProviderTradeNo *string    `db:"provider_trade_no" json:"provider_trade_no"`
	Amount          string     `db:"amount" json:"amount"`
	Status          string     `db:"status" json:"status"`
	PaidAt          *time.Time `db:"paid_at" json:"paid_at"`
	RawPayload      *string    `db:"raw_payload" json:"-"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
}

func (s *Store) CreatePayment(ctx context.Context, p *Payment) (int64, error) {
	return s.InsertReturningID(ctx,
		`INSERT INTO payments(payment_no, order_id, user_id, provider, amount, status)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		p.PaymentNo, p.OrderID, p.UserID, p.Provider, p.Amount, p.Status,
	)
}

func (s *Store) MarkPaymentSuccess(ctx context.Context, paymentNo, providerTradeNo string, paidAt time.Time) error {
	q := s.Rebind(`UPDATE payments SET status = 'success', provider_trade_no = ?, paid_at = ?
		WHERE payment_no = ? AND status <> 'success'`)
	_, err := s.DB.ExecContext(ctx, q, providerTradeNo, paidAt, paymentNo)
	return err
}

func (s *Store) ListPayments(ctx context.Context, limit int) ([]Payment, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := s.Rebind(`SELECT id, payment_no, order_id, user_id, provider, provider_trade_no,
		amount, status, paid_at, raw_payload, created_at, updated_at
		FROM payments ORDER BY id DESC LIMIT ?`)
	var rows []Payment
	if err := s.DB.SelectContext(ctx, &rows, q, limit); err != nil {
		return nil, err
	}
	return rows, nil
}

type PaymentCallback struct {
	ID              int64   `db:"id" json:"id"`
	Provider        string  `db:"provider" json:"provider"`
	ProviderEventID *string `db:"provider_event_id" json:"provider_event_id"`
	OrderNo         *string `db:"order_no" json:"order_no"`
	SignatureValid  int     `db:"signature_valid" json:"signature_valid"`
	Processed       int     `db:"processed" json:"processed"`
	ProcessedAt     *string `db:"processed_at" json:"processed_at"`
	RawHeaders      *string `db:"raw_headers" json:"-"`
	RawBody         *string `db:"raw_body" json:"-"`
	ErrorMessage    *string `db:"error_message" json:"error_message"`
	CreatedAt       string  `db:"created_at" json:"created_at"`
}

// CreatePaymentCallback inserts a callback record. Returns (id, alreadyExists, err).
// alreadyExists is true when (provider, provider_event_id) collide.
func (s *Store) CreatePaymentCallback(ctx context.Context, provider, eventID, orderNo, headers, body string) (int64, bool, error) {
	id, err := s.InsertReturningID(ctx,
		`INSERT INTO payment_callbacks(provider, provider_event_id, order_no, raw_headers, raw_body)
		 VALUES (?, ?, ?, ?, ?)`,
		provider, eventID, orderNo, headers, body,
	)
	if err != nil {
		if IsUniqueViolation(err) {
			return 0, true, nil
		}
		return 0, false, err
	}
	return id, false, nil
}

func (s *Store) MarkCallbackProcessed(ctx context.Context, id int64, errMsg string) error {
	q := s.Rebind(`UPDATE payment_callbacks SET processed = 1, processed_at = ?, error_message = ? WHERE id = ?`)
	_, err := s.DB.ExecContext(ctx, q, Now(), nullIfEmpty(errMsg), id)
	return err
}

func (s *Store) ListPaymentCallbacks(ctx context.Context, limit int) ([]PaymentCallback, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := s.Rebind(`SELECT id, provider, provider_event_id, order_no, signature_valid, processed,
		processed_at, raw_headers, raw_body, error_message, created_at
		FROM payment_callbacks ORDER BY id DESC LIMIT ?`)
	var rows []PaymentCallback
	if err := s.DB.SelectContext(ctx, &rows, q, limit); err != nil {
		return nil, err
	}
	return rows, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
