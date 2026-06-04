package server

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/zboard/api-server/internal/authsvc"
	"github.com/zboard/api-server/internal/store"
)

func TestUserTrafficSnapshotFallsBackToUserTotalsWhenSnapshotMissing(t *testing.T) {
	r, st, _ := setupAdminCRUDRouter(t)
	ctx := context.Background()
	auth := authsvc.New(st, "setup-token", nil)

	userID, err := auth.RegisterUser(ctx, "snapshot-fallback@example.com", "secret123")
	if err != nil {
		t.Fatalf("register user: %v", err)
	}
	token, _, err := auth.LoginUser(ctx, "snapshot-fallback@example.com", "secret123")
	if err != nil {
		t.Fatalf("login user: %v", err)
	}
	if err := st.AdminUpdateUser(ctx, userID, store.AdminUpdateUserInput{
		Email:        "snapshot-fallback@example.com",
		Balance:      "0.00",
		TrafficLimit: 10_000,
		TrafficUsed:  4096,
		Status:       "active",
	}); err != nil {
		t.Fatalf("update user traffic: %v", err)
	}

	resp := adminJSON(t, r, token, http.MethodGet, "/api/v1/traffic/snapshot", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("snapshot status=%d body=%s", resp.Code, resp.Body.String())
	}
	var got struct {
		Snapshot struct {
			TotalUsed    int64 `json:"total_used"`
			TrafficLimit int64 `json:"traffic_limit"`
		} `json:"snapshot"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	if got.Snapshot.TotalUsed != 4096 || got.Snapshot.TrafficLimit != 10_000 {
		t.Fatalf("snapshot should fall back to user totals, got %+v", got.Snapshot)
	}
}
