package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zboard/api-server/internal/authsvc"
	"github.com/zboard/api-server/internal/bizsvc"
	"github.com/zboard/api-server/internal/nodesvc"
	"github.com/zboard/api-server/internal/payment"
	"github.com/zboard/api-server/internal/payment/registry"
	"github.com/zboard/api-server/internal/store"
	"github.com/zboard/api-server/internal/testsupport"
	"github.com/zboard/api-server/internal/worker"
)

type securityPaymentProvider struct {
	create    payment.CreateRequest
	callback  *payment.CallbackData
	verifyErr error
}

func (p *securityPaymentProvider) Name() string { return "secure_provider" }

func (p *securityPaymentProvider) CreatePayment(_ context.Context, req payment.CreateRequest) (*payment.CreateResponse, error) {
	p.create = req
	return &payment.CreateResponse{
		ProviderOrderNo: "gateway-order-1",
		PayURL:          "https://pay.example.com/checkout/gateway-order-1",
		RawResponse:     `{"id":"gateway-order-1"}`,
	}, nil
}

func (p *securityPaymentProvider) VerifyCallback(_ context.Context, _ map[string]string, _ []byte) (*payment.CallbackData, error) {
	if p.verifyErr != nil {
		return nil, p.verifyErr
	}
	return p.callback, nil
}

func setupSecurityPaymentRouter(t *testing.T) (*store.Store, *registry.Registry, *securityPaymentProvider, http.Handler, string, string) {
	t.Helper()
	ctx := context.Background()
	st := testsupport.NewStore(t)
	auth := authsvc.New(st, "setup-token", nil)
	reg := registry.New(st)
	prov := &securityPaymentProvider{}
	reg.Register(prov)
	r := New(Deps{
		DB:       st.DB,
		Store:    st,
		Auth:     auth,
		Biz:      bizsvc.New(st),
		Nodes:    nodesvc.New(st),
		Worker:   worker.New(st),
		Payments: reg,
	})

	planID, err := st.CreatePlan(ctx, store.CreatePlanInput{
		Name:         "secure-plan",
		Price:        "9.00",
		DurationDays: 30,
		TrafficLimit: 1000,
		DeviceLimit:  1,
	})
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}
	userID, err := auth.RegisterUser(ctx, "secure-pay@example.com", "secret123")
	if err != nil {
		t.Fatalf("register user: %v", err)
	}
	token, _, err := auth.LoginUser(ctx, "secure-pay@example.com", "secret123")
	if err != nil {
		t.Fatalf("login user: %v", err)
	}
	order, err := bizsvc.New(st).CreateOrder(ctx, userID, planID, "monthly", "")
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	return st, reg, prov, r, token, order.Order.OrderNo
}

func seedPendingPayment(t *testing.T, st *store.Store, orderNo, provider string) {
	t.Helper()
	o, err := st.FindOrderByNo(context.Background(), orderNo)
	if err != nil {
		t.Fatalf("find order for payment seed: %v", err)
	}
	if _, err := st.CreatePayment(context.Background(), &store.Payment{
		PaymentNo: "PAY-" + provider + "-" + orderNo,
		OrderID:   o.ID,
		UserID:    o.UserID,
		Provider:  provider,
		Amount:    o.Amount,
		Status:    "pending",
	}); err != nil {
		t.Fatalf("seed pending payment: %v", err)
	}
}

func TestPayOrderRequiresExplicitRealProvider(t *testing.T) {
	st, _, _, r, token, orderNo := setupSecurityPaymentRouter(t)

	resp := adminJSON(t, r, token, http.MethodPost, "/api/v1/orders/"+orderNo+"/pay", nil)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("pay without provider status=%d body=%s", resp.Code, resp.Body.String())
	}

	payments, err := st.ListPayments(context.Background(), 10)
	if err != nil {
		t.Fatalf("list payments: %v", err)
	}
	if len(payments) != 0 {
		t.Fatalf("pay without provider should not create payment rows: %+v", payments)
	}
}

func TestPublicMockPaymentCallbackRouteIsNotRegistered(t *testing.T) {
	_, _, _, r, _, _ := setupSecurityPaymentRouter(t)

	resp := adminJSON(t, r, "", http.MethodPost, "/api/v1/payments/mock-callback", map[string]any{
		"event_id": "evt-1", "order_no": "ORD-1", "payment_no": "PAY-1",
	})
	if resp.Code != http.StatusNotFound {
		t.Fatalf("mock callback status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func TestPaymentCallbackVerifyErrorDoesNotLeakInternalDetail(t *testing.T) {
	_, _, prov, r, _, _ := setupSecurityPaymentRouter(t)
	prov.verifyErr = errors.New("expected sign should-not-leak")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/secure_provider/callback", strings.NewReader("bad-signature"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("callback status=%d body=%s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "should-not-leak") || strings.Contains(rr.Body.String(), "expected sign") {
		t.Fatalf("callback leaked verifier internals: %s", rr.Body.String())
	}
}

func TestPaymentCallbackRejectsAmountMismatch(t *testing.T) {
	st, _, prov, r, token, orderNo := setupSecurityPaymentRouter(t)
	if err := st.SetSettings(context.Background(), map[string]string{"site_url": "https://panel.example.test"}); err != nil {
		t.Fatalf("settings: %v", err)
	}

	pay := adminJSON(t, r, token, http.MethodPost, "/api/v1/orders/"+orderNo+"/pay?provider=secure_provider", nil)
	if pay.Code != http.StatusOK {
		t.Fatalf("create payment status=%d body=%s", pay.Code, pay.Body.String())
	}
	prov.callback = &payment.CallbackData{
		ProviderOrderNo: "gateway-order-1",
		OrderNo:         orderNo,
		Amount:          "0.01",
		Status:          "success",
		RawBody:         `{"amount":"0.01"}`,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/secure_provider/callback", strings.NewReader("paid"))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code == http.StatusOK {
		t.Fatalf("amount mismatch callback should fail, got status=%d body=%s", rr.Code, rr.Body.String())
	}
	paid, err := st.FindOrderByNo(context.Background(), orderNo)
	if err != nil {
		t.Fatalf("find order: %v", err)
	}
	if paid.Status != "pending" {
		t.Fatalf("amount mismatch must not pay order, got status=%q", paid.Status)
	}
}

func TestPaymentCreateUsesConfiguredPublicOrigin(t *testing.T) {
	st, _, prov, r, token, orderNo := setupSecurityPaymentRouter(t)
	if err := st.SetSettings(context.Background(), map[string]string{"site_url": "https://panel.example.test/app"}); err != nil {
		t.Fatalf("settings: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+orderNo+"/pay?provider=secure_provider", nil)
	req.Host = "evil.example.test"
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("create payment status=%d body=%s", rr.Code, rr.Body.String())
	}
	if strings.Contains(prov.create.NotifyURL, "evil.example.test") || strings.Contains(prov.create.ReturnURL, "evil.example.test") {
		t.Fatalf("payment URLs used request Host: notify=%q return=%q", prov.create.NotifyURL, prov.create.ReturnURL)
	}
	if prov.create.NotifyURL != "https://panel.example.test/api/v1/payments/secure_provider/callback" {
		t.Fatalf("notify URL=%q", prov.create.NotifyURL)
	}
	if prov.create.ReturnURL != "https://panel.example.test/dashboard" {
		t.Fatalf("return URL=%q", prov.create.ReturnURL)
	}
}

// expireOrderForTest forces an order's expiry into the past so callers can
// exercise the expired-order code paths without sleeping.
func expireOrderForTest(t *testing.T, st *store.Store, orderNo string) {
	t.Helper()
	q := st.Rebind(`UPDATE orders SET expired_at = ? WHERE order_no = ?`)
	if _, err := st.DB.ExecContext(context.Background(), q, time.Now().UTC().Add(-time.Minute), orderNo); err != nil {
		t.Fatalf("expire order: %v", err)
	}
}

func TestPayExpiredOrderIsRejectedBeforeCharging(t *testing.T) {
	st, _, prov, r, token, orderNo := setupSecurityPaymentRouter(t)
	if err := st.SetSettings(context.Background(), map[string]string{"site_url": "https://panel.example.test"}); err != nil {
		t.Fatalf("settings: %v", err)
	}
	expireOrderForTest(t, st, orderNo)

	resp := adminJSON(t, r, token, http.MethodPost, "/api/v1/orders/"+orderNo+"/pay?provider=secure_provider", nil)
	if resp.Code != http.StatusConflict {
		t.Fatalf("expired order pay status=%d body=%s", resp.Code, resp.Body.String())
	}
	// The gateway must never be hit: charging an expired order risks taking
	// money for a subscription that can't be activated.
	if prov.create.OrderNo != "" {
		t.Fatalf("expired order should not reach the gateway, got create=%+v", prov.create)
	}
	payments, err := st.ListPayments(context.Background(), 10)
	if err != nil {
		t.Fatalf("list payments: %v", err)
	}
	if len(payments) != 0 {
		t.Fatalf("expired order pay should not create payment rows: %+v", payments)
	}
}

// A real gateway can call back after the order's 30-minute window lapses. As
// long as the user actually initiated payment (a pending payment row exists),
// the late callback must still activate the plan instead of stranding a paid
// order as expired.
func TestExpiredOrderWithInitiatedPaymentStillActivatesOnCallback(t *testing.T) {
	st, _, prov, r, token, orderNo := setupSecurityPaymentRouter(t)
	if err := st.SetSettings(context.Background(), map[string]string{"site_url": "https://panel.example.test"}); err != nil {
		t.Fatalf("settings: %v", err)
	}

	pay := adminJSON(t, r, token, http.MethodPost, "/api/v1/orders/"+orderNo+"/pay?provider=secure_provider", nil)
	if pay.Code != http.StatusOK {
		t.Fatalf("create payment status=%d body=%s", pay.Code, pay.Body.String())
	}

	expireOrderForTest(t, st, orderNo)

	o, err := st.FindOrderByNo(context.Background(), orderNo)
	if err != nil {
		t.Fatalf("find order: %v", err)
	}
	prov.callback = &payment.CallbackData{
		ProviderOrderNo: "gateway-order-1",
		OrderNo:         orderNo,
		Amount:          o.Amount,
		Status:          "success",
		RawBody:         `{"amount":"` + o.Amount + `"}`,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/secure_provider/callback", strings.NewReader("paid"))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("late callback status=%d body=%s", rr.Code, rr.Body.String())
	}
	paid, err := st.FindOrderByNo(context.Background(), orderNo)
	if err != nil {
		t.Fatalf("find order: %v", err)
	}
	if paid.Status != "paid" {
		t.Fatalf("late callback after expiry should still pay the order, got status=%q", paid.Status)
	}
}

func TestPaymentCreateRequiresConfiguredPublicOrigin(t *testing.T) {
	st, _, prov, r, token, orderNo := setupSecurityPaymentRouter(t)
	// No site_url configured: gateway callback/return URLs would otherwise point
	// at an unreachable localhost address and strand the paid order.

	resp := adminJSON(t, r, token, http.MethodPost, "/api/v1/orders/"+orderNo+"/pay?provider=secure_provider", nil)
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing public origin pay status=%d body=%s", resp.Code, resp.Body.String())
	}
	if prov.create.OrderNo != "" {
		t.Fatalf("missing origin should not reach the gateway, got create=%+v", prov.create)
	}
	payments, err := st.ListPayments(context.Background(), 10)
	if err != nil {
		t.Fatalf("list payments: %v", err)
	}
	if len(payments) != 0 {
		t.Fatalf("missing origin should not create payment rows: %+v", payments)
	}
}
