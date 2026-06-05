package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/zboard/api-server/internal/store"
	"github.com/zboard/api-server/internal/testsupport"
)

func TestListActiveUsersByDayUsesPositiveTrafficAndDeduplicatesUsers(t *testing.T) {
	s := testsupport.NewStore(t)
	ctx := context.Background()

	planID, err := s.CreatePlan(ctx, store.CreatePlanInput{
		Name:         "active-users-plan",
		Price:        "9.90",
		DurationDays: 30,
		TrafficLimit: 1000,
		DeviceLimit:  2,
	})
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}
	activeUserID, err := s.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "real-active@example.com",
		PasswordHash: "hash",
		PlanID:       &planID,
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("create active user: %v", err)
	}
	zeroUserID, err := s.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "zero-traffic@example.com",
		PasswordHash: "hash",
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("create zero user: %v", err)
	}
	previousDayUserID, err := s.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "previous-day@example.com",
		PasswordHash: "hash",
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("create previous day user: %v", err)
	}
	nodeID, _, err := s.CreateNode(ctx, store.CreateNodeInput{
		Name:     "active-node",
		Host:     "active.example.com",
		Port:     443,
		Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("CreateNode: %v", err)
	}

	day := time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC)
	insertTrafficLog(t, s, activeUserID, nodeID, 100, 900, day.Add(8*time.Hour))
	insertTrafficLog(t, s, activeUserID, nodeID, 50, 150, day.Add(12*time.Hour))
	insertTrafficLog(t, s, zeroUserID, nodeID, 0, 0, day.Add(9*time.Hour))
	insertTrafficLog(t, s, previousDayUserID, nodeID, 500, 500, day.AddDate(0, 0, -1).Add(10*time.Hour))

	items, total, summary, err := s.ListActiveUsersByDay(ctx, day, store.PageParams{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListActiveUsersByDay: %v", err)
	}
	if total != 1 || summary.ActiveUsers != 1 {
		t.Fatalf("active user count total=%d summary=%d, want 1", total, summary.ActiveUsers)
	}
	if summary.UploadTotal != 150 || summary.DownloadTotal != 1050 || summary.TrafficTotal != 1200 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if len(items) != 1 {
		t.Fatalf("items len=%d, want 1: %+v", len(items), items)
	}
	item := items[0]
	if item.UserID != activeUserID || item.Email != "real-active@example.com" {
		t.Fatalf("unexpected active user item: %+v", item)
	}
	if item.PlanID == nil || *item.PlanID != planID {
		t.Fatalf("plan id not included: %+v", item)
	}
	if item.UploadTotal != 150 || item.DownloadTotal != 1050 || item.TrafficTotal != 1200 || item.NodeCount != 1 {
		t.Fatalf("unexpected item traffic: %+v", item)
	}
	if item.FirstActiveAt == "" || item.LastActiveAt == "" {
		t.Fatalf("active time range should be populated: %+v", item)
	}
}

func insertTrafficLog(t *testing.T, s *store.Store, userID, nodeID, upload, download int64, reportedAt time.Time) {
	t.Helper()
	total := upload + download
	_, err := s.DB.ExecContext(context.Background(), s.Rebind(`INSERT INTO traffic_logs(user_id, node_id, upload_delta, download_delta, total_delta, reported_at)
		VALUES (?, ?, ?, ?, ?, ?)`), userID, nodeID, upload, download, total, reportedAt)
	if err != nil {
		t.Fatalf("insert traffic log: %v", err)
	}
}
