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

func TestRecordTrafficKeepsUserAndSnapshotInSync(t *testing.T) {
	s := testsupport.NewStore(t)
	ctx := context.Background()

	uid, err := s.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "traffic@example.com",
		PasswordHash: "hash",
		TrafficLimit: 10_000,
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("AdminCreateUser: %v", err)
	}
	nodeID, _, err := s.CreateNode(ctx, store.CreateNodeInput{
		Name:     "节点",
		Host:     "node.example.com",
		Port:     443,
		Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("CreateNode: %v", err)
	}
	if err := s.EnsureNodeUser(ctx, uid, nodeID, "client-id", "vless"); err != nil {
		t.Fatalf("EnsureNodeUser: %v", err)
	}

	if err := s.RecordTraffic(ctx, []store.TrafficDelta{{
		UserID:        uid,
		NodeID:        nodeID,
		UploadDelta:   100,
		DownloadDelta: 900,
	}}); err != nil {
		t.Fatalf("RecordTraffic: %v", err)
	}

	u, err := s.FindUserByID(ctx, uid)
	if err != nil {
		t.Fatalf("FindUserByID: %v", err)
	}
	if u.TrafficUsed != 1000 {
		t.Fatalf("traffic_used = %d, want 1000", u.TrafficUsed)
	}
	snaps, err := s.ListTrafficSnapshots(ctx, 10)
	if err != nil {
		t.Fatalf("ListTrafficSnapshots: %v", err)
	}
	if len(snaps) != 1 {
		t.Fatalf("snapshots len=%d, want 1", len(snaps))
	}
	if snaps[0].TotalUsed != u.TrafficUsed || snaps[0].TrafficLimit != u.TrafficLimit {
		t.Fatalf("snapshot not in sync with user: snap=%+v user=%+v", snaps[0], u)
	}
}
