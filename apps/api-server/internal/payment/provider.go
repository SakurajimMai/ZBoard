// Package payment defines the provider-agnostic payment interface. Each
// gateway (EasyPay, Creem, NOWPayments) implements Provider so the business
// layer doesn't couple to any single vendor.
package payment

import "context"

// CreateRequest is the input to Provider.CreatePayment.
type CreateRequest struct {
	OrderNo   string // internal order number
	Amount    string // decimal string, e.g. "9.90"
	Currency  string // CNY / USD / USDT / BTC etc.
	Subject   string // display name shown to user
	PayType   string // provider-specific: alipay / wxpay / card / crypto
	NotifyURL string // async webhook callback URL
	ReturnURL string // user redirect after payment
	CancelURL string // user redirect after cancelling payment
	ClientIP  string // payer IP (some gateways require)
	UserID    int64  // internal user id for reference
}

// CreateResponse is what Provider.CreatePayment returns.
type CreateResponse struct {
	ProviderOrderNo string // the gateway's own order/session ID
	PayURL          string // URL to redirect the user to
	QRCode          string // optional QR code content (for wechat native)
	RawResponse     string // full response body for audit
}

// CallbackData is the parsed + verified webhook payload.
type CallbackData struct {
	ProviderOrderNo string // gateway's order ID
	OrderNo         string // our order_no echoed back
	Amount          string // amount actually paid
	Status          string // normalized: "success" | "failed" | "pending"
	RawBody         string // raw request body for audit
}

// Provider is the interface every payment gateway must implement.
type Provider interface {
	// Name returns the provider identifier stored in the payments table.
	Name() string

	// CreatePayment initiates a payment and returns a pay URL or QR code.
	CreatePayment(ctx context.Context, req CreateRequest) (*CreateResponse, error)

	// VerifyCallback parses and signature-verifies an incoming webhook. It
	// returns the normalized callback data or an error if verification fails.
	VerifyCallback(ctx context.Context, headers map[string]string, body []byte) (*CallbackData, error)
}
