// Package epay implements the 易支付 (EasyPay) payment provider.
//
// EasyPay API flow:
//   1. Server builds a signed request (MD5 sign) and redirects user to the
//      gateway's cashier page.
//   2. User pays via Alipay / WeChat.
//   3. Gateway POSTs an async notification to notify_url with the same MD5
//      signature scheme.
//   4. Server verifies signature, marks order paid, returns "success" text.
//
// Signature: sort params alphabetically (exclude sign, sign_type, empty values),
// join with "&", append "&key", MD5 hex.
package epay

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/zboard/api-server/internal/payment"
)

// Config holds EasyPay gateway credentials.
type Config struct {
	APIURL    string // e.g. https://pay.example.com
	PID       string // merchant ID (partner ID)
	SecretKey string // MD5 signing key
}

type epayProvider struct {
	cfg Config
}

// New creates an EasyPay provider.
func New(cfg Config) payment.Provider {
	return &epayProvider{cfg: cfg}
}

func (p *epayProvider) Name() string { return "epay" }

func (p *epayProvider) CreatePayment(_ context.Context, req payment.CreateRequest) (*payment.CreateResponse, error) {
	params := map[string]string{
		"pid":          p.cfg.PID,
		"type":         mapPayType(req.PayType),
		"out_trade_no": req.OrderNo,
		"notify_url":   req.NotifyURL,
		"return_url":   req.ReturnURL,
		"name":         req.Subject,
		"money":        req.Amount,
		"clientip":     req.ClientIP,
	}
	params["sign"] = sign(params, p.cfg.SecretKey)
	params["sign_type"] = "MD5"

	// Build the redirect URL (GET-based cashier).
	u, err := url.Parse(p.cfg.APIURL + "/submit.php")
	if err != nil {
		return nil, fmt.Errorf("epay: parse api url: %w", err)
	}
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	return &payment.CreateResponse{
		ProviderOrderNo: "", // EasyPay assigns its own trade_no in the callback
		PayURL:          u.String(),
		RawResponse:     u.String(),
	}, nil
}

func (p *epayProvider) VerifyCallback(_ context.Context, _ map[string]string, body []byte) (*payment.CallbackData, error) {
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, fmt.Errorf("epay: parse callback body: %w", err)
	}
	receivedSign := values.Get("sign")
	if receivedSign == "" {
		return nil, fmt.Errorf("epay: missing sign in callback")
	}

	// Rebuild params map excluding sign, sign_type, empty values.
	params := map[string]string{}
	for k := range values {
		if k == "sign" || k == "sign_type" {
			continue
		}
		v := values.Get(k)
		if v != "" {
			params[k] = v
		}
	}
	expected := sign(params, p.cfg.SecretKey)
	if !strings.EqualFold(expected, receivedSign) {
		return nil, fmt.Errorf("epay: signature mismatch (got %s, want %s)", receivedSign, expected)
	}

	status := "failed"
	if values.Get("trade_status") == "TRADE_SUCCESS" {
		status = "success"
	}

	return &payment.CallbackData{
		ProviderOrderNo: values.Get("trade_no"),
		OrderNo:         values.Get("out_trade_no"),
		Amount:          values.Get("money"),
		Status:          status,
		RawBody:         string(body),
	}, nil
}

// sign builds the EasyPay MD5 signature.
// Steps: sort keys, join "k=v" with "&", append the secret key, MD5 hex.
func sign(params map[string]string, key string) string {
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if v == "" || k == "sign" || k == "sign_type" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, k+"="+params[k])
	}
	raw := strings.Join(pairs, "&") + key
	sum := md5.Sum([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func mapPayType(t string) string {
	switch strings.ToLower(t) {
	case "alipay", "ali":
		return "alipay"
	case "wxpay", "wechat", "wx":
		return "wxpay"
	case "qqpay", "qq":
		return "qqpay"
	default:
		return "alipay"
	}
}
