package stripe

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/zboard/api-server/internal/payment"
)

func TestCreatePaymentBuildsCheckoutSession(t *testing.T) {
	var form url.Values
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/checkout/sessions" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk_test" {
			t.Fatalf("authorization = %q", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		form = r.Form
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"cs_test_123","url":"https://checkout.stripe.com/c/pay","payment_intent":"pi_123"}`))
	}))
	defer ts.Close()

	p := New(Config{SecretKey: "sk_test", WebhookSecret: "whsec_test", APIURL: ts.URL})
	resp, err := p.CreatePayment(context.Background(), payment.CreateRequest{
		OrderNo:   "ORD123",
		Amount:    "9.90",
		Currency:  "USD",
		Subject:   "Plan A",
		ReturnURL: "https://panel.example.com/dashboard",
	})
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}
	if resp.ProviderOrderNo != "pi_123" || !strings.Contains(resp.PayURL, "checkout.stripe.com") {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if form.Get("client_reference_id") != "ORD123" {
		t.Fatalf("client_reference_id = %q", form.Get("client_reference_id"))
	}
	if form.Get("line_items[0][price_data][unit_amount]") != "990" {
		t.Fatalf("unit_amount = %q", form.Get("line_items[0][price_data][unit_amount]"))
	}
}

func TestVerifyCallbackSuccess(t *testing.T) {
	body := []byte(`{"id":"evt_123","type":"checkout.session.completed","data":{"object":{"id":"cs_123","payment_intent":"pi_123","client_reference_id":"ORD123","payment_status":"paid","amount_total":990,"metadata":{"order_no":"ORD123"}}}}`)
	secret := "whsec_test"
	timestamp := "1710000000"
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%s.%s", timestamp, string(body))))
	sig := hex.EncodeToString(mac.Sum(nil))

	p := New(Config{SecretKey: "sk_test", WebhookSecret: secret})
	data, err := p.VerifyCallback(context.Background(), map[string]string{
		"Stripe-Signature": "t=" + timestamp + ",v1=" + sig,
	}, body)
	if err != nil {
		t.Fatalf("VerifyCallback: %v", err)
	}
	if data.Status != "success" || data.OrderNo != "ORD123" || data.ProviderOrderNo != "evt_123" {
		t.Fatalf("unexpected data: %+v", data)
	}
}

func TestVerifyCallbackRejectsBadSignature(t *testing.T) {
	p := New(Config{SecretKey: "sk_test", WebhookSecret: "whsec_test"})
	_, err := p.VerifyCallback(context.Background(), map[string]string{
		"Stripe-Signature": "t=1710000000,v1=bad",
	}, []byte(`{"id":"evt_bad"}`))
	if err == nil {
		t.Fatal("expected signature error")
	}
}
