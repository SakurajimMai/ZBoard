package store

import (
	"context"
	"time"
)

// IdempotencyRecord is the persisted state for a `Idempotency-Key` request.
type IdempotencyRecord struct {
	ID           int64      `db:"id"`
	KeyValue     string     `db:"key_value"`
	Scope        string     `db:"scope"`
	RequestHash  *string    `db:"request_hash"`
	ResponseBody *string    `db:"response_body"`
	Status       string     `db:"status"`
	ExpiresAt    time.Time  `db:"expires_at"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
}

// FindIdempotency returns the existing record or nil if not found.
func (s *Store) FindIdempotency(ctx context.Context, scope, key string) (*IdempotencyRecord, error) {
	q := s.Rebind(`SELECT id, key_value, scope, request_hash, response_body, status,
		expires_at, created_at, updated_at FROM idempotency_keys
		WHERE scope = ? AND key_value = ?`)
	var r IdempotencyRecord
	if err := s.DB.GetContext(ctx, &r, q, scope, key); err != nil {
		return nil, err
	}
	return &r, nil
}

// ClaimIdempotency tries to insert a `processing` row. If a row already exists,
// returns the existing record without modifying it.
func (s *Store) ClaimIdempotency(ctx context.Context, scope, key, requestHash string, ttl time.Duration) (claimed *IdempotencyRecord, existing *IdempotencyRecord, err error) {
	expires := time.Now().UTC().Add(ttl)
	id, err := s.InsertReturningID(ctx,
		`INSERT INTO idempotency_keys(key_value, scope, request_hash, status, expires_at)
		 VALUES (?, ?, ?, 'processing', ?)`,
		key, scope, requestHash, expires,
	)
	if err != nil {
		if IsUniqueViolation(err) {
			rec, ferr := s.FindIdempotency(ctx, scope, key)
			if ferr != nil {
				return nil, nil, ferr
			}
			return nil, rec, nil
		}
		return nil, nil, err
	}
	return &IdempotencyRecord{
		ID:          id,
		KeyValue:    key,
		Scope:       scope,
		RequestHash: &requestHash,
		Status:      "processing",
		ExpiresAt:   expires,
	}, nil, nil
}

// CompleteIdempotency stores the rendered response body and marks status.
func (s *Store) CompleteIdempotency(ctx context.Context, id int64, responseBody string, status string) error {
	q := s.Rebind(`UPDATE idempotency_keys SET response_body = ?, status = ? WHERE id = ?`)
	_, err := s.DB.ExecContext(ctx, q, responseBody, status, id)
	return err
}
