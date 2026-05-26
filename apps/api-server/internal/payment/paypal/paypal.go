// Package paypal implements PayPal Orders v2 checkout payments.
package paypal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zboard/api-server/internal/payment"
)

type Config struct {
	ClientID     string
	ClientSecret string
	WebhookID    string
	APIURL       string
}

type paypalProvider struct {
	cfg    Config
	client *http.Client
}

func New(cfg Config) payment.Provider {
	if strings.TrimSpace(cfg.APIURL) == "" {
		cfg.APIURL = "https://api-m.paypal.com"
	}
	cfg.APIURL = strings.TrimRight(cfg.APIURL, "/")
	return &paypalProvider{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *paypalProvider) Name() string { return "paypal" }

func (p *paypalProvider) CreatePayment(ctx context.Context, req payment.CreateRequest) (*payment.CreateResponse, error) {
	token, err := p.accessToken(ctx)
	if err != nil {
		return nil, err
	}
	body := map[string]any{
		"intent": "CAPTURE",
		"purchase_units": []map[string]any{{
			"reference_id": req.OrderNo,
			"description":  req.Subject,
			"custom_id":    req.OrderNo,
			"amount": map[string]string{
				"currency_code": strings.ToUpper(defaultString(req.Currency, "USD")),
				"value":         req.Amount,
			},
		}},
		"payment_source": map[string]any{
			"paypal": map[string]any{
				"experience_context": map[string]string{
					"return_url":          req.ReturnURL,
					"cancel_url":          defaultString(req.CancelURL, req.ReturnURL),
					"brand_name":          "Zboard",
					"shipping_preference": "NO_SHIPPING",
					"user_action":         "PAY_NOW",
				},
			},
		},
	}
	raw, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.APIURL+"/v2/checkout/orders", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("PayPal-Request-Id", req.OrderNo)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("paypal: create order request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("paypal: create order %d %s", resp.StatusCode, string(respBody))
	}

	var out struct {
		ID    string `json:"id"`
		Links []struct {
			Href string `json:"href"`
			Rel  string `json:"rel"`
		} `json:"links"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("paypal: decode order response: %w", err)
	}
	payURL := ""
	for _, link := range out.Links {
		if link.Rel == "approve" {
			payURL = link.Href
			break
		}
	}
	if payURL == "" {
		return nil, fmt.Errorf("paypal: approve link missing")
	}
	return &payment.CreateResponse{
		ProviderOrderNo: out.ID,
		PayURL:          payURL,
		RawResponse:     string(respBody),
	}, nil
}

func (p *paypalProvider) VerifyCallback(ctx context.Context, headers map[string]string, body []byte) (*payment.CallbackData, error) {
	if err := p.verifyWebhook(ctx, headers, body); err != nil {
		return nil, err
	}
	var event struct {
		ID           string `json:"id"`
		EventType    string `json:"event_type"`
		ResourceType string `json:"resource_type"`
		Resource     struct {
			ID            string `json:"id"`
			Status        string `json:"status"`
			CustomID      string `json:"custom_id"`
			InvoiceID     string `json:"invoice_id"`
			PurchaseUnits []struct {
				ReferenceID string `json:"reference_id"`
				CustomID    string `json:"custom_id"`
				Payments    struct {
					Captures []struct {
						ID     string `json:"id"`
						Status string `json:"status"`
						Amount struct {
							Value string `json:"value"`
						} `json:"amount"`
					} `json:"captures"`
				} `json:"payments"`
			} `json:"purchase_units"`
			Amount struct {
				Value string `json:"value"`
			} `json:"amount"`
		} `json:"resource"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("paypal: decode webhook: %w", err)
	}

	orderNo := firstNonEmpty(event.Resource.CustomID, event.Resource.InvoiceID)
	amount := event.Resource.Amount.Value
	providerOrderNo := firstNonEmpty(event.ID, event.Resource.ID)
	status := "pending"
	if event.EventType == "CHECKOUT.ORDER.APPROVED" {
		status = "pending"
	} else if event.EventType == "PAYMENT.CAPTURE.COMPLETED" || event.Resource.Status == "COMPLETED" {
		status = "success"
	} else if strings.Contains(event.EventType, "DENIED") || strings.Contains(event.EventType, "FAILED") || strings.Contains(event.EventType, "VOIDED") {
		status = "failed"
	}
	for _, unit := range event.Resource.PurchaseUnits {
		orderNo = firstNonEmpty(orderNo, unit.ReferenceID, unit.CustomID)
		for _, capture := range unit.Payments.Captures {
			providerOrderNo = firstNonEmpty(capture.ID, providerOrderNo)
			amount = firstNonEmpty(amount, capture.Amount.Value)
			if capture.Status == "COMPLETED" {
				status = "success"
			}
		}
	}
	return &payment.CallbackData{
		ProviderOrderNo: providerOrderNo,
		OrderNo:         orderNo,
		Amount:          amount,
		Status:          status,
		RawBody:         string(body),
	}, nil
}

func (p *paypalProvider) accessToken(ctx context.Context) (string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.APIURL+"/v1/oauth2/token", strings.NewReader("grant_type=client_credentials"))
	if err != nil {
		return "", err
	}
	httpReq.SetBasicAuth(p.cfg.ClientID, p.cfg.ClientSecret)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("paypal: token request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("paypal: token %d %s", resp.StatusCode, string(respBody))
	}
	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", fmt.Errorf("paypal: decode token response: %w", err)
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("paypal: empty access token")
	}
	return out.AccessToken, nil
}

func (p *paypalProvider) verifyWebhook(ctx context.Context, headers map[string]string, body []byte) error {
	if p.cfg.WebhookID == "" {
		return fmt.Errorf("paypal: webhook_id is required")
	}
	token, err := p.accessToken(ctx)
	if err != nil {
		return err
	}
	bodyMap := map[string]any{}
	if err := json.Unmarshal(body, &bodyMap); err != nil {
		return fmt.Errorf("paypal: decode webhook body: %w", err)
	}
	verifyBody := map[string]any{
		"auth_algo":         headerValue(headers, "Paypal-Auth-Algo"),
		"cert_url":          headerValue(headers, "Paypal-Cert-Url"),
		"transmission_id":   headerValue(headers, "Paypal-Transmission-Id"),
		"transmission_sig":  headerValue(headers, "Paypal-Transmission-Sig"),
		"transmission_time": headerValue(headers, "Paypal-Transmission-Time"),
		"webhook_id":        p.cfg.WebhookID,
		"webhook_event":     bodyMap,
	}
	raw, _ := json.Marshal(verifyBody)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.APIURL+"/v1/notifications/verify-webhook-signature", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("paypal: verify webhook request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("paypal: verify webhook %d %s", resp.StatusCode, string(respBody))
	}
	var out struct {
		VerificationStatus string `json:"verification_status"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("paypal: decode verify response: %w", err)
	}
	if out.VerificationStatus != "SUCCESS" {
		return fmt.Errorf("paypal: webhook signature status %s", out.VerificationStatus)
	}
	return nil
}

func (p *paypalProvider) CaptureOrder(ctx context.Context, orderID string) (*payment.CallbackData, error) {
	token, err := p.accessToken(ctx)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.APIURL+"/v2/checkout/orders/"+orderID+"/capture", nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("paypal: capture request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("paypal: capture %d %s", resp.StatusCode, string(respBody))
	}
	var out struct {
		ID            string `json:"id"`
		Status        string `json:"status"`
		PurchaseUnits []struct {
			ReferenceID string `json:"reference_id"`
			Payments    struct {
				Captures []struct {
					ID     string `json:"id"`
					Status string `json:"status"`
					Amount struct {
						Value string `json:"value"`
					} `json:"amount"`
				} `json:"captures"`
			} `json:"payments"`
		} `json:"purchase_units"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("paypal: decode capture response: %w", err)
	}
	orderNo := ""
	providerOrderNo := out.ID
	amount := ""
	status := "pending"
	if out.Status == "COMPLETED" {
		status = "success"
	}
	for _, unit := range out.PurchaseUnits {
		orderNo = firstNonEmpty(orderNo, unit.ReferenceID)
		for _, capture := range unit.Payments.Captures {
			providerOrderNo = firstNonEmpty(capture.ID, providerOrderNo)
			amount = firstNonEmpty(amount, capture.Amount.Value)
			if capture.Status == "COMPLETED" {
				status = "success"
			}
		}
	}
	return &payment.CallbackData{
		ProviderOrderNo: providerOrderNo,
		OrderNo:         orderNo,
		Amount:          amount,
		Status:          status,
		RawBody:         string(respBody),
	}, nil
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
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
