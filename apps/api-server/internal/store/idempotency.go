package store

import (
	"context"
	"database/sql"
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

// FindIdempotency returns the existing record or nil if not found. Records past
// their expires_at are treated as if absent — callers must claim a fresh row.
func (s *Store) FindIdempotency(ctx context.Context, scope, key string) (*IdempotencyRecord, error) {
	q := s.Rebind(`SELECT id, key_value, scope, request_hash, response_body, status,
		expires_at, created_at, updated_at FROM idempotency_keys
		WHERE scope = ? AND key_value = ? AND expires_at > ?`)
	var r IdempotencyRecord
	if err := s.DB.GetContext(ctx, &r, q, scope, key, time.Now().UTC()); err != nil {
		return nil, err
	}
	return &r, nil
}

// ClaimIdempotency tries to insert a `processing` row. If a row already exists
// AND has not expired, returns the existing record without modifying it. An
// expired duplicate is atomically reclaimed: exactly one concurrent caller wins
// the conditional UPDATE (its RowsAffected==1) and becomes the new owner; every
// other caller falls back to reading the winner's row and is returned it as an
// `existing` record, so the upper layer rejects/replays on request_hash just as
// it would for a live duplicate. This prevents the TOCTOU where two callers both
// saw the row as expired and both proceeded to create the underlying resource.
func (s *Store) ClaimIdempotency(ctx context.Context, scope, key, requestHash string, ttl time.Duration) (claimed *IdempotencyRecord, existing *IdempotencyRecord, err error) {
	expires := time.Now().UTC().Add(ttl)
	id, err := s.InsertReturningID(ctx,
		`INSERT INTO idempotency_keys(key_value, scope, request_hash, status, expires_at)
		 VALUES (?, ?, ?, 'processing', ?)`,
		key, scope, requestHash, expires,
	)
	if err == nil {
		return &IdempotencyRecord{
			ID:          id,
			KeyValue:    key,
			Scope:       scope,
			RequestHash: &requestHash,
			Status:      "processing",
			ExpiresAt:   expires,
		}, nil, nil
	}
	if !IsUniqueViolation(err) {
		return nil, nil, err
	}

	// A row already exists. If it's still live, hand it back as existing.
	if rec, ferr := s.FindIdempotency(ctx, scope, key); ferr == nil {
		return nil, rec, nil
	} else if !IsNoRows(ferr) {
		return nil, nil, ferr
	}

	// The existing row is expired. Race to reclaim it: the UPDATE is guarded by
	// `expires_at <= now`, so only the first caller flips it; concurrent callers
	// affect 0 rows. The winner resets the body/hash and takes ownership.
	now := time.Now().UTC()
	upd := s.Rebind(`UPDATE idempotency_keys
		SET request_hash = ?, response_body = NULL, status = 'processing', expires_at = ?
		WHERE scope = ? AND key_value = ? AND expires_at <= ?`)
	res, uerr := s.DB.ExecContext(ctx, upd, requestHash, expires, scope, key, now)
	if uerr != nil {
		return nil, nil, uerr
	}
	affected, uerr := res.RowsAffected()
	if uerr != nil {
		return nil, nil, uerr
	}
	if affected > 0 {
		// We won the reclaim. Re-read the row so the caller gets the canonical
		// record *including its ID* — bizsvc later calls CompleteIdempotency(id)
		// to store the response body, so a zero ID would silently lose it.
		rec, ferr := s.FindIdempotency(ctx, scope, key)
		if ferr != nil {
			return nil, nil, ferr
		}
		return rec, nil, nil
	}

	// We lost the race: another caller reclaimed the row first. Return their
	// (now non-expired) row as existing so the caller treats this as a duplicate.
	rec, ferr := s.FindIdempotency(ctx, scope, key)
	if ferr != nil {
		if IsNoRows(ferr) {
			// Extremely unlikely (row vanished). Surface as a generic conflict.
			return nil, nil, sql.ErrNoRows
		}
		return nil, nil, ferr
	}
	return nil, rec, nil
}

// CompleteIdempotency stores the rendered response body and marks status.
func (s *Store) CompleteIdempotency(ctx context.Context, id int64, responseBody string, status string) error {
	q := s.Rebind(`UPDATE idempotency_keys SET response_body = ?, status = ? WHERE id = ?`)
	_, err := s.DB.ExecContext(ctx, q, responseBody, status, id)
	return err
}
