package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/zboard/api-server/internal/httpx"
)

func emailVerifyAvailable(ctx context.Context, d Deps) bool {
	if d.Store == nil {
		return false
	}
	host, err := d.Store.GetSetting(ctx, "smtp_host", "")
	if err != nil {
		return false
	}
	from, err := d.Store.GetSetting(ctx, "smtp_from_email", "")
	if err != nil {
		return false
	}
	if strings.TrimSpace(host) != "" && strings.TrimSpace(from) != "" {
		return true
	}
	return d.Auth != nil && d.Auth.Mailer != nil && d.Auth.Mailer.Enabled()
}

func requireEmailVerifyEffective(ctx context.Context, d Deps) (bool, error) {
	required, err := d.Store.BoolSetting(ctx, "require_email_verify", false)
	if err != nil || !required {
		return false, err
	}
	if !emailVerifyAvailable(ctx, d) {
		return false, nil
	}
	return true, nil
}

func parseEmailDomainWhitelist(raw string) map[string]bool {
	items := strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case ',', '，', ';', '；', '\n', '\r', '\t', ' ':
			return true
		default:
			return false
		}
	})
	out := make(map[string]bool, len(items))
	for _, item := range items {
		domain := strings.Trim(strings.TrimSpace(strings.ToLower(item)), ".")
		if domain != "" {
			out[domain] = true
		}
	}
	return out
}

func requireEmailDomainAllowed(ctx context.Context, d Deps, email string) error {
	raw, err := d.Store.GetSetting(ctx, "email_domain_whitelist", "")
	if err != nil {
		return err
	}
	allowed := parseEmailDomainWhitelist(raw)
	if len(allowed) == 0 {
		return nil
	}
	parts := strings.Split(strings.TrimSpace(strings.ToLower(email)), "@")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
		return httpx.NewError(http.StatusBadRequest, "bad_request", "邮箱格式不正确")
	}
	domain := strings.Trim(parts[1], ".")
	if domain == "" || !allowed[domain] {
		return httpx.NewError(http.StatusForbidden, "email_domain_not_allowed", "邮箱域名不在允许注册范围内")
	}
	return nil
}
