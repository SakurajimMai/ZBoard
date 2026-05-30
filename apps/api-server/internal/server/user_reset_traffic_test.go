package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/zboard/api-server/internal/authsvc"
	"github.com/zboard/api-server/internal/bizsvc"
	"github.com/zboard/api-server/internal/store"
)

func TestUserResetTrafficCreatesPaymentOrderWithoutBalance(t *testing.T) {
	r, st, _ := setupAdminCRUDRouter(t)
	ctx := context.Background()
	auth := authsvc.New(st, "setup-token", nil)

	planID, err := st.CreatePlan(ctx, store.CreatePlanInput{
		Name:              "resettable",
		Price:             "20.00",
		ResetTrafficPrice: "5.00",
		DurationDays:      30,
		TrafficLimit:      1000,
		DeviceLimit:       3,
	})
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}
	userID, err := auth.RegisterUser(ctx, "reset-order@example.com", "secret123")
	if err != nil {
		t.Fatalf("register user: %v", err)
	}
	token, _, err := auth.LoginUser(ctx, "reset-order@example.com", "secret123")
	if err != nil {
		t.Fatalf("login user: %v", err)
	}
	expiredAt := time.Now().UTC().Add(24 * time.Hour)
	if err := st.AdminUpdateUser(ctx, userID, store.AdminUpdateUserInput{
		Email:        "reset-order@example.com",
		Balance:      "0.00",
		PlanID:       &planID,
		ExpiredAt:    &expiredAt,
		TrafficLimit: 1000,
		TrafficUsed:  700,
		Status:       "active",
	}); err != nil {
		t.Fatalf("update user: %v", err)
	}

	resp := adminJSON(t, r, token, http.MethodPost, "/api/v1/traffic/reset", nil)
	if resp.Code != http.StatusCreated {
		t.Fatalf("reset traffic order status=%d body=%s", resp.Code, resp.Body.String())
	}
	var got struct {
		Order store.Order `json:"order"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Order.Kind != store.OrderKindTrafficReset {
		t.Fatalf("order kind=%q, want %q", got.Order.Kind, store.OrderKindTrafficReset)
	}
	amount, err := strconv.ParseFloat(got.Order.Amount, 64)
	if err != nil {
		t.Fatalf("parse order amount %q: %v", got.Order.Amount, err)
	}
	if amount != 5 || got.Order.PlanID != planID || got.Order.Status != "pending" {
		t.Fatalf("unexpected reset order: %+v", got.Order)
	}
	u, err := st.FindUserByID(ctx, userID)
	if err != nil {
		t.Fatalf("find user: %v", err)
	}
	if u.Balance != "0.00" || u.TrafficUsed != 700 {
		t.Fatalf("reset order should not deduct balance or clear traffic before payment: %+v", u)
	}
}

func TestTrafficResetPaymentCallbackClearsTraffic(t *testing.T) {
	_, st, _ := setupAdminCRUDRouter(t)
	ctx := context.Background()

	planID, err := st.CreatePlan(ctx, store.CreatePlanInput{
		Name:              "reset-callback",
		Price:             "20.00",
		ResetTrafficPrice: "5.00",
		DurationDays:      30,
		TrafficLimit:      1000,
		DeviceLimit:       3,
	})
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}
	expiredAt := time.Now().UTC().Add(24 * time.Hour)
	userID, err := st.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "reset-callback@example.com",
		PasswordHash: "hash",
		Balance:      "0.00",
		PlanID:       &planID,
		ExpiredAt:    &expiredAt,
		TrafficLimit: 1000,
		TrafficUsed:  700,
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	orderNo := "RT202605250001"
	order := &store.Order{
		OrderNo:   orderNo,
		UserID:    userID,
		PlanID:    planID,
		Kind:      store.OrderKindTrafficReset,
		Amount:    "5.00",
		Currency:  "CNY",
		Status:    "pending",
		ExpiredAt: ptrTime(time.Now().UTC().Add(30 * time.Minute)),
	}
	if _, err := st.CreateOrder(ctx, order); err != nil {
		t.Fatalf("create reset order: %v", err)
	}

	seedPendingPayment(t, st, orderNo, "epay")
	if err := bizsvc.New(st).ActivateByCallback(ctx, orderNo, "epay", "trade-reset-1", order.Amount); err != nil {
		t.Fatalf("activate callback: %v", err)
	}
	u, err := st.FindUserByID(ctx, userID)
	if err != nil {
		t.Fatalf("find user: %v", err)
	}
	if u.TrafficUsed != 0 {
		t.Fatalf("traffic_used=%d, want 0", u.TrafficUsed)
	}
	paid, err := st.FindOrderByNo(ctx, orderNo)
	if err != nil {
		t.Fatalf("find order: %v", err)
	}
	if paid.Status != "paid" {
		t.Fatalf("order status=%q, want paid", paid.Status)
	}
}

func TestTrafficResetPaymentCallbackRestoresSkippedNodeMapping(t *testing.T) {
	r, st, adminToken := setupAdminCRUDRouter(t)
	ctx := context.Background()

	planID, err := st.CreatePlan(ctx, store.CreatePlanInput{
		Name:              "reset-restores-node",
		Price:             "20.00",
		ResetTrafficPrice: "5.00",
		DurationDays:      30,
		TrafficLimit:      1000,
		DeviceLimit:       2,
	})
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}
	expiredAt := time.Now().UTC().Add(24 * time.Hour)
	userID, err := st.AdminCreateUser(ctx, store.AdminCreateUserInput{
		Email:        "reset-restore-node@example.com",
		PasswordHash: "hash",
		Balance:      "0.00",
		PlanID:       &planID,
		ExpiredAt:    &expiredAt,
		TrafficLimit: 1000,
		TrafficUsed:  1000,
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	existingNodeID, _, err := st.CreateNode(ctx, store.CreateNodeInput{
		Name:     "existing-node",
		Host:     "existing.example.com",
		Port:     443,
		Protocol: "vless",
	})
	if err != nil {
		t.Fatalf("create existing node: %v", err)
	}
	existingClientID := "11111111-1111-4111-8111-111111111111"
	if err := st.EnsureNodeUserWithLimits(ctx, userID, existingNodeID, existingClientID, "vless", 0, 2); err != nil {
		t.Fatalf("seed existing node user: %v", err)
	}

	createNode := adminJSON(t, r, adminToken, http.MethodPost, "/api/admin/v1/nodes", map[string]any{
		"name":         "new-node-while-over-quota",
		"host":         "new.example.com",
		"port":         443,
		"protocol":     "vless",
		"transport":    "tcp",
		"security":     "tls",
		"runtime_type": "xray",
	})
	if createNode.Code != http.StatusCreated {
		t.Fatalf("create node status=%d body=%s", createNode.Code, createNode.Body.String())
	}
	var created struct {
		NodeID int64 `json:"node_id"`
	}
	if err := json.Unmarshal(createNode.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created node: %v", err)
	}
	if _, err := st.FindNodeUser(ctx, userID, created.NodeID); !store.IsNoRows(err) {
		t.Fatalf("over-quota user should be skipped for new node, err=%v", err)
	}

	orderNo := "RT202605250002"
	order := &store.Order{
		OrderNo:   orderNo,
		UserID:    userID,
		PlanID:    planID,
		Kind:      store.OrderKindTrafficReset,
		Amount:    "5.00",
		Currency:  "CNY",
		Status:    "pending",
		ExpiredAt: ptrTime(time.Now().UTC().Add(30 * time.Minute)),
	}
	if _, err := st.CreateOrder(ctx, order); err != nil {
		t.Fatalf("create reset order: %v", err)
	}

	seedPendingPayment(t, st, orderNo, "epay")
	if err := bizsvc.New(st).ActivateByCallback(ctx, orderNo, "epay", "trade-reset-2", order.Amount); err != nil {
		t.Fatalf("activate callback: %v", err)
	}
	nu, err := st.FindNodeUser(ctx, userID, created.NodeID)
	if err != nil {
		t.Fatalf("find restored node user: %v", err)
	}
	if nu.Enabled != 1 || nu.ClientID != existingClientID || nu.DeviceLimit != 2 {
		t.Fatalf("unexpected restored node user: %+v", nu)
	}
}
