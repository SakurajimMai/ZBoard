package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/authsvc"
	"github.com/zboard/api-server/internal/bizsvc"
	"github.com/zboard/api-server/internal/nodesvc"
	"github.com/zboard/api-server/internal/payment/registry"
	"github.com/zboard/api-server/internal/store"
	"github.com/zboard/api-server/internal/testsupport"
	"github.com/zboard/api-server/internal/worker"
)

func setupAdminCRUDRouter(t *testing.T) (*gin.Engine, *store.Store, string) {
	t.Helper()
	st := testsupport.NewStore(t)
	auth := authsvc.New(st, "setup-token", nil)
	adminID, err := auth.BootstrapAdmin(context.Background(), "setup-token", "admin@example.com", "admin123")
	if err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}
	if adminID == 0 {
		t.Fatalf("bootstrap admin returned zero id")
	}
	token, _, err := auth.LoginAdmin(context.Background(), "admin@example.com", "admin123")
	if err != nil {
		t.Fatalf("login admin: %v", err)
	}
	r := New(Deps{
		DB:          st.DB,
		Store:       st,
		Auth:        auth,
		Biz:         bizsvc.New(st),
		Nodes:       nodesvc.New(st),
		Worker:      worker.New(st),
		Payments:    registry.New(st),
		TokenSecret: "test-token-secret-not-default",
	})
	return r, st, token
}

func adminJSON(t *testing.T, r http.Handler, token, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var b bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&b).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &b)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func ptrTime(t time.Time) *time.Time { return &t }

func TestAdminCanCreateAndUpdateUser(t *testing.T) {
	r, st, token := setupAdminCRUDRouter(t)

	nodeID, _, err := st.CreateNode(context.Background(), store.CreateNodeInput{
		Name:     "预置节点",
		Host:     "seed.example.com",
		Port:     443,
		Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("seed node: %v", err)
	}

	create := adminJSON(t, r, token, http.MethodPost, "/api/admin/v1/users", map[string]any{
		"email":         "managed@example.com",
		"password":      "secret123",
		"traffic_limit": int64(50 * 1024 * 1024 * 1024),
		"status":        "active",
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create user status=%d body=%s", create.Code, create.Body.String())
	}
	var created struct {
		UserID int64 `json:"user_id"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.UserID == 0 {
		t.Fatalf("create user returned zero id")
	}
	if _, err := st.FindNodeUser(context.Background(), created.UserID, nodeID); err != nil {
		t.Fatalf("created user should be provisioned on active node: %v", err)
	}

	update := adminJSON(t, r, token, http.MethodPut, "/api/admin/v1/users/"+strconv.FormatInt(created.UserID, 10), map[string]any{
		"email":         "managed-updated@example.com",
		"traffic_limit": int64(80 * 1024 * 1024 * 1024),
		"status":        "disabled",
	})
	if update.Code != http.StatusOK {
		t.Fatalf("update user status=%d body=%s", update.Code, update.Body.String())
	}
	u, err := st.FindUserByID(context.Background(), created.UserID)
	if err != nil {
		t.Fatalf("find user: %v", err)
	}
	if u.Email != "managed-updated@example.com" || u.TrafficLimit != int64(80*1024*1024*1024) || u.Status != "disabled" {
		t.Fatalf("unexpected user after update: %+v", u)
	}
	nu, err := st.FindNodeUser(context.Background(), created.UserID, nodeID)
	if err != nil {
		t.Fatalf("find node user: %v", err)
	}
	if nu.Enabled != 0 {
		t.Fatalf("disabled user should disable node_users, got enabled=%d", nu.Enabled)
	}
}

func TestAdminUserPlanDeviceLimitSyncToNodeUsers(t *testing.T) {
	r, st, token := setupAdminCRUDRouter(t)
	nodeID, _, err := st.CreateNode(context.Background(), store.CreateNodeInput{
		Name:     "预置节点",
		Host:     "seed.example.com",
		Port:     443,
		Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("seed node: %v", err)
	}
	planID, err := st.CreatePlan(context.Background(), store.CreatePlanInput{
		Name:         "设备套餐",
		Price:        "9.90",
		DurationDays: 30,
		TrafficLimit: int64(100 * 1024 * 1024 * 1024),
		DeviceLimit:  2,
	})
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}

	create := adminJSON(t, r, token, http.MethodPost, "/api/admin/v1/users", map[string]any{
		"email":         "limited@example.com",
		"password":      "secret123",
		"plan_id":       planID,
		"traffic_limit": int64(100 * 1024 * 1024 * 1024),
		"status":        "active",
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create user status=%d body=%s", create.Code, create.Body.String())
	}
	var created struct {
		UserID int64 `json:"user_id"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	nu, err := st.FindNodeUser(context.Background(), created.UserID, nodeID)
	if err != nil {
		t.Fatalf("find node user: %v", err)
	}
	if nu.DeviceLimit != 2 {
		t.Fatalf("node user device limit = %d, want 2", nu.DeviceLimit)
	}

	update := adminJSON(t, r, token, http.MethodPut, "/api/admin/v1/plans/"+strconv.FormatInt(planID, 10), map[string]any{
		"name":          "设备套餐",
		"price":         "19.90",
		"duration_days": 30,
		"traffic_limit": int64(100 * 1024 * 1024 * 1024),
		"device_limit":  4,
		"status":        "active",
	})
	if update.Code != http.StatusOK {
		t.Fatalf("update plan status=%d body=%s", update.Code, update.Body.String())
	}
	nu, err = st.FindNodeUser(context.Background(), created.UserID, nodeID)
	if err != nil {
		t.Fatalf("find node user after plan update: %v", err)
	}
	if nu.DeviceLimit != 4 {
		t.Fatalf("node user device limit after plan update = %d, want 4", nu.DeviceLimit)
	}
}

func TestSubscriptionRejectsExpiredAndTrafficExceededUsers(t *testing.T) {
	r, st, _ := setupAdminCRUDRouter(t)
	userID, err := st.AdminCreateUser(context.Background(), store.AdminCreateUserInput{
		Email:        "expired@example.com",
		PasswordHash: "hash",
		TrafficLimit: 100,
		TrafficUsed:  0,
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	expired := time.Now().UTC().Add(-time.Hour)
	if err := st.AdminUpdateUser(context.Background(), userID, store.AdminUpdateUserInput{
		Email:        "expired@example.com",
		Balance:      "0.00",
		ExpiredAt:    &expired,
		TrafficLimit: 100,
		TrafficUsed:  0,
		Status:       "active",
	}); err != nil {
		t.Fatalf("expire user: %v", err)
	}
	token := "sub-expired"
	if _, err := st.CreateSubToken(context.Background(), userID, token, hashSubToken(token)); err != nil {
		t.Fatalf("create token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/sub/"+token, nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expired subscription status=%d body=%s", rr.Code, rr.Body.String())
	}

	activeExpiry := time.Now().UTC().Add(time.Hour)
	if err := st.AdminUpdateUser(context.Background(), userID, store.AdminUpdateUserInput{
		Email:        "expired@example.com",
		Balance:      "0.00",
		ExpiredAt:    &activeExpiry,
		TrafficLimit: 100,
		TrafficUsed:  100,
		Status:       "active",
	}); err != nil {
		t.Fatalf("mark over quota: %v", err)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/sub/"+token, nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("traffic exceeded subscription status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestUserCanChangePasswordAndDeleteAccount(t *testing.T) {
	r, st, _ := setupAdminCRUDRouter(t)
	ctx := context.Background()
	auth := authsvc.New(st, "setup-token", nil)
	userID, err := auth.RegisterUser(ctx, "profile@example.com", "oldpass123")
	if err != nil {
		t.Fatalf("register user: %v", err)
	}
	token, _, err := auth.LoginUser(ctx, "profile@example.com", "oldpass123")
	if err != nil {
		t.Fatalf("login user: %v", err)
	}

	change := adminJSON(t, r, token, http.MethodPost, "/api/v1/me/password", map[string]any{
		"current_password": "oldpass123",
		"new_password":     "newpass123",
	})
	if change.Code != http.StatusOK {
		t.Fatalf("change password status=%d body=%s", change.Code, change.Body.String())
	}
	if _, _, err := auth.LoginUser(ctx, "profile@example.com", "oldpass123"); err == nil {
		t.Fatalf("old password should not work")
	}
	if _, _, err := auth.LoginUser(ctx, "profile@example.com", "newpass123"); err != nil {
		t.Fatalf("new password should work: %v", err)
	}

	deleteResp := adminJSON(t, r, token, http.MethodDelete, "/api/v1/me", map[string]any{
		"password": "newpass123",
	})
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("delete account status=%d body=%s", deleteResp.Code, deleteResp.Body.String())
	}
	u, err := st.FindUserByID(ctx, userID)
	if err != nil {
		t.Fatalf("find user: %v", err)
	}
	if u.Status != "deleted" {
		t.Fatalf("user status=%q, want deleted", u.Status)
	}
	if _, err := auth.ResolveUserToken(ctx, token); err == nil {
		t.Fatalf("deleted account token should be logged out")
	}
}

func TestSubscriptionUserinfoUsesUploadDownloadSnapshot(t *testing.T) {
	r, st, _ := setupAdminCRUDRouter(t)
	userID, err := st.AdminCreateUser(context.Background(), store.AdminCreateUserInput{
		Email:        "traffic-sub@example.com",
		PasswordHash: "hash",
		TrafficLimit: 10_000,
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	nodeID, _, err := st.CreateNode(context.Background(), store.CreateNodeInput{
		Name:     "流量节点",
		Host:     "traffic.example.com",
		Port:     443,
		Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}
	if err := st.EnsureNodeUser(context.Background(), userID, nodeID, "11111111-1111-4111-8111-111111111111", "vless"); err != nil {
		t.Fatalf("ensure node user: %v", err)
	}
	if err := st.RecordTraffic(context.Background(), []store.TrafficDelta{{
		UserID:        userID,
		NodeID:        nodeID,
		UploadDelta:   123,
		DownloadDelta: 456,
	}}); err != nil {
		t.Fatalf("record traffic: %v", err)
	}
	token := "sub-traffic"
	if _, err := st.CreateSubToken(context.Background(), userID, token, hashSubToken(token)); err != nil {
		t.Fatalf("create token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sub/"+token, nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("subscription status=%d body=%s", rr.Code, rr.Body.String())
	}
	got := rr.Header().Get("Subscription-Userinfo")
	for _, want := range []string{"upload=123", "download=456", "total=10000"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Subscription-Userinfo missing %q: %s", want, got)
		}
	}
}

func TestSettingsPersistAndControlRegistration(t *testing.T) {
	r, _, token := setupAdminCRUDRouter(t)

	body := `{"settings":{"allow_register":"0","site_name":"9Cloud","require_email_verify":"1"}}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/admin/v1/settings", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update settings code=%d body=%s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/admin/v1/settings", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get settings code=%d body=%s", w.Code, w.Body.String())
	}
	var got struct {
		Settings map[string]string `json:"settings"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	if got.Settings["site_name"] != "9Cloud" || got.Settings["allow_register"] != "0" {
		t.Fatalf("settings not persisted: %#v", got.Settings)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(`{"email":"blocked@example.com","password":"secret123"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("register code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestSubscriptionEnforcesDeviceLimit(t *testing.T) {
	r, st, _ := setupAdminCRUDRouter(t)
	userID, err := st.AdminCreateUser(context.Background(), store.AdminCreateUserInput{
		Email:        "devices@example.com",
		PasswordHash: "hash",
		TrafficLimit: 1000,
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	nodeID, _, err := st.CreateNode(context.Background(), store.CreateNodeInput{
		Name:     "设备节点",
		Host:     "device.example.com",
		Port:     443,
		Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}
	if err := st.EnsureNodeUserWithLimits(context.Background(), userID, nodeID, "11111111-1111-4111-8111-111111111111", "vless", 0, 1); err != nil {
		t.Fatalf("ensure node user: %v", err)
	}
	token := "sub-devices"
	if _, err := st.CreateSubToken(context.Background(), userID, token, hashSubToken(token)); err != nil {
		t.Fatalf("create token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sub/"+token, nil)
	req.Header.Set("User-Agent", "Client-A")
	req.RemoteAddr = "127.0.0.1:10000"
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("first device status=%d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/sub/"+token, nil)
	req.Header.Set("User-Agent", "Client-B")
	req.RemoteAddr = "127.0.0.2:10000"
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden || !bytes.Contains(rr.Body.Bytes(), []byte("device_limit_exceeded")) {
		t.Fatalf("second device should be rejected, status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSubscriptionDeviceLimitIgnoresExpiredOnlineWindow(t *testing.T) {
	r, st, _ := setupAdminCRUDRouter(t)
	ctx := context.Background()
	userID, err := st.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "devices-window@example.com",
		PasswordHash: "hash",
		TrafficLimit: 1000,
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	nodeID, _, err := st.CreateNode(ctx, store.CreateNodeInput{
		Name:     "设备窗口节点",
		Host:     "device-window.example.com",
		Port:     443,
		Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}
	if err := st.EnsureNodeUserWithLimits(ctx, userID, nodeID, "11111111-1111-4111-8111-111111111111", "vless", 0, 1); err != nil {
		t.Fatalf("ensure node user: %v", err)
	}
	token := "sub-devices-window"
	if _, err := st.CreateSubToken(ctx, userID, token, hashSubToken(token)); err != nil {
		t.Fatalf("create token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sub/"+token, nil)
	req.Header.Set("User-Agent", "Client-A")
	req.RemoteAddr = "127.0.0.1:10000"
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("first device status=%d body=%s", rr.Code, rr.Body.String())
	}

	staleSeenAt := time.Now().UTC().Add(-2 * time.Hour)
	if _, err := st.DB.ExecContext(ctx, st.Rebind(
		`UPDATE user_devices SET last_seen_at = ? WHERE user_id = ?`),
		staleSeenAt, userID); err != nil {
		t.Fatalf("mark device stale: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/sub/"+token, nil)
	req.Header.Set("User-Agent", "Client-B")
	req.RemoteAddr = "127.0.0.2:10000"
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("stale device should not consume online slot, status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSubscriptionRejectsUnknownTarget(t *testing.T) {
	r, st, _ := setupAdminCRUDRouter(t)
	userID, err := st.AdminCreateUser(context.Background(), store.AdminCreateUserInput{
		Email:        "unknown-target@example.com",
		PasswordHash: "hash",
		TrafficLimit: 1000,
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	nodeID, _, err := st.CreateNode(context.Background(), store.CreateNodeInput{
		Name:     "订阅节点",
		Host:     "sub.example.com",
		Port:     443,
		Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}
	if err := st.EnsureNodeUserWithLimits(context.Background(), userID, nodeID, "11111111-1111-4111-8111-111111111111", "vless", 0, 1); err != nil {
		t.Fatalf("ensure node user: %v", err)
	}
	token := "sub-unknown-target"
	if _, err := st.CreateSubToken(context.Background(), userID, token, hashSubToken(token)); err != nil {
		t.Fatalf("create token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sub/"+token+"?target=anything", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden || !bytes.Contains(rr.Body.Bytes(), []byte("subscription_target_disabled")) {
		t.Fatalf("unknown target should be rejected, status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSubscriptionInfersCompatibleTargetFromUserAgent(t *testing.T) {
	r, st, _ := setupAdminCRUDRouter(t)
	ctx := context.Background()
	userID, err := st.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "ua-target@example.com",
		PasswordHash: "hash",
		TrafficLimit: 1000,
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	nodeID, _, err := st.CreateNode(ctx, store.CreateNodeInput{
		Name:     "UA Target Node",
		Host:     "ua-target.example.com",
		Port:     443,
		Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}
	if err := st.EnsureNodeUser(ctx, userID, nodeID, "11111111-1111-4111-8111-111111111111", "vless"); err != nil {
		t.Fatalf("ensure node user: %v", err)
	}
	tokenValue := "sub-ua-target"
	if _, err := st.CreateSubToken(ctx, userID, tokenValue, hashSubToken(tokenValue)); err != nil {
		t.Fatalf("create sub token: %v", err)
	}

	cases := []struct {
		name        string
		ua          string
		wantContent string
		wantBody    string
	}{
		{name: "v2rayn", ua: "v2rayN/7.12.5", wantContent: "text/plain", wantBody: "vless://"},
		{name: "shadowrocket", ua: "Shadowrocket/2.2.50", wantContent: "text/plain", wantBody: "vless://"},
		{name: "passwall", ua: "PassWall", wantContent: "text/plain", wantBody: "vless://"},
		{name: "clash", ua: "clash.meta", wantContent: "text/yaml", wantBody: "proxies:"},
		{name: "sing-box", ua: "sing-box/1.11.8", wantContent: "application/json", wantBody: `"outbounds"`},
		{name: "hiddify", ua: "HiddifyNext/2.5.7", wantContent: "application/json", wantBody: `"outbounds"`},
		{name: "furious", ua: "Furious/0.10.0", wantContent: "application/json", wantBody: `"outbounds"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/sub/"+tokenValue, nil)
			req.Header.Set("User-Agent", tc.ua)
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("subscription status=%d body=%s", rr.Code, rr.Body.String())
			}
			if got := rr.Header().Get("Content-Type"); !strings.HasPrefix(got, tc.wantContent) {
				t.Fatalf("content-type=%q, want prefix %q", got, tc.wantContent)
			}
			body := rr.Body.String()
			if tc.wantContent == "text/plain" {
				raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(body))
				if err != nil {
					t.Fatalf("decode base64 subscription: %v", err)
				}
				body = string(raw)
			}
			if !strings.Contains(body, tc.wantBody) {
				t.Fatalf("body missing %q:\n%s", tc.wantBody, body)
			}
		})
	}
}

func TestSubscriptionAcceptsClientTargetAliases(t *testing.T) {
	r, st, _ := setupAdminCRUDRouter(t)
	ctx := context.Background()
	userID, err := st.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "target-alias@example.com",
		PasswordHash: "hash",
		TrafficLimit: 1000,
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	nodeID, _, err := st.CreateNode(ctx, store.CreateNodeInput{
		Name:     "Target Alias Node",
		Host:     "target-alias.example.com",
		Port:     443,
		Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}
	if err := st.EnsureNodeUser(ctx, userID, nodeID, "11111111-1111-4111-8111-111111111111", "vless"); err != nil {
		t.Fatalf("ensure node user: %v", err)
	}
	tokenValue := "sub-target-alias"
	if _, err := st.CreateSubToken(ctx, userID, tokenValue, hashSubToken(tokenValue)); err != nil {
		t.Fatalf("create sub token: %v", err)
	}

	cases := []struct {
		target      string
		wantStatus  int
		wantContent string
	}{
		{target: "v2rayn", wantStatus: http.StatusOK, wantContent: "text/plain"},
		{target: "shadowrocket", wantStatus: http.StatusOK, wantContent: "text/plain"},
		{target: "passwall", wantStatus: http.StatusOK, wantContent: "text/plain"},
		{target: "clash", wantStatus: http.StatusOK, wantContent: "text/yaml"},
		{target: "mihomo", wantStatus: http.StatusOK, wantContent: "text/yaml"},
		{target: "sing-box", wantStatus: http.StatusOK, wantContent: "application/json"},
		{target: "hiddify", wantStatus: http.StatusOK, wantContent: "application/json"},
		{target: "furious", wantStatus: http.StatusOK, wantContent: "application/json"},
		{target: "unknown", wantStatus: http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.target, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/sub/"+tokenValue+"?target="+url.QueryEscape(tc.target), nil)
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			if rr.Code != tc.wantStatus {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
			if tc.wantContent != "" {
				if got := rr.Header().Get("Content-Type"); !strings.HasPrefix(got, tc.wantContent) {
					t.Fatalf("content-type=%q, want prefix %q", got, tc.wantContent)
				}
			}
		})
	}
}

func TestSubscriptionFormatsDoNotConsumeExtraDeviceSlots(t *testing.T) {
	r, st, _ := setupAdminCRUDRouter(t)
	userID, err := st.AdminCreateUser(context.Background(), store.AdminCreateUserInput{
		Email:        "format-device@example.com",
		PasswordHash: "hash",
		TrafficLimit: 1000,
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	nodeID, _, err := st.CreateNode(context.Background(), store.CreateNodeInput{
		Name:     "格式节点",
		Host:     "format.example.com",
		Port:     443,
		Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}
	if err := st.EnsureNodeUserWithLimits(context.Background(), userID, nodeID, "11111111-1111-4111-8111-111111111111", "vless", 0, 1); err != nil {
		t.Fatalf("ensure node user: %v", err)
	}
	token := "sub-format-device"
	if _, err := st.CreateSubToken(context.Background(), userID, token, hashSubToken(token)); err != nil {
		t.Fatalf("create token: %v", err)
	}

	for _, target := range []string{"base64", "clash"} {
		req := httptest.NewRequest(http.MethodGet, "/api/sub/"+token+"?target="+target, nil)
		req.Header.Set("User-Agent", "Same-Client")
		req.RemoteAddr = "127.0.0.10:10000"
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("same device target %s status=%d body=%s", target, rr.Code, rr.Body.String())
		}
	}

	count, err := st.CountUserDevices(context.Background(), userID)
	if err != nil {
		t.Fatalf("count devices: %v", err)
	}
	if count != 1 {
		t.Fatalf("same device across subscription formats counted as %d devices, want 1", count)
	}
}

func TestEmailVerifySettingDoesNotAffectLogin(t *testing.T) {
	r, st, token := setupAdminCRUDRouter(t)
	auth := authsvc.New(st, "setup-token", nil)
	if _, err := auth.RegisterUser(context.Background(), "login@example.com", "secret123"); err != nil {
		t.Fatalf("register seed user: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/admin/v1/settings", bytes.NewBufferString(`{"settings":{"allow_register":"1","require_email_verify":"1"}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update settings code=%d body=%s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(`{"email":"need-code@example.com","password":"secret123"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register should not require email code when SMTP is unavailable, code=%d body=%s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"email":"login@example.com","password":"secret123"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login should not require email code, code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestAdminCanUpdatePlanAndNode(t *testing.T) {
	r, st, token := setupAdminCRUDRouter(t)

	planCreate := adminJSON(t, r, token, http.MethodPost, "/api/admin/v1/plans", map[string]any{
		"name":          "基础套餐",
		"price":         "9.90",
		"duration_days": 30,
		"traffic_limit": int64(100 * 1024 * 1024 * 1024),
		"device_limit":  2,
		"features":      []string{"100 GB 流量", "2 台设备同时在线"},
	})
	if planCreate.Code != http.StatusCreated {
		t.Fatalf("create plan status=%d body=%s", planCreate.Code, planCreate.Body.String())
	}
	var planResp struct {
		PlanID int64 `json:"plan_id"`
	}
	if err := json.Unmarshal(planCreate.Body.Bytes(), &planResp); err != nil {
		t.Fatalf("decode plan response: %v", err)
	}
	planUpdate := adminJSON(t, r, token, http.MethodPut, "/api/admin/v1/plans/1", map[string]any{
		"name":          "专业套餐",
		"price":         "19.90",
		"duration_days": 60,
		"traffic_limit": int64(200 * 1024 * 1024 * 1024),
		"device_limit":  5,
		"features":      []string{"200 GB 流量", "5 台设备同时在线", "支持 HY2"},
		"status":        "inactive",
		"sort":          9,
	})
	if planUpdate.Code != http.StatusOK {
		t.Fatalf("update plan status=%d body=%s", planUpdate.Code, planUpdate.Body.String())
	}
	plan, err := st.FindPlanByID(context.Background(), planResp.PlanID)
	if err != nil {
		t.Fatalf("find plan: %v", err)
	}
	if plan.Name != "专业套餐" || plan.Price != "19.90" || plan.DurationDays != 60 || plan.Status != "inactive" {
		t.Fatalf("unexpected plan after update: %+v", plan)
	}
	if len(plan.Features) != 3 || plan.Features[2] != "支持 HY2" {
		t.Fatalf("plan features not persisted: %#v", plan.Features)
	}

	nodeCreate := adminJSON(t, r, token, http.MethodPost, "/api/admin/v1/nodes", map[string]any{
		"name":     "香港 01",
		"host":     "hk.example.com",
		"port":     443,
		"protocol": "vless",
	})
	if nodeCreate.Code != http.StatusCreated {
		t.Fatalf("create node status=%d body=%s", nodeCreate.Code, nodeCreate.Body.String())
	}
	var nodeResp struct {
		NodeID int64 `json:"node_id"`
	}
	if err := json.Unmarshal(nodeCreate.Body.Bytes(), &nodeResp); err != nil {
		t.Fatalf("decode node response: %v", err)
	}
	nodeUpdate := adminJSON(t, r, token, http.MethodPut, "/api/admin/v1/nodes/1", map[string]any{
		"name":         "日本 01",
		"region":       "JP",
		"host":         "jp.example.com",
		"port":         8443,
		"protocol":     "hysteria2",
		"runtime_type": "sing-box",
		"security":     "tls",
		"status":       "inactive",
		"port_range":   "20000-40000",
	})
	if nodeUpdate.Code != http.StatusOK {
		t.Fatalf("update node status=%d body=%s", nodeUpdate.Code, nodeUpdate.Body.String())
	}
	node, err := st.FindNodeByID(context.Background(), nodeResp.NodeID)
	if err != nil {
		t.Fatalf("find node: %v", err)
	}
	if node.Name != "日本 01" || node.Host != "jp.example.com" || node.Port != 8443 || node.Protocol != "hysteria2" || node.PortRange != "20000-40000" || node.Status != "inactive" {
		t.Fatalf("unexpected node after update: %+v", node)
	}
	if node.TLSInsecure != 1 {
		t.Fatalf("hysteria2 node should default tls_insecure=1 for self-signed agent certificates, got %d", node.TLSInsecure)
	}
}

func TestAdminCreateAndUpdateNodeAutoEnqueuesSyncTask(t *testing.T) {
	r, st, token := setupAdminCRUDRouter(t)
	ctx := context.Background()

	nodeCreate := adminJSON(t, r, token, http.MethodPost, "/api/admin/v1/nodes", map[string]any{
		"name":         "US-01",
		"host":         "us.example.com",
		"port":         20925,
		"protocol":     "hysteria2",
		"runtime_type": "sing-box",
		"security":     "tls",
	})
	if nodeCreate.Code != http.StatusCreated {
		t.Fatalf("create node status=%d body=%s", nodeCreate.Code, nodeCreate.Body.String())
	}
	var created struct {
		NodeID     int64  `json:"node_id"`
		SyncTaskID string `json:"sync_task_id"`
	}
	if err := json.Unmarshal(nodeCreate.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.SyncTaskID == "" {
		t.Fatalf("create node should return sync_task_id, body=%s", nodeCreate.Body.String())
	}
	assertPendingSyncTasks := func(want int) {
		t.Helper()
		var count int
		if err := st.DB.GetContext(ctx, &count, st.Rebind(
			`SELECT COUNT(*) FROM node_tasks WHERE node_id = ? AND task_type = 'sync_config' AND status = 'pending'`),
			created.NodeID); err != nil {
			t.Fatalf("count sync tasks: %v", err)
		}
		if count != want {
			t.Fatalf("pending sync task count=%d, want %d", count, want)
		}
	}
	assertPendingSyncTasks(1)

	nodeUpdate := adminJSON(t, r, token, http.MethodPut, "/api/admin/v1/nodes/"+strconv.FormatInt(created.NodeID, 10), map[string]any{
		"name":         "US-01",
		"region":       "US",
		"host":         "117.55.226.11",
		"port":         20925,
		"protocol":     "hysteria2",
		"runtime_type": "sing-box",
		"security":     "tls",
		"status":       "active",
		"port_range":   "21000-22000",
	})
	if nodeUpdate.Code != http.StatusOK {
		t.Fatalf("update node status=%d body=%s", nodeUpdate.Code, nodeUpdate.Body.String())
	}
	if !bytes.Contains(nodeUpdate.Body.Bytes(), []byte("sync_task_id")) {
		t.Fatalf("update active node should return sync_task_id, body=%s", nodeUpdate.Body.String())
	}
	assertPendingSyncTasks(2)
}

func TestAdminCreateNodeBackfillsOnlyProvisionableUsers(t *testing.T) {
	r, st, token := setupAdminCRUDRouter(t)
	ctx := context.Background()
	now := time.Now().UTC()
	future := now.Add(24 * time.Hour)
	past := now.Add(-24 * time.Hour)

	validID, err := st.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "node-valid@example.com",
		PasswordHash: "hash",
		ExpiredAt:    &future,
		TrafficLimit: 100,
		TrafficUsed:  10,
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("create valid user: %v", err)
	}
	expiredID, err := st.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "node-expired@example.com",
		PasswordHash: "hash",
		ExpiredAt:    &past,
		TrafficLimit: 100,
		TrafficUsed:  10,
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("create expired user: %v", err)
	}
	overQuotaID, err := st.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "node-overquota@example.com",
		PasswordHash: "hash",
		ExpiredAt:    &future,
		TrafficLimit: 100,
		TrafficUsed:  100,
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("create over quota user: %v", err)
	}

	resp := adminJSON(t, r, token, http.MethodPost, "/api/admin/v1/nodes", map[string]any{
		"name":     "Backfill Node",
		"host":     "provision.example.com",
		"port":     443,
		"protocol": "vless",
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("create node status=%d body=%s", resp.Code, resp.Body.String())
	}
	var body struct {
		NodeID int64 `json:"node_id"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode node: %v", err)
	}
	if _, err := st.FindNodeUser(ctx, validID, body.NodeID); err != nil {
		t.Fatalf("valid user should be provisioned: %v", err)
	}
	if _, err := st.FindNodeUser(ctx, expiredID, body.NodeID); !store.IsNoRows(err) {
		t.Fatalf("expired user should not be provisioned, err=%v", err)
	}
	if _, err := st.FindNodeUser(ctx, overQuotaID, body.NodeID); !store.IsNoRows(err) {
		t.Fatalf("over-quota user should not be provisioned, err=%v", err)
	}
}

func TestAdminUpdateNodeActiveBackfillsProvisionableUsers(t *testing.T) {
	r, st, token := setupAdminCRUDRouter(t)
	ctx := context.Background()
	future := time.Now().UTC().Add(24 * time.Hour)

	userID, err := st.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "node-update-backfill@example.com",
		PasswordHash: "hash",
		ExpiredAt:    &future,
		TrafficLimit: 100,
		TrafficUsed:  10,
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	nodeID, _, err := st.CreateNode(ctx, store.CreateNodeInput{
		Name:     "Existing HY2",
		Host:     "hy.example.com",
		Port:     20925,
		Protocol: "hysteria2",
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}
	if _, err := st.FindNodeUser(ctx, userID, nodeID); !store.IsNoRows(err) {
		t.Fatalf("test setup should start without node_user mapping, err=%v", err)
	}

	updateResp := adminJSON(t, r, token, http.MethodPut, "/api/admin/v1/nodes/"+strconv.FormatInt(nodeID, 10), map[string]any{
		"name":       "Existing HY2",
		"host":       "hy.example.com",
		"port":       20925,
		"protocol":   "hysteria2",
		"security":   "tls",
		"status":     "active",
		"port_range": "21000-22000",
	})
	if updateResp.Code != http.StatusOK {
		t.Fatalf("update node status=%d body=%s", updateResp.Code, updateResp.Body.String())
	}
	nu, err := st.FindNodeUser(ctx, userID, nodeID)
	if err != nil {
		t.Fatalf("active node update should provision user: %v", err)
	}
	if nu.Protocol != "hysteria2" || nu.Enabled != 1 {
		t.Fatalf("unexpected provisioned node user: %+v", nu)
	}

	tokenValue := "sub-hy2-backfill"
	if _, err := st.CreateSubToken(ctx, userID, tokenValue, hashSubToken(tokenValue)); err != nil {
		t.Fatalf("create sub token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/sub/"+tokenValue+"?target=base64", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("subscription status=%d body=%s", rr.Code, rr.Body.String())
	}
	raw, err := base64.StdEncoding.DecodeString(rr.Body.String())
	if err != nil {
		t.Fatalf("decode subscription: %v", err)
	}
	if !bytes.Contains(raw, []byte("hysteria2://")) {
		t.Fatalf("subscription should include hy2 node, got:\n%s", raw)
	}
}

func TestSubscriptionBackfillsMissingActiveNodeMappingForMigratedUser(t *testing.T) {
	r, st, _ := setupAdminCRUDRouter(t)
	ctx := context.Background()
	future := time.Now().UTC().Add(24 * time.Hour)

	planID, err := st.CreatePlan(ctx, store.CreatePlanInput{
		Name:         "migrated-plan",
		Price:        "9.90",
		DurationDays: 30,
		TrafficLimit: 1000,
		DeviceLimit:  2,
	})
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}
	userID, err := st.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "migrated-hy2@example.com",
		PasswordHash: "hash",
		PlanID:       &planID,
		ExpiredAt:    &future,
		TrafficLimit: 1000,
		TrafficUsed:  100,
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("create migrated user: %v", err)
	}
	nodeID, _, err := st.CreateNode(ctx, store.CreateNodeInput{
		Name:         "HY2 Only",
		Region:       "US",
		Host:         "hy2.example.com",
		Port:         20925,
		Protocol:     "hysteria2",
		RuntimeType:  "sing-box",
		PortRange:    "21000-22000",
		UpMbps:       100,
		DownMbps:     200,
		ObfsPassword: "obfs-secret",
	})
	if err != nil {
		t.Fatalf("create hy2 node: %v", err)
	}
	if _, err := st.FindNodeUser(ctx, userID, nodeID); !store.IsNoRows(err) {
		t.Fatalf("test setup should start without node mapping, err=%v", err)
	}
	tokenValue := "sub-migrated-hy2"
	if _, err := st.CreateSubToken(ctx, userID, tokenValue, hashSubToken(tokenValue)); err != nil {
		t.Fatalf("create sub token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sub/"+tokenValue+"?target=base64", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("subscription status=%d body=%s", rr.Code, rr.Body.String())
	}
	raw, err := base64.StdEncoding.DecodeString(rr.Body.String())
	if err != nil {
		t.Fatalf("decode subscription: %v", err)
	}
	if !bytes.Contains(raw, []byte("hysteria2://")) {
		t.Fatalf("subscription should self-heal and include hy2 node, got:\n%s", raw)
	}
	nu, err := st.FindNodeUser(ctx, userID, nodeID)
	if err != nil {
		t.Fatalf("missing node mapping should be created: %v", err)
	}
	if nu.Enabled != 1 || nu.Protocol != "hysteria2" || nu.DeviceLimit != 2 {
		t.Fatalf("unexpected healed node mapping: %+v", nu)
	}
}

func TestSubscriptionDoesNotReEnableDisabledNodeMapping(t *testing.T) {
	r, st, _ := setupAdminCRUDRouter(t)
	ctx := context.Background()
	future := time.Now().UTC().Add(24 * time.Hour)

	userID, err := st.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "disabled-node-user@example.com",
		PasswordHash: "hash",
		ExpiredAt:    &future,
		TrafficLimit: 1000,
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	nodeID, _, err := st.CreateNode(ctx, store.CreateNodeInput{
		Name:     "disabled mapping node",
		Host:     "node.example.com",
		Port:     443,
		Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}
	if err := st.EnsureNodeUserWithLimits(ctx, userID, nodeID, "11111111-1111-4111-8111-111111111111", "vless", 0, 1); err != nil {
		t.Fatalf("seed node user: %v", err)
	}
	if err := st.SetNodeUserEnabledForUser(ctx, userID, 0); err != nil {
		t.Fatalf("disable node user: %v", err)
	}
	tokenValue := "sub-disabled-node"
	if _, err := st.CreateSubToken(ctx, userID, tokenValue, hashSubToken(tokenValue)); err != nil {
		t.Fatalf("create sub token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sub/"+tokenValue+"?target=base64", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("subscription status=%d body=%s", rr.Code, rr.Body.String())
	}
	raw, err := base64.StdEncoding.DecodeString(rr.Body.String())
	if err != nil {
		t.Fatalf("decode subscription: %v", err)
	}
	if len(strings.TrimSpace(string(raw))) != 0 {
		t.Fatalf("disabled node mapping should stay out of subscription, got:\n%s", raw)
	}
	nu, err := st.FindNodeUser(ctx, userID, nodeID)
	if err != nil {
		t.Fatalf("find node user: %v", err)
	}
	if nu.Enabled != 0 {
		t.Fatalf("subscription self-heal must not re-enable existing disabled mapping: %+v", nu)
	}
}

func TestAdminNodeListReturnsHealthIndicators(t *testing.T) {
	r, st, token := setupAdminCRUDRouter(t)
	ctx := context.Background()

	greenID, _, err := st.CreateNode(ctx, store.CreateNodeInput{
		Name:     "使用中节点",
		Host:     "green.example.com",
		Port:     443,
		Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("create green node: %v", err)
	}
	yellowID, _, err := st.CreateNode(ctx, store.CreateNodeInput{
		Name:     "空闲节点",
		Host:     "yellow.example.com",
		Port:     443,
		Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("create yellow node: %v", err)
	}
	oldTrafficID, _, err := st.CreateNode(ctx, store.CreateNodeInput{
		Name:     "历史使用节点",
		Host:     "old-traffic.example.com",
		Port:     443,
		Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("create old traffic node: %v", err)
	}
	redID, _, err := st.CreateNode(ctx, store.CreateNodeInput{
		Name:     "异常节点",
		Host:     "red.example.com",
		Port:     443,
		Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("create red node: %v", err)
	}
	userID, err := st.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "node-health@example.com",
		PasswordHash: "hash",
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := st.EnsureNodeUserWithLimits(ctx, userID, greenID, "11111111-1111-4111-8111-111111111111", "vless", 0, 1); err != nil {
		t.Fatalf("ensure node user: %v", err)
	}
	if err := st.EnsureNodeUserWithLimits(ctx, userID, yellowID, "22222222-2222-4222-8222-222222222222", "vless", 0, 1); err != nil {
		t.Fatalf("ensure idle node user: %v", err)
	}
	if err := st.EnsureNodeUserWithLimits(ctx, userID, oldTrafficID, "33333333-3333-4333-8333-333333333333", "vless", 0, 1); err != nil {
		t.Fatalf("ensure old traffic node user: %v", err)
	}
	now := time.Now().UTC()
	for _, id := range []int64{greenID, yellowID, oldTrafficID} {
		if err := st.RecordHeartbeat(ctx, store.HeartbeatInput{
			NodeID:        id,
			AgentVersion:  "test",
			RuntimeStatus: "running",
			ReportedAt:    now,
		}); err != nil {
			t.Fatalf("record running heartbeat %d: %v", id, err)
		}
	}
	if err := st.RecordTraffic(ctx, []store.TrafficDelta{
		{UserID: userID, NodeID: greenID, UploadDelta: 128, DownloadDelta: 512},
		{UserID: userID, NodeID: oldTrafficID, UploadDelta: 64, DownloadDelta: 64},
	}); err != nil {
		t.Fatalf("record traffic: %v", err)
	}
	if _, err := st.DB.ExecContext(ctx, st.Rebind(`UPDATE traffic_logs SET reported_at = ? WHERE node_id = ?`), now.Add(-10*time.Minute), oldTrafficID); err != nil {
		t.Fatalf("age old traffic: %v", err)
	}
	if err := st.RecordHeartbeat(ctx, store.HeartbeatInput{
		NodeID:        redID,
		AgentVersion:  "test",
		RuntimeStatus: "stopped",
		ReportedAt:    now,
	}); err != nil {
		t.Fatalf("record stopped heartbeat: %v", err)
	}

	resp := adminJSON(t, r, token, http.MethodGet, "/api/admin/v1/nodes?page=1&page_size=10", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("list nodes status=%d body=%s", resp.Code, resp.Body.String())
	}
	var got struct {
		Items []struct {
			ID              int64  `json:"id"`
			HealthStatus    string `json:"health_status"`
			HealthLabel     string `json:"health_label"`
			ActiveUserCount int64  `json:"active_user_count"`
			RuntimeStatus   string `json:"runtime_status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode nodes: %v", err)
	}
	byID := map[int64]struct {
		HealthStatus    string
		HealthLabel     string
		ActiveUserCount int64
		RuntimeStatus   string
	}{}
	for _, n := range got.Items {
		byID[n.ID] = struct {
			HealthStatus    string
			HealthLabel     string
			ActiveUserCount int64
			RuntimeStatus   string
		}{n.HealthStatus, n.HealthLabel, n.ActiveUserCount, n.RuntimeStatus}
	}
	if got := byID[greenID]; got.HealthStatus != "green" || got.HealthLabel != "使用中" || got.ActiveUserCount != 1 || got.RuntimeStatus != "running" {
		t.Fatalf("green node health mismatch: %+v", got)
	}
	if got := byID[yellowID]; got.HealthStatus != "yellow" || got.HealthLabel != "空闲" || got.ActiveUserCount != 0 || got.RuntimeStatus != "running" {
		t.Fatalf("yellow node health mismatch: %+v", got)
	}
	if got := byID[oldTrafficID]; got.HealthStatus != "yellow" || got.HealthLabel != "空闲" || got.ActiveUserCount != 0 || got.RuntimeStatus != "running" {
		t.Fatalf("old traffic node health mismatch: %+v", got)
	}
	if got := byID[redID]; got.HealthStatus != "red" || got.HealthLabel != "异常" || got.RuntimeStatus != "stopped" {
		t.Fatalf("red node health mismatch: %+v", got)
	}
}

func TestAdminNodeListReturnsTrafficTotals(t *testing.T) {
	r, st, token := setupAdminCRUDRouter(t)
	ctx := context.Background()

	nodeID, _, err := st.CreateNode(ctx, store.CreateNodeInput{
		Name:     "统计节点",
		Host:     "traffic-node.example.com",
		Port:     443,
		Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}
	userID, err := st.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "node-traffic@example.com",
		PasswordHash: "hash",
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := st.EnsureNodeUser(ctx, userID, nodeID, "11111111-1111-4111-8111-111111111111", "vless"); err != nil {
		t.Fatalf("ensure node user: %v", err)
	}
	if err := st.RecordTraffic(ctx, []store.TrafficDelta{
		{UserID: userID, NodeID: nodeID, UploadDelta: 1234, DownloadDelta: 5678},
		{UserID: userID, NodeID: nodeID, UploadDelta: 11, DownloadDelta: 22},
	}); err != nil {
		t.Fatalf("record traffic: %v", err)
	}

	resp := adminJSON(t, r, token, http.MethodGet, "/api/admin/v1/nodes?page=1&page_size=10", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("list nodes status=%d body=%s", resp.Code, resp.Body.String())
	}
	var got struct {
		Items []struct {
			ID            int64 `json:"id"`
			UploadTotal   int64 `json:"upload_total"`
			DownloadTotal int64 `json:"download_total"`
			TrafficTotal  int64 `json:"traffic_total"`
		} `json:"items"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode nodes: %v", err)
	}
	for _, n := range got.Items {
		if n.ID != nodeID {
			continue
		}
		if n.UploadTotal != 1245 || n.DownloadTotal != 5700 || n.TrafficTotal != 6945 {
			t.Fatalf("node traffic mismatch: %+v", n)
		}
		return
	}
	t.Fatalf("node %d not found in response: %+v", nodeID, got.Items)
}

func TestAdminPlanPagination(t *testing.T) {
	r, _, token := setupAdminCRUDRouter(t)
	for i := 0; i < 15; i++ {
		resp := adminJSON(t, r, token, http.MethodPost, "/api/admin/v1/plans", map[string]any{
			"name":          "套餐" + strconv.Itoa(i),
			"price":         "9.90",
			"duration_days": 30,
			"traffic_limit": int64(10 * 1024 * 1024 * 1024),
			"device_limit":  3,
			"sort":          i,
		})
		if resp.Code != http.StatusCreated {
			t.Fatalf("create plan %d status=%d body=%s", i, resp.Code, resp.Body.String())
		}
	}
	list := adminJSON(t, r, token, http.MethodGet, "/api/admin/v1/plans?page=2&page_size=5", nil)
	if list.Code != http.StatusOK {
		t.Fatalf("list plans status=%d body=%s", list.Code, list.Body.String())
	}
	var got struct {
		Items []any `json:"items"`
		Page  int   `json:"page"`
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if got.Page != 2 || len(got.Items) != 5 || got.Total != 15 {
		t.Fatalf("unexpected pagination response: page=%d len=%d total=%d", got.Page, len(got.Items), got.Total)
	}
}

func TestAdminOverviewUsesAggregateTotals(t *testing.T) {
	r, st, token := setupAdminCRUDRouter(t)
	ctx := context.Background()

	planID, err := st.CreatePlan(ctx, store.CreatePlanInput{
		Name:         "统计套餐",
		Price:        "1.00",
		DurationDays: 30,
		TrafficLimit: 1024,
		DeviceLimit:  1,
	})
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}
	userID, err := st.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "overview@example.com",
		PasswordHash: "hash",
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	now := time.Now().UTC()
	for i := 0; i < 105; i++ {
		status := "paid"
		if i == 0 {
			status = "pending"
		}
		if _, err := st.CreateOrder(ctx, &store.Order{
			OrderNo:   "OV-" + strconv.Itoa(i),
			UserID:    userID,
			PlanID:    planID,
			Amount:    "1.50",
			Currency:  "CNY",
			Status:    status,
			ExpiredAt: ptrTime(time.Now().UTC().Add(time.Hour)),
		}); err != nil {
			t.Fatalf("create order %d: %v", i, err)
		}
	}
	for i := 0; i < 105; i++ {
		nodeID, _, err := st.CreateNode(ctx, store.CreateNodeInput{
			Name:     "概览节点" + strconv.Itoa(i),
			Host:     "overview-" + strconv.Itoa(i) + ".example.com",
			Port:     443,
			Protocol: "vless",
		})
		if err != nil {
			t.Fatalf("create node %d: %v", i, err)
		}
		if i == 0 {
			if err := st.UpdateNode(ctx, nodeID, store.UpdateNodeInput{
				Name:        "停用节点",
				Host:        "overview-0.example.com",
				Port:        443,
				Protocol:    "vless",
				Transport:   "tcp",
				Security:    "tls",
				RuntimeType: "xray",
				Status:      "inactive",
				WSPath:      "/",
				SNI:         "overview-0.example.com",
			}); err != nil {
				t.Fatalf("disable node: %v", err)
			}
		}
		switch i {
		case 0:
			if err := st.RecordHeartbeat(ctx, store.HeartbeatInput{NodeID: nodeID, AgentVersion: "test", RuntimeStatus: "running", ReportedAt: now}); err != nil {
				t.Fatalf("record inactive node heartbeat: %v", err)
			}
		case 1, 2:
			if err := st.RecordHeartbeat(ctx, store.HeartbeatInput{NodeID: nodeID, AgentVersion: "test", RuntimeStatus: "running", ReportedAt: now}); err != nil {
				t.Fatalf("record online node heartbeat: %v", err)
			}
		case 3:
			if err := st.RecordHeartbeat(ctx, store.HeartbeatInput{NodeID: nodeID, AgentVersion: "test", RuntimeStatus: "running", ReportedAt: now.Add(-10 * time.Minute)}); err != nil {
				t.Fatalf("record stale node heartbeat: %v", err)
			}
		case 4:
			if err := st.RecordHeartbeat(ctx, store.HeartbeatInput{NodeID: nodeID, AgentVersion: "test", RuntimeStatus: "stopped", ReportedAt: now}); err != nil {
				t.Fatalf("record stopped node heartbeat: %v", err)
			}
		}
	}

	if err := st.RecordTraffic(ctx, []store.TrafficDelta{{
		UserID:        userID,
		NodeID:        2,
		UploadDelta:   1099511627776,
		DownloadDelta: 1099511627776,
	}}); err != nil {
		t.Fatalf("record traffic: %v", err)
	}

	resp := adminJSON(t, r, token, http.MethodGet, "/api/admin/v1/overview", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("overview status=%d body=%s", resp.Code, resp.Body.String())
	}
	var got struct {
		Users        int64  `json:"users"`
		ActiveNodes  int64  `json:"active_nodes"`
		PaidOrders   int64  `json:"paid_orders"`
		Revenue      string `json:"revenue"`
		RevenueTrend []struct {
			Month   string  `json:"month"`
			Label   string  `json:"label"`
			Revenue float64 `json:"revenue"`
		} `json:"revenue_trend"`
		TrafficTrend []struct {
			Day   string  `json:"day"`
			Label string  `json:"label"`
			Total int64   `json:"total"`
			TB    float64 `json:"tb"`
		} `json:"traffic_trend"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode overview: %v", err)
	}
	if got.Users != 1 || got.ActiveNodes != 2 || got.PaidOrders != 104 || got.Revenue != "156.00" {
		t.Fatalf("unexpected overview: %+v", got)
	}
	if len(got.RevenueTrend) != 6 || got.RevenueTrend[5].Revenue != 156 {
		t.Fatalf("unexpected revenue trend: %+v", got.RevenueTrend)
	}
	if len(got.TrafficTrend) != 7 || got.TrafficTrend[6].Total != 2199023255552 || got.TrafficTrend[6].TB != 2 {
		t.Fatalf("unexpected traffic trend: %+v", got.TrafficTrend)
	}
}

func TestAdminSendTestEmailRequiresMailerConfig(t *testing.T) {
	r, _, token := setupAdminCRUDRouter(t)

	resp := adminJSON(t, r, token, http.MethodPost, "/api/admin/v1/settings/test-email", nil)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("test email status=%d body=%s", resp.Code, resp.Body.String())
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte("mailer_not_configured")) {
		t.Fatalf("test email should report missing mailer, body=%s", resp.Body.String())
	}
}

func TestResolveTestEmailRecipientAllowsCustomAddress(t *testing.T) {
	admin := &store.AdminUser{Email: "admin@example.com"}

	got, err := resolveTestEmailRecipient(testEmailBody{Email: " tester@example.com "}, admin)
	if err != nil {
		t.Fatalf("resolve custom recipient: %v", err)
	}
	if got != "tester@example.com" {
		t.Fatalf("recipient=%q, want tester@example.com", got)
	}

	got, err = resolveTestEmailRecipient(testEmailBody{}, admin)
	if err != nil {
		t.Fatalf("resolve fallback recipient: %v", err)
	}
	if got != "admin@example.com" {
		t.Fatalf("fallback recipient=%q, want admin@example.com", got)
	}

	if _, err := resolveTestEmailRecipient(testEmailBody{Email: "not-an-email"}, admin); err == nil {
		t.Fatalf("invalid custom recipient should fail")
	}
}

func TestAdminSendTestEmailRejectsInvalidCustomAddress(t *testing.T) {
	r, _, token := setupAdminCRUDRouter(t)

	resp := adminJSON(t, r, token, http.MethodPost, "/api/admin/v1/settings/test-email", map[string]any{
		"email": "not-an-email",
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("test email status=%d body=%s", resp.Code, resp.Body.String())
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte("invalid_test_email")) {
		t.Fatalf("test email should report invalid address, body=%s", resp.Body.String())
	}
}

// TestAdminNodeListMasksRealityPrivateKey is the H17 regression: the admin node
// list must not return the raw Reality private key (or hysteria2 obfs password).
// A blunt json:"-" would break the edit form, which reads the value back — so we
// also verify that re-submitting the masked placeholder preserves the stored key.
func TestAdminNodeListMasksRealityPrivateKey(t *testing.T) {
	r, st, token := setupAdminCRUDRouter(t)
	ctx := context.Background()

	const realKey = "PRIVATE-KEY-HEX-1234567890"
	create := adminJSON(t, r, token, http.MethodPost, "/api/admin/v1/nodes", map[string]any{
		"name":                "reality-secret",
		"host":                "rs.example.com",
		"port":                443,
		"protocol":            "vless",
		"transport":           "tcp",
		"security":            "reality",
		"runtime_type":        "xray",
		"reality_server_name": "www.cloudflare.com",
		"reality_public_key":  "PUBKEY",
		"reality_private_key": realKey,
		"reality_dest":        "www.cloudflare.com:443",
		"flow":                "xtls-rprx-vision",
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create reality node status=%d body=%s", create.Code, create.Body.String())
	}

	list := adminJSON(t, r, token, http.MethodGet, "/api/admin/v1/nodes", nil)
	if list.Code != http.StatusOK {
		t.Fatalf("list nodes status=%d body=%s", list.Code, list.Body.String())
	}
	if bytes.Contains(list.Body.Bytes(), []byte(realKey)) {
		t.Fatalf("admin node list leaked raw reality private key: %s", list.Body.String())
	}
	if !bytes.Contains(list.Body.Bytes(), []byte("****")) {
		t.Fatalf("admin node list should mask the private key, body=%s", list.Body.String())
	}

	// Decode the masked value the form would echo back, then re-submit it.
	var listResp struct {
		Items []struct {
			ID                int64  `json:"id"`
			RealityPrivateKey string `json:"reality_private_key"`
		} `json:"items"`
	}
	body := list.Body.Bytes()
	// httpx.OK wraps payload under "data"; tolerate either shape.
	if err := json.Unmarshal(body, &listResp); err != nil || len(listResp.Items) == 0 {
		var wrapped struct {
			Data struct {
				Items []struct {
					ID                int64  `json:"id"`
					RealityPrivateKey string `json:"reality_private_key"`
				} `json:"items"`
			} `json:"data"`
		}
		if err2 := json.Unmarshal(body, &wrapped); err2 != nil {
			t.Fatalf("decode list: %v / %v body=%s", err, err2, list.Body.String())
		}
		listResp.Items = wrapped.Data.Items
	}
	if len(listResp.Items) == 0 {
		t.Fatalf("no nodes in list response: %s", list.Body.String())
	}
	maskedKey := listResp.Items[0].RealityPrivateKey
	nodeID := listResp.Items[0].ID
	if maskedKey == realKey {
		t.Fatalf("decoded list still contains raw key")
	}

	upd := adminJSON(t, r, token, http.MethodPut, "/api/admin/v1/nodes/"+strconv.FormatInt(nodeID, 10), map[string]any{
		"name":                "reality-secret",
		"host":                "rs.example.com",
		"port":                443,
		"protocol":            "vless",
		"transport":           "tcp",
		"security":            "reality",
		"runtime_type":        "xray",
		"reality_server_name": "www.cloudflare.com",
		"reality_public_key":  "PUBKEY",
		"reality_private_key": maskedKey, // echo the mask back, as the UI does
		"reality_dest":        "www.cloudflare.com:443",
		"flow":                "xtls-rprx-vision",
		"status":              "active",
	})
	if upd.Code != http.StatusOK {
		t.Fatalf("update with masked key status=%d body=%s", upd.Code, upd.Body.String())
	}

	// The stored key must be unchanged — the mask must not have overwritten it.
	n, err := st.FindNodeByID(ctx, nodeID)
	if err != nil {
		t.Fatalf("find node: %v", err)
	}
	if n.RealityPrivateKey != realKey {
		t.Fatalf("masked re-submit corrupted stored key: got %q want %q", n.RealityPrivateKey, realKey)
	}
}

func TestAdminRejectsIncompleteRealityNode(t *testing.T) {
	r, _, token := setupAdminCRUDRouter(t)

	create := adminJSON(t, r, token, http.MethodPost, "/api/admin/v1/nodes", map[string]any{
		"name":                "美国 01",
		"host":                "us.example.com",
		"port":                443,
		"protocol":            "vless",
		"transport":           "tcp",
		"security":            "reality",
		"runtime_type":        "xray",
		"reality_server_name": "www.cloudflare.com",
		"reality_public_key":  "PBK",
		"reality_private_key": "",
		"reality_dest":        "www.cloudflare.com:443",
		"flow":                "xtls-rprx-vision",
	})
	if create.Code != http.StatusBadRequest {
		t.Fatalf("create incomplete reality node status=%d body=%s", create.Code, create.Body.String())
	}
	if !bytes.Contains(create.Body.Bytes(), []byte("reality_private_key")) {
		t.Fatalf("create error should mention missing private key, body=%s", create.Body.String())
	}

	okCreate := adminJSON(t, r, token, http.MethodPost, "/api/admin/v1/nodes", map[string]any{
		"name":         "美国 01",
		"host":         "us.example.com",
		"port":         443,
		"protocol":     "vless",
		"transport":    "tcp",
		"security":     "tls",
		"runtime_type": "xray",
	})
	if okCreate.Code != http.StatusCreated {
		t.Fatalf("create base node status=%d body=%s", okCreate.Code, okCreate.Body.String())
	}

	update := adminJSON(t, r, token, http.MethodPut, "/api/admin/v1/nodes/1", map[string]any{
		"name":                "美国 01",
		"host":                "us.example.com",
		"port":                443,
		"protocol":            "vless",
		"transport":           "tcp",
		"security":            "reality",
		"runtime_type":        "xray",
		"reality_server_name": "www.cloudflare.com",
		"reality_public_key":  "",
		"reality_private_key": "PRIVATE-KEY-HEX",
		"reality_dest":        "www.cloudflare.com:443",
		"status":              "active",
	})
	if update.Code != http.StatusBadRequest {
		t.Fatalf("update incomplete reality node status=%d body=%s", update.Code, update.Body.String())
	}
	if !bytes.Contains(update.Body.Bytes(), []byte("reality_public_key")) {
		t.Fatalf("update error should mention missing public key, body=%s", update.Body.String())
	}
}

func TestAdminRealityNodeDoesNotDefaultFlow(t *testing.T) {
	r, st, token := setupAdminCRUDRouter(t)

	create := adminJSON(t, r, token, http.MethodPost, "/api/admin/v1/nodes", map[string]any{
		"name":                "美国 02",
		"host":                "us2.example.com",
		"port":                443,
		"protocol":            "vless",
		"transport":           "tcp",
		"security":            "reality",
		"runtime_type":        "xray",
		"reality_server_name": "www.cloudflare.com",
		"reality_public_key":  "PBK",
		"reality_private_key": "PRIVATE-KEY-HEX",
		"reality_dest":        "www.cloudflare.com:443",
		"flow":                "",
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create reality node status=%d body=%s", create.Code, create.Body.String())
	}
	var created struct {
		NodeID int64 `json:"node_id"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	node, err := st.FindNodeByID(context.Background(), created.NodeID)
	if err != nil {
		t.Fatalf("find node: %v", err)
	}
	if node.Flow != "" {
		t.Fatalf("create should preserve empty flow, got %q", node.Flow)
	}

	update := adminJSON(t, r, token, http.MethodPut, "/api/admin/v1/nodes/"+strconv.FormatInt(created.NodeID, 10), map[string]any{
		"name":                "美国 02",
		"host":                "us2.example.com",
		"port":                443,
		"protocol":            "vless",
		"transport":           "xhttp",
		"security":            "reality",
		"runtime_type":        "xray",
		"reality_server_name": "www.cloudflare.com",
		"reality_public_key":  "PBK",
		"reality_private_key": "PRIVATE-KEY-HEX",
		"reality_dest":        "www.cloudflare.com:443",
		"status":              "active",
		"flow":                "",
	})
	if update.Code != http.StatusOK {
		t.Fatalf("update reality node status=%d body=%s", update.Code, update.Body.String())
	}
	node, err = st.FindNodeByID(context.Background(), created.NodeID)
	if err != nil {
		t.Fatalf("find node after update: %v", err)
	}
	if node.Flow != "" {
		t.Fatalf("update should preserve empty flow, got %q", node.Flow)
	}
	if node.Transport != "xhttp" {
		t.Fatalf("update should preserve xhttp transport, got %q", node.Transport)
	}
}

func TestAdminCanGenerateRealityConfig(t *testing.T) {
	r, _, token := setupAdminCRUDRouter(t)

	resp := adminJSON(t, r, token, http.MethodPost, "/api/admin/v1/reality/generate", map[string]any{
		"server_name": "www.cloudflare.com",
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("generate reality config status=%d body=%s", resp.Code, resp.Body.String())
	}
	var got struct {
		RealityPublicKey  string `json:"reality_public_key"`
		RealityPrivateKey string `json:"reality_private_key"`
		RealityShortID    string `json:"reality_short_id"`
		RealityServerName string `json:"reality_server_name"`
		RealityDest       string `json:"reality_dest"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.RealityServerName != "www.cloudflare.com" || got.RealityDest != "www.cloudflare.com:443" {
		t.Fatalf("unexpected server/dest: %+v", got)
	}
	for name, value := range map[string]string{
		"public":  got.RealityPublicKey,
		"private": got.RealityPrivateKey,
	} {
		raw, err := base64.RawURLEncoding.DecodeString(value)
		if err != nil {
			t.Fatalf("%s key is not raw-url-base64: %q: %v", name, value, err)
		}
		if len(raw) != 32 {
			t.Fatalf("%s key length = %d, want 32", name, len(raw))
		}
	}
	if got.RealityPublicKey == got.RealityPrivateKey {
		t.Fatalf("public/private keys should differ")
	}
	shortID, err := hex.DecodeString(got.RealityShortID)
	if err != nil {
		t.Fatalf("short id is not hex: %q: %v", got.RealityShortID, err)
	}
	if len(shortID) != 8 {
		t.Fatalf("short id byte length = %d, want 8", len(shortID))
	}
}
