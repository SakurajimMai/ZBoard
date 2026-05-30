package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/zboard/api-server/internal/agentauth"
	"github.com/zboard/api-server/internal/authsvc"
	"github.com/zboard/api-server/internal/authx"
	"github.com/zboard/api-server/internal/bizsvc"
	"github.com/zboard/api-server/internal/nodesvc"
	"github.com/zboard/api-server/internal/payment/registry"
	"github.com/zboard/api-server/internal/store"
	"github.com/zboard/api-server/internal/testsupport"
	"github.com/zboard/api-server/internal/worker"
)

func setupSecurityAgentRouter(t *testing.T) (http.Handler, *store.Store, int64, string, int64, string, int64) {
	t.Helper()
	ctx := context.Background()
	st := testsupport.NewStore(t)
	auth := authsvc.New(st, "setup-token", nil)
	r := New(Deps{
		DB:       st.DB,
		Store:    st,
		Auth:     auth,
		Biz:      bizsvc.New(st),
		Nodes:    nodesvc.New(st),
		Worker:   worker.New(st),
		Payments: registry.New(st),
	})

	node1, secret1, err := st.CreateNode(ctx, store.CreateNodeInput{
		Name: "agent-node-1", Host: "agent-1.example.com", Port: 443, Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("create node1: %v", err)
	}
	node2, secret2, err := st.CreateNode(ctx, store.CreateNodeInput{
		Name: "agent-node-2", Host: "agent-2.example.com", Port: 443, Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("create node2: %v", err)
	}
	userID, err := st.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "agent-security@example.com",
		PasswordHash: "hash",
		TrafficLimit: 1000,
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := st.EnsureNodeUserWithLimits(ctx, userID, node2, "11111111-1111-4111-8111-111111111111", "vless", 0, 1); err != nil {
		t.Fatalf("ensure node2 user: %v", err)
	}
	return r, st, node1, secret1, node2, secret2, userID
}

func signAgentRequest(t *testing.T, method, path, secret string, nodeID int64, body []byte, nonce string) http.Header {
	t.Helper()
	bodySum := sha256.Sum256(body)
	bodyHash := hex.EncodeToString(bodySum[:])
	secretHash := authx.HashToken(secret)
	ts := strconv.FormatInt(time.Now().UTC().Unix(), 10)
	msg := strconv.FormatInt(nodeID, 10) + "|" + ts + "|" + nonce + "|" + bodyHash + "|" + method + "|" + path
	mac := hmac.New(sha256.New, []byte(secretHash))
	mac.Write([]byte(msg))

	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set(agentauth.HeaderNodeID, strconv.FormatInt(nodeID, 10))
	h.Set(agentauth.HeaderTimestamp, ts)
	h.Set(agentauth.HeaderNonce, nonce)
	h.Set(agentauth.HeaderBodyHash, bodyHash)
	h.Set(agentauth.HeaderSignature, hex.EncodeToString(mac.Sum(nil)))
	return h
}

func signedAgentJSON(t *testing.T, r http.Handler, method, path, secret string, nodeID int64, nonce string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	headers := signAgentRequest(t, method, path, secret, nodeID, raw, nonce)
	for k, vs := range headers {
		req.Header[k] = vs
	}
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func TestAgentTrafficReportRejectsUserNotAssignedToSignedNode(t *testing.T) {
	r, st, node1, secret1, _, _, userID := setupSecurityAgentRouter(t)

	resp := signedAgentJSON(t, r, http.MethodPost, "/api/agent/v1/traffic/report", secret1, node1, "traffic-forged-1", map[string]any{
		"items": []map[string]any{{
			"user_id": userID, "upload_delta": 10, "download_delta": 5,
		}},
	})
	if resp.Code == http.StatusOK {
		t.Fatalf("forged traffic report should fail, body=%s", resp.Body.String())
	}

	u, err := st.FindUserByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("find user: %v", err)
	}
	if u.TrafficUsed != 0 {
		t.Fatalf("forged traffic report must not change traffic_used, got %d", u.TrafficUsed)
	}
}

func TestAgentTaskResultRejectsTaskFromAnotherNode(t *testing.T) {
	r, st, node1, secret1, node2, _, _ := setupSecurityAgentRouter(t)
	ctx := context.Background()
	if err := st.CreateNodeTask(ctx, "task-node2-only", node2, "sync_config", `{"version":"v1"}`); err != nil {
		t.Fatalf("create node2 task: %v", err)
	}

	resp := signedAgentJSON(t, r, http.MethodPost, "/api/agent/v1/tasks/task-node2-only/result", secret1, node1, "task-forged-1", map[string]any{
		"status": "success",
	})
	if resp.Code == http.StatusOK {
		t.Fatalf("forged task result should fail, body=%s", resp.Body.String())
	}

	task, err := st.FindNodeTaskByTaskID(ctx, "task-node2-only")
	if err != nil {
		t.Fatalf("find task: %v", err)
	}
	if task.Status == "success" {
		t.Fatalf("forged task result must not complete task: %+v", task)
	}
}

// TestAgentTrafficReportRejectsDisabledUser is the C7 regression: once a user's
// node_user row is disabled (e.g. the worker closed it for expiry/over-quota),
// the node that legitimately hosts that user must NOT be able to keep charging
// traffic to it. The signing node here genuinely owns the assignment — what's
// being tested is that the *disabled* state, not just node ownership, gates the
// allowlist.
func TestAgentTrafficReportRejectsDisabledUser(t *testing.T) {
	r, st, _, _, node2, secret2, userID := setupSecurityAgentRouter(t)
	ctx := context.Background()

	// Disable the user's node_user assignment, as the worker does on expiry.
	if err := st.SetNodeUserEnabledForUser(ctx, userID, 0); err != nil {
		t.Fatalf("disable node user: %v", err)
	}

	resp := signedAgentJSON(t, r, http.MethodPost, "/api/agent/v1/traffic/report", secret2, node2, "traffic-disabled-1", map[string]any{
		"items": []map[string]any{{
			"user_id": userID, "upload_delta": 100, "download_delta": 50,
		}},
	})
	if resp.Code == http.StatusOK {
		t.Fatalf("traffic report for disabled user should fail, body=%s", resp.Body.String())
	}

	u, err := st.FindUserByID(ctx, userID)
	if err != nil {
		t.Fatalf("find user: %v", err)
	}
	if u.TrafficUsed != 0 {
		t.Fatalf("disabled-user traffic must not be recorded, got traffic_used=%d", u.TrafficUsed)
	}
}

// TestAgentTrafficReportAcceptsEnabledUser is the positive control: with the
// assignment enabled, the owning node's report is accepted. This guards against
// an over-tight C7 fix that would break legitimate accounting.
func TestAgentTrafficReportAcceptsEnabledUser(t *testing.T) {
	r, st, _, _, node2, secret2, userID := setupSecurityAgentRouter(t)
	ctx := context.Background()

	resp := signedAgentJSON(t, r, http.MethodPost, "/api/agent/v1/traffic/report", secret2, node2, "traffic-ok-1", map[string]any{
		"items": []map[string]any{{
			"user_id": userID, "upload_delta": 100, "download_delta": 50,
		}},
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("traffic report for enabled user should succeed, code=%d body=%s", resp.Code, resp.Body.String())
	}

	u, err := st.FindUserByID(ctx, userID)
	if err != nil {
		t.Fatalf("find user: %v", err)
	}
	if u.TrafficUsed != 150 {
		t.Fatalf("enabled-user traffic should be recorded, got traffic_used=%d want 150", u.TrafficUsed)
	}
}
