package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zboard/api-server/internal/authsvc"
	"github.com/zboard/api-server/internal/bizsvc"
	"github.com/zboard/api-server/internal/nodesvc"
	"github.com/zboard/api-server/internal/payment"
	"github.com/zboard/api-server/internal/payment/registry"
	"github.com/zboard/api-server/internal/store"
	"github.com/zboard/api-server/internal/testsupport"
	"github.com/zboard/api-server/internal/worker"
)

type fakePayPalProvider struct {
	name       string
	lastCreate payment.CreateRequest
	capture    *payment.CallbackData
}

func (p *fakePayPalProvider) Name() string { return p.name }

func (p *fakePayPalProvider) CreatePayment(_ context.Context, req payment.CreateRequest) (*payment.CreateResponse, error) {
	p.lastCreate = req
	return &payment.CreateResponse{
		ProviderOrderNo: "PAYPAL-ORDER-1",
		PayURL:          "https://paypal.example.com/checkout?token=PAYPAL-ORDER-1",
		RawResponse:     `{"id":"PAYPAL-ORDER-1"}`,
	}, nil
}

func (p *fakePayPalProvider) VerifyCallback(_ context.Context, _ map[string]string, _ []byte) (*payment.CallbackData, error) {
	return p.capture, nil
}

func (p *fakePayPalProvider) CaptureOrder(_ context.Context, _ string) (*payment.CallbackData, error) {
	return p.capture, nil
}

func TestCustomPayPalProviderUsesCaptureReturnRouteAndProviderName(t *testing.T) {
	ctx := context.Background()
	st := testsupport.NewStore(t)
	auth := authsvc.New(st, "setup-token", nil)
	reg := registry.New(st)
	prov := &fakePayPalProvider{name: "paypal_live"}
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

	if _, err := st.CreatePaymentProvider(ctx, store.CreatePaymentProviderInput{
		Name:         "paypal_live",
		DisplayName:  "PayPal Live",
		ProviderType: "paypal",
		ConfigJSON:   `{}`,
		Enabled:      1,
	}); err != nil {
		t.Fatalf("create provider row: %v", err)
	}
	planID, err := st.CreatePlan(ctx, store.CreatePlanInput{
		Name:         "paypal-plan",
		Price:        "9.00",
		DurationDays: 30,
		TrafficLimit: 1000,
		DeviceLimit:  1,
	})
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}
	userID, err := auth.RegisterUser(ctx, "paypal-live@example.com", "secret123")
	if err != nil {
		t.Fatalf("register user: %v", err)
	}
	token, _, err := auth.LoginUser(ctx, "paypal-live@example.com", "secret123")
	if err != nil {
		t.Fatalf("login user: %v", err)
	}
	order, err := bizsvc.New(st).CreateOrder(ctx, userID, planID, "monthly", "")
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := st.SetSettings(ctx, map[string]string{"site_url": "https://panel.example.test"}); err != nil {
		t.Fatalf("settings: %v", err)
	}

	pay := adminJSON(t, r, token, http.MethodPost, "/api/v1/orders/"+order.Order.OrderNo+"/pay?provider=paypal_live", nil)
	if pay.Code != http.StatusOK {
		t.Fatalf("pay status=%d body=%s", pay.Code, pay.Body.String())
	}
	if !strings.Contains(prov.lastCreate.ReturnURL, "/api/v1/payments/paypal/return?provider=paypal_live") {
		t.Fatalf("return_url=%q, want capture route with actual provider name", prov.lastCreate.ReturnURL)
	}
	prov.capture = &payment.CallbackData{
		ProviderOrderNo: "CAPTURE-1",
		OrderNo:         order.Order.OrderNo,
		Amount:          "9.00",
		Status:          "success",
		RawBody:         `{"id":"CAPTURE-1"}`,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/payments/paypal/return?provider=paypal_live&token=PAYPAL-ORDER-1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("return status=%d body=%s", rr.Code, rr.Body.String())
	}
	paid, err := st.FindOrderByNo(ctx, order.Order.OrderNo)
	if err != nil {
		t.Fatalf("find order: %v", err)
	}
	if paid.Status != "paid" {
		t.Fatalf("order status=%q, want paid", paid.Status)
	}
	payments, err := st.ListPayments(ctx, 10)
	if err != nil {
		t.Fatalf("list payments: %v", err)
	}
	if len(payments) != 1 || payments[0].Provider != "paypal_live" || payments[0].Status != "success" {
		raw, _ := json.Marshal(payments)
		t.Fatalf("unexpected payments: %s", raw)
	}
}
