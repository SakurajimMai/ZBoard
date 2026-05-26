package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterAllowsCommaSeparatedEmailDomainWhitelist(t *testing.T) {
	r, st, _ := setupAdminCRUDRouter(t)
	if err := st.SetSettings(t.Context(), map[string]string{
		"allow_register":         "1",
		"email_domain_whitelist": "gmail.com,qq.com,163.com,yahoo.com,sina.com,126.com,outlook.com,yeah.net,foxmail.com",
	}); err != nil {
		t.Fatalf("set settings: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(`{"email":"allowed@qq.com","password":"secret123"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("register with comma whitelist code=%d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(`{"email":"blocked@example.org","password":"secret123"}`))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden || !bytes.Contains(rr.Body.Bytes(), []byte("email_domain_not_allowed")) {
		t.Fatalf("register outside whitelist should be rejected, code=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSendRegisterCodeRejectsDomainOutsideCommaSeparatedWhitelist(t *testing.T) {
	r, st, _ := setupAdminCRUDRouter(t)
	if err := st.SetSettings(t.Context(), map[string]string{
		"allow_register":         "1",
		"smtp_host":              "smtp.example.com",
		"smtp_from_email":        "noreply@example.com",
		"email_domain_whitelist": "gmail.com,qq.com,163.com",
	}); err != nil {
		t.Fatalf("set settings: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/send-email-code", bytes.NewBufferString(`{"email":"blocked@example.org","purpose":"register"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden || !bytes.Contains(rr.Body.Bytes(), []byte("email_domain_not_allowed")) {
		t.Fatalf("send register code outside whitelist should be rejected, code=%d body=%s", rr.Code, rr.Body.String())
	}
}
