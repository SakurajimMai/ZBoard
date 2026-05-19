// Package nowpay implements the NOWPayments crypto payment provider.
//
// NOWPayments API flow:
//   1. Server POSTs to /v1/payment to create a crypto payment.
//   2. User is shown the payment address / QR code (or redirected to
//      NOWPayments hosted invoice page).
//   3. On status change, NOWPayments POSTs an IPN (Instant Payment
//      Notification) to the configured callback URL.
//   4. Server verifies the IPN signature (HMAC-SHA512 of sorted JSON body
//      using the IPN secret) and activates the user when status is
//      "finished" or "confirmed".
package nowpay

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/zboard/api-server/internal/payment"
)

type Config struct {
	APIKey    string // x-api-key header
	IPNSecret string // HMAC-SHA512 key for IPN verification
	APIURL    string // defaults to https://api.nowpayments.io
}

type nowProvider struct {
	cfg    Config
	client *http.Client
}

func New(cfg Config) payment.Provider {
	if cfg.APIURL == "" {
		cfg.APIURL = "https://api.nowpayments.io"
	}
	return &nowProvider{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *nowProvider) Name() string { return "nowpayments" }

type createPaymentReq struct {
	PriceAmount      float64 `json:"price_amount"`
	PriceCurrency    string  `json:"price_currency"`
	PayCurrency      string  `json:"pay_currency"`
	OrderID          string  `json:"order_id"`
	OrderDescription string  `json:"order_description"`
	IPNURL           string  `json:"ipn_callback_url"`
}

type createPaymentResp struct {
	PaymentID      int64   `json:"payment_id"`
	PayAddress     string  `json:"pay_address"`
	PayAmount      float64 `json:"pay_amount"`
	PayCurrency    string  `json:"pay_currency"`
	InvoiceURL     string  `json:"invoice_url"`
	PaymentStatus  string  `json:"payment_status"`
}

func (p *nowProvider) CreatePayment(ctx context.Context, req payment.CreateRequest) (*payment.CreateResponse, error) {
	priceAmount := parseFloat(req.Amount)
	payCurrency := strings.ToLower(req.PayType) // e.g. "btc", "eth", "usdttrc20"
	if payCurrency == "" || payCurrency == "crypto" {
		payCurrency = "usdttrc20"
	}
	priceCurrency := strings.ToLower(req.Currency)
	if priceCurrency == "cny" {
		priceCurrency = "usd"
	}

	body := createPaymentReq{
		PriceAmount:      priceAmount,
		PriceCurrency:    priceCurrency,
		PayCurrency:      payCurrency,
		OrderID:          req.OrderNo,
		OrderDescription: req.Subject,
		IPNURL:           req.NotifyURL,
	}
	raw, _ := json.Marshal(body)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.cfg.APIURL+"/v1/payment", strings.NewReader(string(raw)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.cfg.APIKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("nowpay: request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("nowpay: %d %s", resp.StatusCode, string(respBody))
	}

	var cr createPaymentResp
	if err := json.Unmarshal(respBody, &cr); err != nil {
		return nil, fmt.Errorf("nowpay: decode: %w", err)
	}

	payURL := cr.InvoiceURL
	if payURL == "" {
		payURL = fmt.Sprintf("https://nowpayments.io/payment/?iid=%d", cr.PaymentID)
	}

	return &payment.CreateResponse{
		ProviderOrderNo: fmt.Sprintf("%d", cr.PaymentID),
		PayURL:          payURL,
		QRCode:          cr.PayAddress,
		RawResponse:     string(respBody),
	}, nil
}

func (p *nowProvider) VerifyCallback(_ context.Context, headers map[string]string, body []byte) (*payment.CallbackData, error) {
	sig := headers["x-nowpayments-sig"]
	if sig == "" {
		sig = headers["X-Nowpayments-Sig"]
	}
	if sig == "" {
		return nil, fmt.Errorf("nowpay: missing x-nowpayments-sig header")
	}

	// NOWPayments signs the sorted JSON body with HMAC-SHA512.
	sorted, err := sortedJSON(body)
	if err != nil {
		return nil, fmt.Errorf("nowpay: sort json: %w", err)
	}
	mac := hmac.New(sha512.New, []byte(p.cfg.IPNSecret))
	mac.Write(sorted)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return nil, fmt.Errorf("nowpay: signature mismatch")
	}

	var ipn struct {
		PaymentID     int64   `json:"payment_id"`
		OrderID       string  `json:"order_id"`
		PaymentStatus string  `json:"payment_status"`
		PriceAmount   float64 `json:"price_amount"`
		PayAmount     float64 `json:"pay_amount"`
	}
	if err := json.Unmarshal(body, &ipn); err != nil {
		return nil, fmt.Errorf("nowpay: decode ipn: %w", err)
	}

	status := "pending"
	switch ipn.PaymentStatus {
	case "finished", "confirmed":
		status = "success"
	case "failed", "expired", "refunded":
		status = "failed"
	case "partially_paid":
		status = "pending"
	}

	return &payment.CallbackData{
		ProviderOrderNo: fmt.Sprintf("%d", ipn.PaymentID),
		OrderNo:         ipn.OrderID,
		Amount:          fmt.Sprintf("%.2f", ipn.PriceAmount),
		Status:          status,
		RawBody:         string(body),
	}, nil
}

// sortedJSON re-encodes the JSON with keys sorted alphabetically at every
// level. NOWPayments requires this for signature verification.
func sortedJSON(data []byte) ([]byte, error) {
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	return marshalSorted(obj)
}

func marshalSorted(obj map[string]any) ([]byte, error) {
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		kb, _ := json.Marshal(k)
		b.Write(kb)
		b.WriteByte(':')
		v := obj[k]
		if m, ok := v.(map[string]any); ok {
			vb, err := marshalSorted(m)
			if err != nil {
				return nil, err
			}
			b.Write(vb)
		} else {
			vb, _ := json.Marshal(v)
			b.Write(vb)
		}
	}
	b.WriteByte('}')
	return []byte(b.String()), nil
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(strings.TrimSpace(s), "%f", &f)
	return f
}
