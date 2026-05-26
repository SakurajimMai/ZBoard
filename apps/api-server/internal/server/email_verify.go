package server

import (
	"context"
	"strings"
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
