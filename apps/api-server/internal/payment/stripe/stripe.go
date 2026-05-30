// Package stripe implements Stripe Checkout payment sessions.
package stripe

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/zboard/api-server/internal/payment"
)

type Config struct {
	SecretKey     string
	WebhookSecret string
	APIURL        string
}

type stripeProvider struct {
	cfg    Config
	client *http.Client
}

func New(cfg Config) payment.Provider {
	if strings.TrimSpace(cfg.APIURL) == "" {
		cfg.APIURL = "https://api.stripe.com"
	}
	cfg.APIURL = strings.TrimRight(cfg.APIURL, "/")
	return &stripeProvider{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *stripeProvider) Name() string { return "stripe" }

func (p *stripeProvider) CreatePayment(ctx context.Context, req payment.CreateRequest) (*payment.CreateResponse, error) {
	amount, err := parseMinorAmount(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("stripe: parse amount: %w", err)
	}
	currency := strings.ToLower(strings.TrimSpace(req.Currency))
	if currency == "" {
		currency = "usd"
	}

	values := url.Values{}
	values.Set("mode", "payment")
	values.Set("success_url", req.ReturnURL)
	values.Set("cancel_url", req.ReturnURL)
	values.Set("client_reference_id", req.OrderNo)
	values.Set("metadata[order_no]", req.OrderNo)
	values.Set("payment_intent_data[metadata][order_no]", req.OrderNo)
	values.Set("line_items[0][quantity]", "1")
	values.Set("line_items[0][price_data][currency]", currency)
	values.Set("line_items[0][price_data][unit_amount]", strconv.FormatInt(amount, 10))
	values.Set("line_items[0][price_data][product_data][name]", req.Subject)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.APIURL+"/v1/checkout/sessions", strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.SecretKey)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("stripe: request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("stripe: %d %s", resp.StatusCode, string(respBody))
	}

	var out struct {
		ID            string `json:"id"`
		URL           string `json:"url"`
		PaymentIntent string `json:"payment_intent"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("stripe: decode response: %w", err)
	}
	if out.URL == "" {
		return nil, fmt.Errorf("stripe: checkout session missing url")
	}
	providerOrderNo := out.PaymentIntent
	if providerOrderNo == "" {
		providerOrderNo = out.ID
	}
	return &payment.CreateResponse{
		ProviderOrderNo: providerOrderNo,
		PayURL:          out.URL,
		RawResponse:     string(respBody),
	}, nil
}

func (p *stripeProvider) VerifyCallback(_ context.Context, headers map[string]string, body []byte) (*payment.CallbackData, error) {
	if err := verifyStripeSignature(headers, body, p.cfg.WebhookSecret); err != nil {
		return nil, err
	}

	var event struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		Data struct {
			Object struct {
				ID              string            `json:"id"`
				PaymentIntent   string            `json:"payment_intent"`
				ClientReference string            `json:"client_reference_id"`
				PaymentStatus   string            `json:"payment_status"`
				Status          string            `json:"status"`
				AmountTotal     int64             `json:"amount_total"`
				Metadata        map[string]string `json:"metadata"`
			} `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("stripe: decode webhook: %w", err)
	}

	obj := event.Data.Object
	orderNo := obj.Metadata["order_no"]
	if orderNo == "" {
		orderNo = obj.ClientReference
	}
	providerOrderNo := obj.PaymentIntent
	if providerOrderNo == "" {
		providerOrderNo = obj.ID
	}
	status := "pending"
	if event.Type == "checkout.session.completed" && obj.PaymentStatus == "paid" {
		status = "success"
	} else if obj.Status == "expired" || event.Type == "checkout.session.expired" {
		status = "failed"
	}

	eventID := event.ID
	if eventID == "" {
		eventID = providerOrderNo
	}
	return &payment.CallbackData{
		ProviderOrderNo: firstNonEmpty(eventID, providerOrderNo),
		OrderNo:         orderNo,
		Amount:          fmt.Sprintf("%.2f", float64(obj.AmountTotal)/100),
		Status:          status,
		RawBody:         string(body),
	}, nil
}

// stripeSignatureToleranceSeconds is the maximum allowed clock drift between
// Stripe's signed timestamp and our local clock. Stripe's official client uses
// 5 minutes; we match that to reject replayed webhooks beyond the window.
const stripeSignatureToleranceSeconds = 300

func verifyStripeSignature(headers map[string]string, body []byte, secret string) error {
	if secret == "" {
		return fmt.Errorf("stripe: webhook_secret is required")
	}
	sigHeader := headerValue(headers, "Stripe-Signature")
	if sigHeader == "" {
		return fmt.Errorf("stripe: missing Stripe-Signature header")
	}
	var timestamp string
	signatures := []string{}
	for _, part := range strings.Split(sigHeader, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch key {
		case "t":
			timestamp = value
		case "v1":
			signatures = append(signatures, value)
		}
	}
	if timestamp == "" || len(signatures) == 0 {
		return fmt.Errorf("stripe: malformed Stripe-Signature header")
	}
	tsInt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("stripe: malformed timestamp")
	}
	now := time.Now().Unix()
	if delta := now - tsInt; delta > stripeSignatureToleranceSeconds || delta < -stripeSignatureToleranceSeconds {
		return fmt.Errorf("stripe: timestamp outside tolerance window")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	for _, sig := range signatures {
		if hmac.Equal([]byte(expected), []byte(sig)) {
			return nil
		}
	}
	return fmt.Errorf("stripe: signature mismatch")
}

func parseMinorAmount(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty amount")
	}
	parts := strings.SplitN(s, ".", 2)
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	frac := "00"
	if len(parts) == 2 {
		frac = parts[1]
		if len(frac) == 1 {
			frac += "0"
		}
		if len(frac) > 2 {
			frac = frac[:2]
		}
	}
	cents, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, err
	}
	return whole*100 + cents, nil
}

func headerValue(headers map[string]string, key string) string {
	if headers == nil {
		return ""
	}
	if v := headers[key]; v != "" {
		return v
	}
	lower := strings.ToLower(key)
	for k, v := range headers {
		if strings.ToLower(k) == lower {
			return v
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
