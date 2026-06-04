package store_test

import (
	"context"
	"testing"

	"github.com/zboard/api-server/internal/store"
	"github.com/zboard/api-server/internal/testsupport"
)

func TestAdminTrafficTrendFallsBackToUserTotalsWhenLogsAreMissing(t *testing.T) {
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
	if points[6].Total != used {
		t.Fatalf("latest traffic total=%d, want fallback user traffic_used=%d; points=%+v", points[6].Total, used, points)
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
