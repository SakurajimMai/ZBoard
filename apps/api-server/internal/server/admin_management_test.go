package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/zboard/api-server/internal/store"
)

func TestAdminListUsersSupportsFilters(t *testing.T) {
	r, st, token := setupAdminCRUDRouter(t)
	ctx := context.Background()

	planID, err := st.CreatePlan(ctx, store.CreatePlanInput{
		Name:         "筛选套餐",
		Price:        "9.90",
		DurationDays: 30,
		TrafficLimit: 1024,
		DeviceLimit:  2,
	})
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}
	expiredAt := time.Now().UTC().Add(-time.Hour)
	activeExpiry := time.Now().UTC().Add(24 * time.Hour)
	if _, err := st.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "active-filter@example.com",
		PasswordHash: "hash",
		PlanID:       &planID,
		ExpiredAt:    &activeExpiry,
		TrafficLimit: 1000,
		TrafficUsed:  250,
		Status:       "active",
	}); err != nil {
		t.Fatalf("create active user: %v", err)
	}
	if _, err := st.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "expired-filter@example.com",
		PasswordHash: "hash",
		ExpiredAt:    &expiredAt,
		TrafficLimit: 1000,
		TrafficUsed:  900,
		Status:       "active",
	}); err != nil {
		t.Fatalf("create expired user: %v", err)
	}
	if _, err := st.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "disabled-filter@example.com",
		PasswordHash: "hash",
		TrafficLimit: 1000,
		TrafficUsed:  100,
		Status:       "disabled",
	}); err != nil {
		t.Fatalf("create disabled user: %v", err)
	}

	resp := adminJSON(t, r, token, http.MethodGet, "/api/admin/v1/users?email=active-filter&status=active&plan_id="+strconv.FormatInt(planID, 10)+"&expires=valid&traffic_min=200&traffic_max=300", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("list filtered users status=%d body=%s", resp.Code, resp.Body.String())
	}
	var got struct {
		Items []struct {
			Email string `json:"email"`
		} `json:"items"`
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Total != 1 || len(got.Items) != 1 || got.Items[0].Email != "active-filter@example.com" {
		t.Fatalf("unexpected filtered users: %+v", got)
	}
}

func TestAdminCanBatchUpdateUsersAndResetSubscriptions(t *testing.T) {
	r, st, token := setupAdminCRUDRouter(t)
	ctx := context.Background()

	var ids []int64
	for i := 0; i < 2; i++ {
		id, err := st.AdminCreateUser(ctx, store.AdminCreateUserInput{
			Email:        "batch-" + strconv.Itoa(i) + "@example.com",
			PasswordHash: "hash",
			Status:       "active",
		})
		if err != nil {
			t.Fatalf("create user %d: %v", i, err)
		}
		if _, err := st.CreateSubToken(ctx, id, "old-token-"+strconv.Itoa(i), hashSubToken("old-token-"+strconv.Itoa(i))); err != nil {
			t.Fatalf("create token %d: %v", i, err)
		}
		ids = append(ids, id)
	}

	resp := adminJSON(t, r, token, http.MethodPost, "/api/admin/v1/users/batch", map[string]any{
		"action":   "disable",
		"user_ids": ids,
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("batch disable status=%d body=%s", resp.Code, resp.Body.String())
	}
	for _, id := range ids {
		u, err := st.FindUserByID(ctx, id)
		if err != nil {
			t.Fatalf("find user %d: %v", id, err)
		}
		if u.Status != "disabled" {
			t.Fatalf("user %d status=%s, want disabled", id, u.Status)
		}
	}

	resp = adminJSON(t, r, token, http.MethodPost, "/api/admin/v1/users/batch", map[string]any{
		"action":   "reset_subscription",
		"user_ids": ids,
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("batch reset subscription status=%d body=%s", resp.Code, resp.Body.String())
	}
	for i, id := range ids {
		tok, err := st.FindActiveSubTokenByUser(ctx, id)
		if err != nil {
			t.Fatalf("find active token %d: %v", id, err)
		}
		if tok.Token == "old-token-"+strconv.Itoa(i) || tok.Token == "" {
			t.Fatalf("token for user %d was not rotated: %+v", id, tok)
		}
	}

	resp = adminJSON(t, r, token, http.MethodPost, "/api/admin/v1/users/batch", map[string]any{
		"action":   "send_email",
		"user_ids": ids,
		"subject":  "通知",
		"content":  "测试邮件内容",
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("send email without mailer should fail explicitly, status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func TestAdminCanReadSubscriptionWithoutRotatingAndResetIdentity(t *testing.T) {
	r, st, token := setupAdminCRUDRouter(t)
	ctx := context.Background()

	userID, err := st.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "identity@example.com",
		PasswordHash: "hash",
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	nodeID, _, err := st.CreateNode(ctx, store.CreateNodeInput{
		Name:     "身份节点",
		Host:     "identity.example.com",
		Port:     443,
		Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}
	if err := st.EnsureNodeUser(ctx, userID, nodeID, "11111111-1111-4111-8111-111111111111", "vless"); err != nil {
		t.Fatalf("ensure node user: %v", err)
	}
	if _, err := st.CreateSubToken(ctx, userID, "stable-token", hashSubToken("stable-token")); err != nil {
		t.Fatalf("create token: %v", err)
	}

	resp := adminJSON(t, r, token, http.MethodGet, "/api/admin/v1/users/"+strconv.FormatInt(userID, 10)+"/subscription", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("get subscription status=%d body=%s", resp.Code, resp.Body.String())
	}
	var got struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode token: %v", err)
	}
	if got.Token != "stable-token" {
		t.Fatalf("subscription token rotated unexpectedly: %q", got.Token)
	}

	resp = adminJSON(t, r, token, http.MethodPost, "/api/admin/v1/users/"+strconv.FormatInt(userID, 10)+"/reset-identity", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("reset identity status=%d body=%s", resp.Code, resp.Body.String())
	}
	var reset struct {
		Token    string `json:"token"`
		ClientID string `json:"client_id"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &reset); err != nil {
		t.Fatalf("decode reset: %v", err)
	}
	if reset.Token == "" || reset.Token == "stable-token" || reset.ClientID == "" || reset.ClientID == "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("identity was not rotated: %+v", reset)
	}
}

func TestAdminCanManageAnnouncementsAndUsersCanReadActivePopups(t *testing.T) {
	r, _, token := setupAdminCRUDRouter(t)

	create := adminJSON(t, r, token, http.MethodPost, "/api/admin/v1/announcements", map[string]any{
		"title":    "维护通知",
		"content":  "今晚 23:00 维护",
		"popup":    true,
		"priority": 5,
		"status":   "active",
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create announcement status=%d body=%s", create.Code, create.Body.String())
	}
	var created struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.ID == 0 {
		t.Fatalf("announcement id is zero")
	}

	update := adminJSON(t, r, token, http.MethodPut, "/api/admin/v1/announcements/"+strconv.FormatInt(created.ID, 10), map[string]any{
		"title":    "维护通知更新",
		"content":  "今晚 23:30 维护",
		"popup":    true,
		"priority": 9,
		"status":   "active",
	})
	if update.Code != http.StatusOK {
		t.Fatalf("update announcement status=%d body=%s", update.Code, update.Body.String())
	}

	list := adminJSON(t, r, token, http.MethodGet, "/api/admin/v1/announcements", nil)
	if list.Code != http.StatusOK {
		t.Fatalf("list announcements status=%d body=%s", list.Code, list.Body.String())
	}
	var adminGot struct {
		Items []struct {
			Title    string `json:"title"`
			Priority int    `json:"priority"`
		} `json:"items"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &adminGot); err != nil {
		t.Fatalf("decode admin list: %v", err)
	}
	if len(adminGot.Items) != 1 || adminGot.Items[0].Title != "维护通知更新" || adminGot.Items[0].Priority != 9 {
		t.Fatalf("unexpected admin announcements: %+v", adminGot.Items)
	}

	publicResp := adminJSON(t, r, "", http.MethodGet, "/api/v1/announcements", nil)
	if publicResp.Code != http.StatusOK {
		t.Fatalf("public announcements status=%d body=%s", publicResp.Code, publicResp.Body.String())
	}
	var publicGot struct {
		Items []struct {
			Title   string `json:"title"`
			Content string `json:"content"`
			Popup   bool   `json:"popup"`
		} `json:"items"`
	}
	if err := json.Unmarshal(publicResp.Body.Bytes(), &publicGot); err != nil {
		t.Fatalf("decode public list: %v", err)
	}
	if len(publicGot.Items) != 1 || publicGot.Items[0].Title != "维护通知更新" || !publicGot.Items[0].Popup {
		t.Fatalf("unexpected public announcements: %+v", publicGot.Items)
	}
}
