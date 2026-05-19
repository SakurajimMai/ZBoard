package store_test

import (
	"context"
	"testing"

	"github.com/zboard/api-server/internal/store"
	"github.com/zboard/api-server/internal/testsupport"
)

func TestUserCreateAndFind(t *testing.T) {
	s := testsupport.NewStore(t)
	ctx := context.Background()

	id, err := s.CreateUser(ctx, "a@example.com", "hash1")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if id == 0 {
		t.Fatalf("expected non-zero id")
	}
	got, err := s.FindUserByEmail(ctx, "a@example.com")
	if err != nil {
		t.Fatalf("FindUserByEmail: %v", err)
	}
	if got.ID != id || got.PasswordHash != "hash1" {
		t.Fatalf("unexpected user: %+v", got)
	}

	// Duplicate email -> unique violation
	if _, err := s.CreateUser(ctx, "a@example.com", "hash2"); err == nil {
		t.Fatalf("expected duplicate email error")
	} else if !store.IsUniqueViolation(err) {
		t.Fatalf("expected unique violation, got %v", err)
	}
}

func TestActivateUserPlanExtendsExpiry(t *testing.T) {
	s := testsupport.NewStore(t)
	ctx := context.Background()
	uid, err := s.CreateUser(ctx, "p@example.com", "h")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	planID, err := s.CreatePlan(ctx, store.CreatePlanInput{
		Name: "P1", Price: "9.90", DurationDays: 30, TrafficLimit: 1024,
	})
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}
	plan, err := s.FindPlanByID(ctx, planID)
	if err != nil {
		t.Fatalf("FindPlanByID: %v", err)
	}
	if err := s.ActivateUserPlan(ctx, uid, plan); err != nil {
		t.Fatalf("ActivateUserPlan: %v", err)
	}
	u, err := s.FindUserByID(ctx, uid)
	if err != nil {
		t.Fatalf("FindUserByID: %v", err)
	}
	if u.PlanID == nil || *u.PlanID != planID {
		t.Fatalf("plan_id not set: %+v", u)
	}
	if u.TrafficLimit != 1024 {
		t.Fatalf("traffic_limit not set: %d", u.TrafficLimit)
	}
	if u.ExpiredAt == nil {
		t.Fatalf("expired_at not set")
	}
	if u.Status != "active" {
		t.Fatalf("status should be active, got %s", u.Status)
	}
}
