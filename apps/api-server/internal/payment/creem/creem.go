// Package creem implements the Creem payment provider.
//
// Creem API flow:
//   1. Server POSTs to /v1/checkouts to create a checkout session.
//   2. User is redirected to the returned checkout URL.
//   3. On payment completion, Creem POSTs a webhook to the configured URL.
//   4. Server verifies the webhook signature (HMAC-SHA256 of the raw body
//      using the webhook secret) and activates the user.
package creem

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zboard/api-server/internal/payment"
)

type Config struct {
	APIKey        string // Creem API key (Bearer token)
	WebhookSecret string // HMAC-SHA256 signing secret for webhooks
	APIURL        string // defaults to https://api.creem.io
}

type creemProvider struct {
	cfg    Config
	client *http.Client
}

func New(cfg Config) payment.Provider {
	if cfg.APIURL == "" {
		cfg.APIURL = "https://api.creem.io"
	}
	return &creemProvider{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *creemProvider) Name() string { return "creem" }

type checkoutRequest struct {
	Amount      int    `json:"amount"`       // in cents
	Currency    string `json:"currency"`     // usd / eur
	OrderID     string `json:"metadata_order_id,omitempty"`
	SuccessURL  string `json:"success_url"`
	CancelURL   string `json:"cancel_url,omitempty"`
	WebhookURL  string `json:"webhook_url,omitempty"`
	Description string `json:"description,omitempty"`
}

type checkoutResponse struct {
	ID          string `json:"id"`
	CheckoutURL string `json:"checkout_url"`
}

func (p *creemProvider) CreatePayment(ctx context.Context, req payment.CreateRequest) (*payment.CreateResponse, error) {
	amountCents, err := parseCents(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("creem: parse amount: %w", err)
	}
	currency := strings.ToLower(req.Currency)
	if currency == "cny" {
		currency = "usd" // Creem doesn't support CNY; caller should convert
	}

	body := checkoutRequest{
		Amount:      amountCents,
		Currency:    currency,
		OrderID:     req.OrderNo,
		SuccessURL:  req.ReturnURL,
		WebhookURL:  req.NotifyURL,
		Description: req.Subject,
	}
	raw, _ := json.Marshal(body)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.cfg.APIURL+"/v1/checkouts", strings.NewReader(string(raw)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("creem: request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("creem: %d %s", resp.StatusCode, string(respBody))
	}

	var cr checkoutResponse
	if err := json.Unmarshal(respBody, &cr); err != nil {
		return nil, fmt.Errorf("creem: decode response: %w", err)
	}
	return &payment.CreateResponse{
		ProviderOrderNo: cr.ID,
		PayURL:          cr.CheckoutURL,
		RawResponse:     string(respBody),
	}, nil
}

func (p *creemProvider) VerifyCallback(_ context.Context, headers map[string]string, body []byte) (*payment.CallbackData, error) {
	sig := headers["creem-signature"]
	if sig == "" {
		sig = headers["Creem-Signature"]
	}
	if sig == "" {
		return nil, fmt.Errorf("creem: missing Creem-Signature header")
	}

	mac := hmac.New(sha256.New, []byte(p.cfg.WebhookSecret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return nil, fmt.Errorf("creem: signature mismatch")
	}

	var event struct {
		Type string `json:"type"`
		Data struct {
			ID      string `json:"id"`
			OrderID string `json:"metadata_order_id"`
			Amount  int    `json:"amount"`
			Status  string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("creem: decode webhook: %w", err)
	}

	status := "pending"
	if event.Type == "payment.completed" || event.Data.Status == "completed" {
		status = "success"
	} else if event.Data.Status == "failed" || event.Data.Status == "cancelled" {
		status = "failed"
	}

	return &payment.CallbackData{
		ProviderOrderNo: event.Data.ID,
		OrderNo:         event.Data.OrderID,
		Amount:          fmt.Sprintf("%.2f", float64(event.Data.Amount)/100),
		Status:          status,
		RawBody:         string(body),
	}, nil
}

// parseCents converts "9.90" → 990.
func parseCents(s string) (int, error) {
	s = strings.TrimSpace(s)
	parts := strings.SplitN(s, ".", 2)
	whole := 0
	frac := 0
	if _, err := fmt.Sscanf(parts[0], "%d", &whole); err != nil {
		return 0, err
	}
	if len(parts) == 2 {
		f := parts[1]
		if len(f) == 1 {
			f += "0"
		} else if len(f) > 2 {
			f = f[:2]
		}
		if _, err := fmt.Sscanf(f, "%d", &frac); err != nil {
			return 0, err
		}
	}
	return whole*100 + frac, nil
}
