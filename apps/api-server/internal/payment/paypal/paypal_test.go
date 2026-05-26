package paypal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zboard/api-server/internal/payment"
)

func TestCreatePaymentCreatesOrderAndReturnsApproveURL(t *testing.T) {
	var createBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/oauth2/token":
			if r.Method != http.MethodPost {
				t.Fatalf("token method = %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"access-token"}`))
		case "/v2/checkout/orders":
			if got := r.Header.Get("Authorization"); got != "Bearer access-token" {
				t.Fatalf("authorization = %q", got)
			}
			if err := json.NewDecoder(r.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"PAYPAL-ORDER-1","links":[{"rel":"approve","href":"https://www.paypal.com/checkoutnow?token=PAYPAL-ORDER-1"}]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	p := New(Config{ClientID: "client", ClientSecret: "secret", APIURL: ts.URL})
	resp, err := p.CreatePayment(context.Background(), payment.CreateRequest{
		OrderNo:   "ORD123",
		Amount:    "12.34",
		Currency:  "USD",
		Subject:   "Plan A",
		ReturnURL: "https://panel.example.com/api/v1/payments/paypal/return",
		CancelURL: "https://panel.example.com/dashboard",
	})
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}
	if resp.ProviderOrderNo != "PAYPAL-ORDER-1" || !strings.Contains(resp.PayURL, "paypal.com") {
		t.Fatalf("unexpected response: %+v", resp)
	}
	units := createBody["purchase_units"].([]any)
	unit := units[0].(map[string]any)
	if unit["reference_id"] != "ORD123" || unit["custom_id"] != "ORD123" {
		t.Fatalf("purchase unit missing order refs: %#v", unit)
	}
	experience := createBody["payment_source"].(map[string]any)["paypal"].(map[string]any)["experience_context"].(map[string]any)
	if experience["return_url"] != "https://panel.example.com/api/v1/payments/paypal/return" {
		t.Fatalf("return_url=%q", experience["return_url"])
	}
	if experience["cancel_url"] != "https://panel.example.com/dashboard" {
		t.Fatalf("cancel_url=%q, want frontend dashboard", experience["cancel_url"])
	}
}

func TestCaptureOrderNormalizesCompletedCapture(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/oauth2/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"access-token"}`))
		case "/v2/checkout/orders/PAYPAL-ORDER-1/capture":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"PAYPAL-ORDER-1","status":"COMPLETED","purchase_units":[{"reference_id":"ORD123","payments":{"captures":[{"id":"CAPTURE-1","status":"COMPLETED","amount":{"value":"12.34"}}]}}]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	p := New(Config{ClientID: "client", ClientSecret: "secret", APIURL: ts.URL}).(*paypalProvider)
	data, err := p.CaptureOrder(context.Background(), "PAYPAL-ORDER-1")
	if err != nil {
		t.Fatalf("CaptureOrder: %v", err)
	}
	if data.Status != "success" || data.OrderNo != "ORD123" || data.ProviderOrderNo != "CAPTURE-1" {
		t.Fatalf("unexpected data: %+v", data)
	}
}

func TestVerifyCallbackUsesPayPalVerificationAPI(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/oauth2/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"access-token"}`))
		case "/v1/notifications/verify-webhook-signature":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"verification_status":"SUCCESS"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	p := New(Config{ClientID: "client", ClientSecret: "secret", WebhookID: "WH-1", APIURL: ts.URL})
	body := []byte(`{"id":"WH-EVT-1","event_type":"PAYMENT.CAPTURE.COMPLETED","resource":{"id":"CAPTURE-1","status":"COMPLETED","custom_id":"ORD123","amount":{"value":"12.34"}}}`)
	data, err := p.VerifyCallback(context.Background(), map[string]string{
		"Paypal-Transmission-Id":   "tid",
		"Paypal-Transmission-Time": "time",
		"Paypal-Transmission-Sig":  "sig",
		"Paypal-Cert-Url":          "https://api.paypal.com/cert.pem",
		"Paypal-Auth-Algo":         "SHA256withRSA",
	}, body)
	if err != nil {
		t.Fatalf("VerifyCallback: %v", err)
	}
	if data.Status != "success" || data.OrderNo != "ORD123" || data.ProviderOrderNo != "WH-EVT-1" {
		t.Fatalf("unexpected data: %+v", data)
	}
}
