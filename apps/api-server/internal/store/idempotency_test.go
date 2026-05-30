package store_test

import (
	"context"
	"sync"
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

// TestIdempotencyExpiredRecoverySingleOwner is the C6 regression: when a stored
// key has expired, a fresh claim may reclaim it — but only ONE caller may own
// the reclaimed row. A negative TTL makes the row expired the instant it's
// written, so the next claim must reclaim it and become the owner.
func TestIdempotencyExpiredRecoverySingleOwner(t *testing.T) {
	s := testsupport.NewStore(t)
	ctx := context.Background()

	// Insert an already-expired row (negative TTL => expires_at in the past).
	first, existing, err := s.ClaimIdempotency(ctx, "orders.create:1", "key-A", "hash-old", -time.Minute)
	if err != nil {
		t.Fatalf("seed expired claim: %v", err)
	}
	if first == nil || existing != nil {
		t.Fatalf("seed should be a fresh claim, got claimed=%v existing=%v", first, existing)
	}

	// A new caller with a new hash should reclaim it and become the owner.
	claimed, existing, err := s.ClaimIdempotency(ctx, "orders.create:1", "key-A", "hash-new", time.Hour)
	if err != nil {
		t.Fatalf("reclaim: %v", err)
	}
	if claimed == nil {
		t.Fatalf("expired key should be reclaimable, got existing=%v", existing)
	}
	if claimed.ID == 0 {
		t.Fatalf("reclaimed owner must carry a real row ID (CompleteIdempotency needs it)")
	}
	if claimed.RequestHash == nil || *claimed.RequestHash != "hash-new" {
		t.Fatalf("reclaimed row should carry the new hash, got %+v", claimed)
	}

	// The reclaimed row must now be live: a follow-up claim with a DIFFERENT
	// hash must be treated as a conflict (existing != nil), NOT a second owner.
	c2, e2, err := s.ClaimIdempotency(ctx, "orders.create:1", "key-A", "hash-other", time.Hour)
	if err != nil {
		t.Fatalf("post-reclaim claim: %v", err)
	}
	if c2 != nil {
		t.Fatalf("only one owner allowed; second caller must get existing, got claimed=%+v", c2)
	}
	if e2 == nil || *e2.RequestHash != "hash-new" {
		t.Fatalf("second caller should see the reclaimed owner's record, got %+v", e2)
	}
}

// TestIdempotencyConcurrentClaimSingleOwner is the core C6 invariant under load:
// many goroutines racing on the SAME fresh key must yield EXACTLY ONE owner.
// Before the fix, the expired-recovery path had a TOCTOU that let two callers
// both believe they owned the key and both create the underlying resource.
func TestIdempotencyConcurrentClaimSingleOwner(t *testing.T) {
	s := testsupport.NewStore(t)
	ctx := context.Background()

	const n = 16
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		owners   int
		existers int
		errs     int
	)
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			claimed, existing, err := s.ClaimIdempotency(ctx, "payments.start:7", "race-key", "h", time.Hour)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err != nil:
				errs++
			case claimed != nil:
				owners++
			case existing != nil:
				existers++
			}
		}()
	}
	close(start)
	wg.Wait()

	if errs != 0 {
		t.Fatalf("unexpected errors during concurrent claim: %d", errs)
	}
	if owners != 1 {
		t.Fatalf("exactly one owner required, got %d owners + %d existers", owners, existers)
	}
	if owners+existers != n {
		t.Fatalf("every caller must resolve to owner-or-existing: %d+%d != %d", owners, existers, n)
	}
}
