package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zboard/api-server/internal/authsvc"
)

func TestLoginWorksWhenCaptchaDependencyOmitted(t *testing.T) {
	r, st, _ := setupAdminCRUDRouter(t)
	auth := authsvc.New(st, "setup-token", nil)
	if _, err := auth.RegisterUser(context.Background(), "captcha-login@example.com", "secret123"); err != nil {
		t.Fatalf("register seed user: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"email":"captcha-login@example.com","password":"secret123"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("login should not require explicit Captcha dependency when captcha disabled, code=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestRegisterEndpointRequiresCaptchaWhenEnabled(t *testing.T) {
	r, st, _ := setupAdminCRUDRouter(t)
	if err := st.SetSettings(context.Background(), map[string]string{
		"captcha_provider":         "turnstile",
		"captcha_secret_key":       "test-secret",
		"captcha_enabled_register": "1",
	}); err != nil {
		t.Fatalf("set captcha settings: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(`{"email":"captcha-register@example.com","password":"secret123"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest || !bytes.Contains(rr.Body.Bytes(), []byte("captcha_required")) {
		t.Fatalf("register without captcha should be rejected, code=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCreateTicketRequiresCaptchaWhenEnabled(t *testing.T) {
	r, st, _ := setupAdminCRUDRouter(t)
	ctx := context.Background()
	auth := authsvc.New(st, "setup-token", nil)
	if _, err := auth.RegisterUser(ctx, "captcha-ticket@example.com", "secret123"); err != nil {
		t.Fatalf("register seed user: %v", err)
	}
	token, _, err := auth.LoginUser(ctx, "captcha-ticket@example.com", "secret123")
	if err != nil {
		t.Fatalf("login seed user: %v", err)
	}
	if err := st.SetSettings(ctx, map[string]string{
		"captcha_provider":       "turnstile",
		"captcha_secret_key":     "test-secret",
		"captcha_enabled_ticket": "1",
	}); err != nil {
		t.Fatalf("set captcha settings: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tickets", bytes.NewBufferString(`{"subject":"需要帮助","category":"general","content":"无法连接节点"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest || !bytes.Contains(rr.Body.Bytes(), []byte("captcha_required")) {
		t.Fatalf("ticket without captcha should be rejected, code=%d body=%s", rr.Code, rr.Body.String())
	}
}
