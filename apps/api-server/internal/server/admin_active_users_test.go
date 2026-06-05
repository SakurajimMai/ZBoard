package server

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/zboard/api-server/internal/store"
)

func TestAdminActiveUsersListsRealTrafficForSelectedDay(t *testing.T) {
	r, st, token := setupAdminCRUDRouter(t)
	ctx := context.Background()

	userID, err := st.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "admin-active@example.com",
		PasswordHash: "hash",
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	nodeID, _, err := st.CreateNode(ctx, store.CreateNodeInput{
		Name:     "admin-active-node",
		Host:     "admin-active.example.com",
		Port:     443,
		Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}
	day := time.Now().UTC()
	insertAdminActiveTrafficLog(t, st, userID, nodeID, 1024, 2048, day)
	insertAdminActiveTrafficLog(t, st, userID, nodeID, 0, 0, day.Add(time.Minute))

	resp := adminJSON(t, r, token, http.MethodGet, "/api/admin/v1/active-users?date="+day.Format("2006-01-02"), nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("active users status=%d body=%s", resp.Code, resp.Body.String())
	}
	var got struct {
		Date    string `json:"date"`
		Total   int64  `json:"total"`
		Summary struct {
			ActiveUsers   int64 `json:"active_users"`
			UploadTotal   int64 `json:"upload_total"`
			DownloadTotal int64 `json:"download_total"`
			TrafficTotal  int64 `json:"traffic_total"`
		} `json:"summary"`
		Items []struct {
			UserID       int64  `json:"user_id"`
			Email        string `json:"email"`
			TrafficTotal int64  `json:"traffic_total"`
		} `json:"items"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Date != day.Format("2006-01-02") || got.Total != 1 || got.Summary.ActiveUsers != 1 || len(got.Items) != 1 {
		t.Fatalf("unexpected active users response: %+v", got)
	}
	if got.Summary.UploadTotal != 1024 || got.Summary.DownloadTotal != 2048 || got.Summary.TrafficTotal != 3072 {
		t.Fatalf("unexpected summary: %+v", got.Summary)
	}
	if got.Items[0].UserID != userID || got.Items[0].Email != "admin-active@example.com" || got.Items[0].TrafficTotal != 3072 {
		t.Fatalf("unexpected active user item: %+v", got.Items[0])
	}
}

func TestAdminActiveUsersRejectsDatesOutsideLastSevenDays(t *testing.T) {
	r, _, token := setupAdminCRUDRouter(t)
	oldDate := time.Now().UTC().AddDate(0, 0, -8).Format("2006-01-02")

	resp := adminJSON(t, r, token, http.MethodGet, "/api/admin/v1/active-users?date="+oldDate, nil)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("old date status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func insertAdminActiveTrafficLog(t *testing.T, s *store.Store, userID, nodeID, upload, download int64, reportedAt time.Time) {
	t.Helper()
	total := upload + download
	_, err := s.DB.ExecContext(context.Background(), s.Rebind(`INSERT INTO traffic_logs(user_id, node_id, upload_delta, download_delta, total_delta, reported_at)
		VALUES (?, ?, ?, ?, ?, ?)`), userID, nodeID, upload, download, total, reportedAt)
	if err != nil {
		t.Fatalf("insert traffic log: %v", err)
	}
}
