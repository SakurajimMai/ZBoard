package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/zboard/api-server/internal/testsupport"
)

func TestIdempotencyClaimAndReplay(t *testing.T) {
	s := testsupport.NewStore(t)
	ctx := context.Background()

	claimed, existing, err := s.ClaimIdempotency(ctx, "scope1", "k1", "hash1", time.Hour)
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}
	if existing != nil || claimed == nil {
		t.Fatalf("expected fresh claim, got existing=%v claimed=%v", existing, claimed)
	}
	if err := s.CompleteIdempotency(ctx, claimed.ID, `{"hi":1}`, "succeeded"); err != nil {
		t.Fatalf("complete: %v", err)
	}

	// Same key + same hash -> existing record returned with response body.
	c2, e2, err := s.ClaimIdempotency(ctx, "scope1", "k1", "hash1", time.Hour)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if c2 != nil {
		t.Fatalf("expected no new claim on replay, got %+v", c2)
	}
	if e2 == nil || e2.Status != "succeeded" || e2.ResponseBody == nil || *e2.ResponseBody != `{"hi":1}` {
		t.Fatalf("expected stored response, got %+v", e2)
	}

	// Same key + different hash still returns the stored record; the caller
	// is responsible for rejecting on hash mismatch.
	_, e3, err := s.ClaimIdempotency(ctx, "scope1", "k1", "different-hash", time.Hour)
	if err != nil {
		t.Fatalf("hash-mismatch claim: %v", err)
	}
	if e3 == nil || *e3.RequestHash != "hash1" {
		t.Fatalf("expected original hash, got %+v", e3)
	}
}
