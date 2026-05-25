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

	if err := bizsvc.New(st).ActivateByCallback(ctx, orderNo, "epay", "trade-reset-1"); err != nil {
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
