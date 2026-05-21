package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

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
		DB:       st.DB,
		Store:    st,
		Auth:     auth,
		Biz:      bizsvc.New(st),
		Nodes:    nodesvc.New(st),
		Worker:   worker.New(st),
		Payments: registry.New(st),
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

func TestAdminCanUpdatePlanAndNode(t *testing.T) {
	r, st, token := setupAdminCRUDRouter(t)

	planCreate := adminJSON(t, r, token, http.MethodPost, "/api/admin/v1/plans", map[string]any{
		"name":          "基础套餐",
		"price":         "9.90",
		"duration_days": 30,
		"traffic_limit": int64(100 * 1024 * 1024 * 1024),
		"device_limit":  2,
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
		"speed_limit":   100,
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
