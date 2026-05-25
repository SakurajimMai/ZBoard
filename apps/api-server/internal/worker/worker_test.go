package worker_test

import (
	"context"
	"testing"
	"time"

	"github.com/zboard/api-server/internal/store"
	"github.com/zboard/api-server/internal/testsupport"
	"github.com/zboard/api-server/internal/worker"
)

func TestRunDisablesExpiredAndOverQuota(t *testing.T) {
	s := testsupport.NewStore(t)
	wk := worker.New(s)
	ctx := context.Background()

	// Plan + two users + two nodes; provision node_users for both.
	planID, err := s.CreatePlan(ctx, store.CreatePlanInput{
		Name: "P", Price: "1", DurationDays: 30, TrafficLimit: 100,
	})
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}
	plan, _ := s.FindPlanByID(ctx, planID)

	uExpired, _ := s.CreateUser(ctx, "expired@example.com", "h")
	uOver, _ := s.CreateUser(ctx, "over@example.com", "h")
	if err := s.ActivateUserPlan(ctx, uExpired, plan); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if err := s.ActivateUserPlan(ctx, uOver, plan); err != nil {
		t.Fatalf("activate: %v", err)
	}

	nodeID1, _, err := s.CreateNode(ctx, store.CreateNodeInput{
		Name: "N1", Host: "h1", Port: 443, Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("CreateNode: %v", err)
	}
	nodeID2, _, err := s.CreateNode(ctx, store.CreateNodeInput{
		Name: "N2", Host: "h2", Port: 443, Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("CreateNode: %v", err)
	}
	for _, uid := range []int64{uExpired, uOver} {
		for _, nid := range []int64{nodeID1, nodeID2} {
			if err := s.EnsureNodeUser(ctx, uid, nid, "cid", "vless"); err != nil {
				t.Fatalf("EnsureNodeUser: %v", err)
			}
		}
	}

	// Age uExpired's expiry into the past (raw SQL via the underlying connection).
	// Use UTC so the worker's UTC-based query compares apples to apples; SQLite
	// stores DATETIME as TEXT and string-comparison is timezone-sensitive.
	if _, err := s.DB.ExecContext(ctx,
		s.Rebind(`UPDATE users SET expired_at = ? WHERE id = ?`),
		time.Now().UTC().Add(-time.Hour), uExpired); err != nil {
		t.Fatalf("age expiry: %v", err)
	}
	// Push uOver over their traffic limit.
	if _, err := s.DB.ExecContext(ctx,
		s.Rebind(`UPDATE users SET traffic_used = traffic_limit + 1 WHERE id = ?`),
		uOver); err != nil {
		t.Fatalf("over-quota: %v", err)
	}

	res, err := wk.Run(ctx)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.ExpiredUsers != 1 || res.OverQuotaUsers != 1 {
		t.Fatalf("counts wrong: %+v", res)
	}
	if res.DisableTasks != 4 { // 2 users × 2 nodes
		t.Fatalf("expected 4 disable_user tasks, got %d", res.DisableTasks)
	}
	if res.DisabledUsers != 2 {
		t.Fatalf("expected 2 disabled users, got %d", res.DisabledUsers)
	}

	// Re-run is a no-op.
	res2, err := wk.Run(ctx)
	if err != nil {
		t.Fatalf("Run #2: %v", err)
	}
	if res2.ExpiredUsers != 0 || res2.OverQuotaUsers != 0 || res2.DisableTasks != 0 {
		t.Fatalf("second run not idempotent: %+v", res2)
	}
}

func TestSweepTimeoutTasks(t *testing.T) {
	s := testsupport.NewStore(t)
	wk := worker.New(s)
	ctx := context.Background()

	nodeID, _, err := s.CreateNode(ctx, store.CreateNodeInput{
		Name: "N", Host: "h", Port: 443, Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("CreateNode: %v", err)
	}
	if err := s.CreateNodeTask(ctx, "t-stale", nodeID, "sync_config", "{}"); err != nil {
		t.Fatalf("CreateNodeTask: %v", err)
	}
	// Pull moves it to running with current locked_at.
	if _, err := s.PullTasksForNode(ctx, nodeID, 10); err != nil {
		t.Fatalf("PullTasksForNode: %v", err)
	}
	// Backdate lock so the sweep matches. Use UTC for the same timezone reason
	// as TestRunDisablesExpiredAndOverQuota.
	if _, err := s.DB.ExecContext(ctx,
		s.Rebind(`UPDATE node_tasks SET locked_at = ? WHERE task_id = ?`),
		time.Now().UTC().Add(-time.Hour), "t-stale"); err != nil {
		t.Fatalf("backdate: %v", err)
	}

	res, err := wk.Run(ctx)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.TimedOutTasks != 1 {
		t.Fatalf("expected 1 timed-out task, got %d", res.TimedOutTasks)
	}
}

func TestRunCreatesSubscriptionReminderNotifications(t *testing.T) {
	s := testsupport.NewStore(t)
	wk := worker.New(s)
	ctx := context.Background()
	if err := s.SetSettings(ctx, map[string]string{
		"subscription_expire_reminder_enabled": "1",
		"traffic_alert_enabled":                "1",
		"reminder_days_before":                 "3",
		"traffic_alert_threshold":              "80",
	}); err != nil {
		t.Fatalf("set settings: %v", err)
	}
	planID, err := s.CreatePlan(ctx, store.CreatePlanInput{
		Name: "P", Price: "1", DurationDays: 30, TrafficLimit: 100,
	})
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}
	plan, _ := s.FindPlanByID(ctx, planID)
	uExpiring, _ := s.CreateUser(ctx, "expiring@example.com", "h")
	uTraffic, _ := s.CreateUser(ctx, "traffic@example.com", "h")
	if err := s.ActivateUserPlan(ctx, uExpiring, plan); err != nil {
		t.Fatalf("activate expiring: %v", err)
	}
	if err := s.ActivateUserPlan(ctx, uTraffic, plan); err != nil {
		t.Fatalf("activate traffic: %v", err)
	}
	if _, err := s.DB.ExecContext(ctx,
		s.Rebind(`UPDATE users SET expired_at = ? WHERE id = ?`),
		time.Now().UTC().Add(48*time.Hour), uExpiring); err != nil {
		t.Fatalf("set expiring: %v", err)
	}
	if _, err := s.DB.ExecContext(ctx,
		s.Rebind(`UPDATE users SET traffic_used = 85 WHERE id = ?`),
		uTraffic); err != nil {
		t.Fatalf("set traffic: %v", err)
	}

	res, err := wk.Run(ctx)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.ExpiryReminders != 1 || res.TrafficReminders != 1 {
		t.Fatalf("reminder counts wrong: %+v", res)
	}
	expiringNotifications, err := s.ListNotifications(ctx, uExpiring, 10)
	if err != nil {
		t.Fatalf("list expiring notifications: %v", err)
	}
	if len(expiringNotifications) != 1 || expiringNotifications[0].Type != "plan_expiring" {
		t.Fatalf("unexpected expiring notifications: %+v", expiringNotifications)
	}
	trafficNotifications, err := s.ListNotifications(ctx, uTraffic, 10)
	if err != nil {
		t.Fatalf("list traffic notifications: %v", err)
	}
	if len(trafficNotifications) != 1 || trafficNotifications[0].Type != "traffic_alert" {
		t.Fatalf("unexpected traffic notifications: %+v", trafficNotifications)
	}

	res, err = wk.Run(ctx)
	if err != nil {
		t.Fatalf("Run #2: %v", err)
	}
	if res.ExpiryReminders != 0 || res.TrafficReminders != 0 {
		t.Fatalf("second run should not duplicate reminders: %+v", res)
	}
}
