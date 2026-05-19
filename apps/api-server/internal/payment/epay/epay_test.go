package epay

import (
	"context"
	"net/url"
	"testing"

	"github.com/zboard/api-server/internal/payment"
)

func TestCreatePaymentBuildsSignedURL(t *testing.T) {
	p := New(Config{
		APIURL:    "https://pay.example.com",
		PID:       "1001",
		SecretKey: "test-secret-key",
	})
	resp, err := p.CreatePayment(context.Background(), payment.CreateRequest{
		OrderNo:   "ZB20260519001",
		Amount:    "9.90",
		Currency:  "CNY",
		Subject:   "Plan A",
		PayType:   "alipay",
		NotifyURL: "https://api.example.com/api/v1/payments/epay/callback",
		ReturnURL: "https://panel.example.com/orders",
		ClientIP:  "1.2.3.4",
	})
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}
	if resp.PayURL == "" {
		t.Fatal("expected non-empty PayURL")
	}
	u, err := url.Parse(resp.PayURL)
	if err != nil {
		t.Fatalf("parse PayURL: %v", err)
	}
	q := u.Query()
	if q.Get("pid") != "1001" {
		t.Errorf("pid = %q", q.Get("pid"))
	}
	if q.Get("type") != "alipay" {
		t.Errorf("type = %q", q.Get("type"))
	}
	if q.Get("sign") == "" {
		t.Error("missing sign param")
	}
	if q.Get("out_trade_no") != "ZB20260519001" {
		t.Errorf("out_trade_no = %q", q.Get("out_trade_no"))
	}
}

func TestVerifyCallbackSuccess(t *testing.T) {
	key := "my-secret"
	p := New(Config{APIURL: "https://pay.example.com", PID: "1001", SecretKey: key}).(*epayProvider)

	// Build a fake callback body with a valid signature.
	params := map[string]string{
		"pid":          "1001",
		"trade_no":     "EP202605190001",
		"out_trade_no": "ZB20260519001",
		"type":         "alipay",
		"name":         "Plan A",
		"money":        "9.90",
		"trade_status": "TRADE_SUCCESS",
	}
	sig := sign(params, key)
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	values.Set("sign", sig)
	values.Set("sign_type", "MD5")

	data, err := p.VerifyCallback(context.Background(), nil, []byte(values.Encode()))
	if err != nil {
		t.Fatalf("VerifyCallback: %v", err)
	}
	if data.Status != "success" {
		t.Errorf("status = %q, want success", data.Status)
	}
	if data.OrderNo != "ZB20260519001" {
		t.Errorf("OrderNo = %q", data.OrderNo)
	}
	if data.ProviderOrderNo != "EP202605190001" {
		t.Errorf("ProviderOrderNo = %q", data.ProviderOrderNo)
	}
}

func TestVerifyCallbackBadSignature(t *testing.T) {
	p := New(Config{APIURL: "https://pay.example.com", PID: "1001", SecretKey: "real-key"})
	body := "pid=1001&trade_no=X&out_trade_no=Y&money=1.00&trade_status=TRADE_SUCCESS&sign=bad&sign_type=MD5"
	_, err := p.VerifyCallback(context.Background(), nil, []byte(body))
	if err == nil {
		t.Fatal("expected signature mismatch error")
	}
}
