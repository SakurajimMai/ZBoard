package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zboard/api-server/internal/authsvc"
	"github.com/zboard/api-server/internal/bizsvc"
	"github.com/zboard/api-server/internal/nodesvc"
	"github.com/zboard/api-server/internal/payment/registry"
	"github.com/zboard/api-server/internal/store"
	"github.com/zboard/api-server/internal/testsupport"
	"github.com/zboard/api-server/internal/worker"
)

func setupSecurityAuthRouter(t *testing.T) (*store.Store, *authsvc.Service, http.Handler) {
	t.Helper()
	st := testsupport.NewStore(t)
	auth := authsvc.New(st, "setup-token", nil)
	r := New(Deps{
		DB:          st.DB,
		Store:       st,
		Auth:        auth,
		Biz:         bizsvc.New(st),
		Nodes:       nodesvc.New(st),
		Worker:      worker.New(st),
		Payments:    registry.New(st),
		TokenSecret: "test-token-secret-not-default",
	})
	return st, auth, r
}

func jsonReq(t *testing.T, r http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func TestDisabledUserSessionCannotUseAuthedRoutes(t *testing.T) {
	st, auth, r := setupSecurityAuthRouter(t)
	ctx := context.Background()
	userID, err := auth.RegisterUser(ctx, "disabled-session@example.com", "secret123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	token, _, err := auth.LoginUser(ctx, "disabled-session@example.com", "secret123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if err := st.SetUserStatus(ctx, userID, "disabled"); err != nil {
		t.Fatalf("disable user: %v", err)
	}

	resp := jsonReq(t, r, http.MethodGet, "/api/v1/me", token, nil)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("disabled user session should be rejected, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestPasswordChangeRevokesExistingUserSessions(t *testing.T) {
	_, auth, r := setupSecurityAuthRouter(t)
	ctx := context.Background()
	if _, err := auth.RegisterUser(ctx, "change-revoke@example.com", "secret123"); err != nil {
		t.Fatalf("register: %v", err)
	}
	oldToken, _, err := auth.LoginUser(ctx, "change-revoke@example.com", "secret123")
	if err != nil {
		t.Fatalf("login old: %v", err)
	}
	currentToken, _, err := auth.LoginUser(ctx, "change-revoke@example.com", "secret123")
	if err != nil {
		t.Fatalf("login current: %v", err)
	}

	change := jsonReq(t, r, http.MethodPost, "/api/v1/me/password", currentToken, map[string]any{
		"current_password": "secret123",
		"new_password":     "changed123",
	})
	if change.Code != http.StatusOK {
		t.Fatalf("change password status=%d body=%s", change.Code, change.Body.String())
	}

	resp := jsonReq(t, r, http.MethodGet, "/api/v1/me", oldToken, nil)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("old session should be revoked, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestPasswordResetRevokesExistingUserSessions(t *testing.T) {
	st, auth, r := setupSecurityAuthRouter(t)
	ctx := context.Background()
	if _, err := auth.RegisterUser(ctx, "reset-revoke@example.com", "secret123"); err != nil {
		t.Fatalf("register: %v", err)
	}
	oldToken, _, err := auth.LoginUser(ctx, "reset-revoke@example.com", "secret123")
	if err != nil {
		t.Fatalf("login old: %v", err)
	}
	if err := st.CreateEmailCode(ctx, "reset-revoke@example.com", "123456", "reset_password", authsvc.EmailCodeTTL); err != nil {
		t.Fatalf("create reset code: %v", err)
	}

	reset := jsonReq(t, r, http.MethodPost, "/api/v1/auth/reset-password", "", map[string]any{
		"email":        "reset-revoke@example.com",
		"new_password": "changed123",
		"code":         "123456",
	})
	if reset.Code != http.StatusOK {
		t.Fatalf("reset password status=%d body=%s", reset.Code, reset.Body.String())
	}

	resp := jsonReq(t, r, http.MethodGet, "/api/v1/me", oldToken, nil)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("old session should be revoked after reset, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestForwardedForDoesNotBypassLoginRateLimit(t *testing.T) {
	_, _, r := setupSecurityAuthRouter(t)
	var last *httptest.ResponseRecorder
	for i := 0; i < loginRateLimitBurst+1; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"xff@example.com","password":"wrong"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "203.0.113."+formatInt(int64(i+1)))
		last = httptest.NewRecorder()
		r.ServeHTTP(last, req)
	}
	if last.Code != http.StatusTooManyRequests {
		t.Fatalf("spoofed X-Forwarded-For should not bypass limiter, got %d body=%s", last.Code, last.Body.String())
	}
}

// With a trusted proxy configured, the limiter must bucket by the real client
// IP carried in X-Forwarded-For so a flood from one client behind the proxy
// can't 429 every other client sharing it.
func TestTrustedProxyRateLimitsByRealClientIP(t *testing.T) {
	st := testsupport.NewStore(t)
	auth := authsvc.New(st, "setup-token", nil)
	r := New(Deps{
		DB:    st.DB,
		Store: st,
		Auth:  auth,
		Biz:   bizsvc.New(st),
		Nodes: nodesvc.New(st),
		// httptest connections originate from 192.0.2.1; trust it as the proxy
		// tier so the real client IP comes from X-Forwarded-For.
		TrustedProxies: []string{"192.0.2.1/32"},
		Payments:       registry.New(st),
		TokenSecret:    "test-token-secret-not-default",
	})

	// Exhaust one client's budget.
	var flooded *httptest.ResponseRecorder
	for i := 0; i < loginRateLimitBurst+1; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"flood@example.com","password":"wrong"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "203.0.113.7")
		flooded = httptest.NewRecorder()
		r.ServeHTTP(flooded, req)
	}
	if flooded.Code != http.StatusTooManyRequests {
		t.Fatalf("flooding client should be limited, got %d body=%s", flooded.Code, flooded.Body.String())
	}

	// A different real client behind the same proxy must still be served.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"other@example.com","password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "203.0.113.8")
	other := httptest.NewRecorder()
	r.ServeHTTP(other, req)
	if other.Code == http.StatusTooManyRequests {
		t.Fatalf("a different client behind the proxy must not inherit another client's 429")
	}
}

// With TrustedPlatform set (e.g. Cloudflare's CF-Connecting-IP), the limiter
// must bucket by that header so a flood from one client doesn't 429 everyone,
// without having to maintain the CDN's egress CIDR list.
func TestTrustedPlatformRateLimitsByHeaderClientIP(t *testing.T) {
	st := testsupport.NewStore(t)
	auth := authsvc.New(st, "setup-token", nil)
	r := New(Deps{
		DB:              st.DB,
		Store:           st,
		Auth:            auth,
		Biz:             bizsvc.New(st),
		Nodes:           nodesvc.New(st),
		TrustedPlatform: "CF-Connecting-IP",
		Payments:        registry.New(st),
		TokenSecret:     "test-token-secret-not-default",
	})

	// Exhaust one client's budget keyed by the CF header.
	var flooded *httptest.ResponseRecorder
	for i := 0; i < loginRateLimitBurst+1; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"flood@example.com","password":"wrong"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("CF-Connecting-IP", "203.0.113.7")
		flooded = httptest.NewRecorder()
		r.ServeHTTP(flooded, req)
	}
	if flooded.Code != http.StatusTooManyRequests {
		t.Fatalf("flooding client should be limited, got %d body=%s", flooded.Code, flooded.Body.String())
	}

	// A different real client (different CF-Connecting-IP) must still be served.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"other@example.com","password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("CF-Connecting-IP", "203.0.113.8")
	other := httptest.NewRecorder()
	r.ServeHTTP(other, req)
	if other.Code == http.StatusTooManyRequests {
		t.Fatalf("a different client must not inherit another client's 429 under TrustedPlatform")
	}
}

func TestCodeVerificationRoutesAreRateLimited(t *testing.T) {
	st, _, r := setupSecurityAuthRouter(t)
	if err := st.SetSettings(context.Background(), map[string]string{
		"smtp_host":       "smtp.example.com",
		"smtp_from_email": "noreply@example.com",
	}); err != nil {
		t.Fatalf("settings: %v", err)
	}

	var last *httptest.ResponseRecorder
	for i := 0; i < codeRateLimitBurst+1; i++ {
		last = jsonReq(t, r, http.MethodPost, "/api/v1/auth/register-with-code", "", map[string]any{
			"email":    "code-limit@example.com",
			"password": "secret123",
			"code":     "000000",
		})
	}
	if last.Code != http.StatusTooManyRequests {
		t.Fatalf("register-with-code should be rate limited, got %d body=%s", last.Code, last.Body.String())
	}
}

func TestAdminLoginIsRateLimited(t *testing.T) {
	_, _, r := setupSecurityAuthRouter(t)
	var last *httptest.ResponseRecorder
	for i := 0; i < loginRateLimitBurst+1; i++ {
		last = jsonReq(t, r, http.MethodPost, "/api/admin/v1/auth/login", "", map[string]any{
			"email":    "admin@example.com",
			"password": "wrong",
		})
	}
	if last.Code != http.StatusTooManyRequests {
		t.Fatalf("admin login should be rate limited, got %d body=%s", last.Code, last.Body.String())
	}
}

func TestResetPasswordDoesNotRevealWhetherEmailExists(t *testing.T) {
	_, auth, r := setupSecurityAuthRouter(t)
	if _, err := auth.RegisterUser(context.Background(), "reset-exists@example.com", "secret123"); err != nil {
		t.Fatalf("register: %v", err)
	}

	existing := jsonReq(t, r, http.MethodPost, "/api/v1/auth/reset-password", "", map[string]any{
		"email":        "reset-exists@example.com",
		"new_password": "changed123",
		"code":         "000000",
	})
	missing := jsonReq(t, r, http.MethodPost, "/api/v1/auth/reset-password", "", map[string]any{
		"email":        "reset-missing@example.com",
		"new_password": "changed123",
		"code":         "000000",
	})
	if existing.Code != missing.Code || existing.Body.String() != missing.Body.String() {
		t.Fatalf("reset responses should not be an email oracle; existing=%d %s missing=%d %s",
			existing.Code, existing.Body.String(), missing.Code, missing.Body.String())
	}
}

func TestEmailCodeLocksAfterRepeatedInvalidAttempts(t *testing.T) {
	st, auth, _ := setupSecurityAuthRouter(t)
	ctx := context.Background()
	if err := st.CreateEmailCode(ctx, "lock-code@example.com", "123456", "reset_password", authsvc.EmailCodeTTL); err != nil {
		t.Fatalf("create code: %v", err)
	}
	for i := 0; i < store.EmailCodeMaxAttempts; i++ {
		if ok, err := st.VerifyEmailCode(ctx, "lock-code@example.com", "000000", "reset_password"); err != nil {
			t.Fatalf("verify wrong code: %v", err)
		} else if ok {
			t.Fatalf("wrong code attempt %d unexpectedly succeeded", i+1)
		}
	}
	if err := auth.ResetPasswordWithCode(ctx, "lock-code@example.com", "changed123", "123456"); err == nil {
		t.Fatalf("correct code should be locked after repeated invalid attempts")
	}
}

func TestSubscriptionTokenIsNotStoredPlaintext(t *testing.T) {
	st, auth, r := setupSecurityAuthRouter(t)
	ctx := context.Background()
	if _, err := auth.RegisterUser(ctx, "sub-secret@example.com", "secret123"); err != nil {
		t.Fatalf("register: %v", err)
	}
	token, _, err := auth.LoginUser(ctx, "sub-secret@example.com", "secret123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	resp := jsonReq(t, r, http.MethodGet, "/api/v1/subscription", token, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("subscription status=%d body=%s", resp.Code, resp.Body.String())
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode subscription response: %v", err)
	}
	row, err := st.FindActiveSubTokenByHash(ctx, hashSubToken(body.Token))
	if err != nil {
		t.Fatalf("find token by hash: %v", err)
	}
	if row.Token == body.Token {
		t.Fatalf("subscription token stored plaintext: %q", row.Token)
	}
}

func TestSubscriptionTokenRejectsMissingOrDefaultSigningSecret(t *testing.T) {
	for _, secret := range []string{"", "   ", "dev-token-secret"} {
		_, _, err := newSubTokenForUser(1, Deps{TokenSecret: secret})
		if err == nil {
			t.Fatalf("expected signing secret %q to be rejected", secret)
		}
	}
}

func TestStoredSubscriptionSaltRequiresExplicitSigningSecret(t *testing.T) {
	_, err := materializeSubToken(1, storedSubTokenPrefix+"salt", Deps{})
	if err == nil {
		t.Fatalf("expected stored subscription salt to require an explicit signing secret")
	}
}
