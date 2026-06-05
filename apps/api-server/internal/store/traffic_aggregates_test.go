package store_test

import (
	"context"
	"testing"

	"github.com/zboard/api-server/internal/store"
	"github.com/zboard/api-server/internal/testsupport"
)

func TestAdminTrafficTrendOnlyUsesDatedTrafficLogs(t *testing.T) {
	s := testsupport.NewStore(t)
	ctx := context.Background()

	used := int64(5 * 1024 * 1024 * 1024)
	_, err := s.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "traffic-fallback@example.com",
		PasswordHash: "hash",
		TrafficLimit: 10 * used,
		TrafficUsed:  used,
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("AdminCreateUser: %v", err)
	}

	points, err := s.AdminTrafficTrend(ctx, 7)
	if err != nil {
		t.Fatalf("AdminTrafficTrend: %v", err)
	}
	if len(points) != 7 {
		t.Fatalf("points len=%d, want 7", len(points))
	}
	for _, p := range points {
		if p.Total != 0 {
			t.Fatalf("trend must only use dated traffic logs, got non-zero point %+v", p)
		}
	}
}

func TestAdminTrafficTrendDoesNotTreatUserCycleTotalAsDailyTraffic(t *testing.T) {
	s := testsupport.NewStore(t)
	ctx := context.Background()

	uid, err := s.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "cycle-total-is-not-daily@example.com",
		PasswordHash: "hash",
		TrafficLimit: 100 * 1024 * 1024 * 1024,
		TrafficUsed:  57 * 1024 * 1024 * 1024,
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("AdminCreateUser: %v", err)
	}
	for i := 0; i < 4; i++ {
		nodeID, _, err := s.CreateNode(ctx, store.CreateNodeInput{
			Name:     "node-cycle-total",
			Host:     "node-cycle-total.example.com",
			Port:     443 + i,
			Protocol: "vless",
		})
		if err != nil {
			t.Fatalf("CreateNode: %v", err)
		}
		if err := s.EnsureNodeUser(ctx, uid, nodeID, "11111111-1111-4111-8111-111111111111", "vless"); err != nil {
			t.Fatalf("EnsureNodeUser: %v", err)
		}
		if _, err := s.DB.ExecContext(ctx, s.Rebind(`UPDATE node_users SET upload = ?, download = ? WHERE user_id = ? AND node_id = ?`),
			int64(10*1024*1024), int64(200*1024*1024), uid, nodeID); err != nil {
			t.Fatalf("seed node user traffic: %v", err)
		}
	}

	points, err := s.AdminTrafficTrend(ctx, 7)
	if err != nil {
		t.Fatalf("AdminTrafficTrend: %v", err)
	}
	for _, p := range points {
		if p.Total != 0 {
			t.Fatalf("trend must only use dated traffic logs, got non-zero point %+v", p)
		}
	}
}

func TestDailyTrafficDoesNotTreatUserCycleTotalAsTodayTraffic(t *testing.T) {
	s := testsupport.NewStore(t)
	ctx := context.Background()

	uid, err := s.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "daily-cycle-total@example.com",
		PasswordHash: "hash",
		TrafficLimit: 100 * 1024 * 1024 * 1024,
		TrafficUsed:  57 * 1024 * 1024 * 1024,
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("AdminCreateUser: %v", err)
	}

	points, err := s.ListDailyTrafficByUser(ctx, uid, 30)
	if err != nil {
		t.Fatalf("ListDailyTrafficByUser: %v", err)
	}
	if len(points) != 0 {
		t.Fatalf("daily traffic must only use dated traffic logs, got %+v", points)
	}
}

func TestNodeViewsUseNodeUserTrafficTotalsWhenLogsAreMissing(t *testing.T) {
	s := testsupport.NewStore(t)
	ctx := context.Background()

	uid, err := s.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "node-traffic-fallback@example.com",
		PasswordHash: "hash",
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("AdminCreateUser: %v", err)
	}
	nodeID, _, err := s.CreateNode(ctx, store.CreateNodeInput{
		Name:     "fallback-node",
		Host:     "fallback.example.com",
		Port:     443,
		Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("CreateNode: %v", err)
	}
	if err := s.EnsureNodeUser(ctx, uid, nodeID, "11111111-1111-4111-8111-111111111111", "vless"); err != nil {
		t.Fatalf("EnsureNodeUser: %v", err)
	}
	if _, err := s.DB.ExecContext(ctx, s.Rebind(`UPDATE node_users SET upload = ?, download = ? WHERE user_id = ? AND node_id = ?`),
		int64(1024), int64(4096), uid, nodeID); err != nil {
		t.Fatalf("seed node user traffic: %v", err)
	}

	nodes, _, err := s.ListAllNodeViewsPage(ctx, store.PageParams{Page: 1, PageSize: 10}, 120)
	if err != nil {
		t.Fatalf("ListAllNodeViewsPage: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("nodes len=%d, want 1", len(nodes))
	}
	if nodes[0].UploadTotal != 1024 || nodes[0].DownloadTotal != 4096 || nodes[0].TrafficTotal != 5120 {
		t.Fatalf("node traffic totals not from node_users: %+v", nodes[0])
	}
}
