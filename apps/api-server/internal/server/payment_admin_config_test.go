package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/zboard/api-server/internal/store"
)

func TestPreserveMaskedConfigOnlyKeepsSensitiveValues(t *testing.T) {
	existing := `{"api_url":"https://sandbox.example.com","client_id":"client-1","client_secret":"secret-1","webhook_id":"WH-1"}`
	incoming := `{"api_url":"","client_id":"client-1","client_secret":"se****-1","webhook_id":""}`

	got := preserveMaskedConfig(existing, incoming)
	var cfg map[string]string
	if err := json.Unmarshal([]byte(got), &cfg); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if cfg["api_url"] != "" {
		t.Fatalf("api_url=%q, want cleared non-sensitive value", cfg["api_url"])
	}
	if cfg["client_secret"] != "secret-1" {
		t.Fatalf("client_secret=%q, want preserved sensitive value", cfg["client_secret"])
	}
	if cfg["webhook_id"] != "" {
		t.Fatalf("webhook_id=%q, want cleared non-sensitive value", cfg["webhook_id"])
	}
}

func TestAdminListPaymentProvidersReturnsFullSensitiveConfig(t *testing.T) {
	r, st, token := setupAdminCRUDRouter(t)
	ctx := context.Background()
	if _, err := st.CreatePaymentProvider(ctx, store.CreatePaymentProviderInput{
		Name:         "stripe_live",
		DisplayName:  "Stripe Live",
		ProviderType: "stripe",
		ConfigJSON:   `{"secret_key":"sk_live_secret","webhook_secret":"whsec_secret","api_url":"https://api.stripe.com"}`,
		Enabled:      1,
		Sort:         1,
	}); err != nil {
		t.Fatalf("create payment provider: %v", err)
	}

	resp := adminJSON(t, r, token, http.MethodGet, "/api/admin/v1/payment-providers", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("list providers status=%d body=%s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	if !strings.Contains(body, "sk_live_secret") || strings.Contains(body, "****") {
		t.Fatalf("admin list should return full sensitive config, body=%s", body)
	}
}
