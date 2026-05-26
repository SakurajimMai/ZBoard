package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPublicSettingsDisableEmailVerifyWhenSMTPUnavailable(t *testing.T) {
	r, st, _ := setupAdminCRUDRouter(t)
	if err := st.SetSettings(t.Context(), map[string]string{
		"require_email_verify": "1",
		"default_language":     "en",
	}); err != nil {
		t.Fatalf("set settings: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("settings code=%d body=%s", rr.Code, rr.Body.String())
	}
	var got struct {
		Settings map[string]string `json:"settings"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	if got.Settings["default_language"] != "en" {
		t.Fatalf("default_language=%q, want en", got.Settings["default_language"])
	}
	if got.Settings["email_verify_available"] != "0" {
		t.Fatalf("email_verify_available=%q, want 0", got.Settings["email_verify_available"])
	}
	if got.Settings["require_email_verify"] != "0" {
		t.Fatalf("require_email_verify should be forced off without SMTP, got %q", got.Settings["require_email_verify"])
	}
}

func TestRegisterDoesNotRequireEmailCodeWhenSMTPUnavailable(t *testing.T) {
	r, st, _ := setupAdminCRUDRouter(t)
	if err := st.SetSettings(t.Context(), map[string]string{
		"allow_register":       "1",
		"require_email_verify": "1",
	}); err != nil {
		t.Fatalf("set settings: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(`{"email":"smtp-off@example.com","password":"secret123"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("register should bypass email code when SMTP unavailable, code=%d body=%s", rr.Code, rr.Body.String())
	}
}
