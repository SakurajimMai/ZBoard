package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/zboard/api-server/internal/authsvc"
	"github.com/zboard/api-server/internal/bizsvc"
	"github.com/zboard/api-server/internal/store"
)

func TestCreateQuarterlyPlanOrderUsesQuarterlyPriceAndGrantsQuarterlyQuota(t *testing.T) {
	r, st, _ := setupAdminCRUDRouter(t)
	ctx := t.Context()
	auth := authsvc.New(st, "setup-token", nil)

	planID, err := st.CreatePlan(ctx, store.CreatePlanInput{
		Name:           "quarterly",
		Price:          "10.00",
		QuarterlyPrice: "27.00",
		YearlyPrice:    "96.00",
		DurationDays:   30,
		TrafficLimit:   1000,
		DeviceLimit:    3,
	})
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}
	userID, err := auth.RegisterUser(ctx, "quarterly@example.com", "secret123")
	if err != nil {
		t.Fatalf("register user: %v", err)
	}
	token, _, err := auth.LoginUser(ctx, "quarterly@example.com", "secret123")
	if err != nil {
		t.Fatalf("login user: %v", err)
	}

	resp := adminJSON(t, r, token, http.MethodPost, "/api/v1/orders", map[string]any{
		"plan_id": planID,
		"period":  "quarterly",
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("create order status=%d body=%s", resp.Code, resp.Body.String())
	}
	var got struct {
		Order store.Order `json:"order"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Order.BillingPeriod != "quarterly" {
		t.Fatalf("billing_period=%q, want quarterly", got.Order.BillingPeriod)
	}
	if got.Order.Amount != "27.00" || got.Order.OriginalAmount != "27.00" || got.Order.CreditAmount != "0.00" {
		t.Fatalf("unexpected quarterly order amount: %+v", got.Order)
	}

	seedPendingPayment(t, st, got.Order.OrderNo, "epay")
	if err := bizsvc.New(st).ActivateByCallback(ctx, got.Order.OrderNo, "epay", "trade-quarterly", got.Order.Amount); err != nil {
		t.Fatalf("activate callback: %v", err)
	}
	u, err := st.FindUserByID(ctx, userID)
	if err != nil {
		t.Fatalf("find user: %v", err)
	}
	if u.PlanPeriod != "quarterly" {
		t.Fatalf("plan_period=%q, want quarterly", u.PlanPeriod)
	}
	if u.TrafficLimit != 3000 {
		t.Fatalf("traffic_limit=%d, want 3000", u.TrafficLimit)
	}
	if u.ExpiredAt == nil || time.Until(*u.ExpiredAt) < 89*24*time.Hour {
		t.Fatalf("expired_at should be roughly 90 days later: %v", u.ExpiredAt)
	}
}

func TestUpgradePlanOrderCreditsRemainingTimeAndUnusedTraffic(t *testing.T) {
	r, st, _ := setupAdminCRUDRouter(t)
	ctx := t.Context()
	auth := authsvc.New(st, "setup-token", nil)

	oldPlanID, err := st.CreatePlan(ctx, store.CreatePlanInput{
		Name:         "old",
		Price:        "30.00",
		DurationDays: 30,
		TrafficLimit: 3000,
		DeviceLimit:  3,
	})
	if err != nil {
		t.Fatalf("create old plan: %v", err)
	}
	newPlanID, err := st.CreatePlan(ctx, store.CreatePlanInput{
		Name:         "new",
		Price:        "60.00",
		DurationDays: 30,
		TrafficLimit: 6000,
		DeviceLimit:  5,
	})
	if err != nil {
		t.Fatalf("create new plan: %v", err)
	}
	userID, err := auth.RegisterUser(ctx, "upgrade@example.com", "secret123")
	if err != nil {
		t.Fatalf("register user: %v", err)
	}
	token, _, err := auth.LoginUser(ctx, "upgrade@example.com", "secret123")
	if err != nil {
		t.Fatalf("login user: %v", err)
	}
	expiredAt := time.Now().UTC().Add(15 * 24 * time.Hour)
	if err := st.AdminUpdateUser(ctx, userID, store.AdminUpdateUserInput{
		Email:        "upgrade@example.com",
		Balance:      "0.00",
		PlanID:       &oldPlanID,
		ExpiredAt:    &expiredAt,
		TrafficLimit: 3000,
		TrafficUsed:  1500,
		Status:       "active",
	}); err != nil {
		t.Fatalf("update user: %v", err)
	}

	resp := adminJSON(t, r, token, http.MethodPost, "/api/v1/orders", map[string]any{
		"plan_id": newPlanID,
		"period":  "monthly",
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("create upgrade order status=%d body=%s", resp.Code, resp.Body.String())
	}
	var got struct {
		Order store.Order `json:"order"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Order.Amount != "52.50" || got.Order.OriginalAmount != "60.00" || got.Order.CreditAmount != "7.50" {
		t.Fatalf("unexpected upgrade order amount: %+v", got.Order)
	}

	beforePay := time.Now().UTC()
	seedPendingPayment(t, st, got.Order.OrderNo, "epay")
	if err := bizsvc.New(st).ActivateByCallback(ctx, got.Order.OrderNo, "epay", "trade-upgrade", got.Order.Amount); err != nil {
		t.Fatalf("activate callback: %v", err)
	}
	u, err := st.FindUserByID(ctx, userID)
	if err != nil {
		t.Fatalf("find user: %v", err)
	}
	if u.PlanID == nil || *u.PlanID != newPlanID {
		t.Fatalf("plan_id=%v, want %d", u.PlanID, newPlanID)
	}
	if u.TrafficLimit != 6000 || u.TrafficUsed != 0 {
		t.Fatalf("traffic should reset to new monthly quota: %+v", u)
	}
	if u.ExpiredAt == nil {
		t.Fatalf("expired_at not set")
	}
	if u.ExpiredAt.Before(beforePay.Add(29*24*time.Hour)) || u.ExpiredAt.After(beforePay.Add(31*24*time.Hour)) {
		t.Fatalf("upgrade should start a fresh monthly cycle, got %v", u.ExpiredAt)
	}
}
