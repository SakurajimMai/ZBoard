package server

import (
	"encoding/json"
	"testing"
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
