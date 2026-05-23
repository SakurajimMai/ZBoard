package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/store"
)

var publicSettingKeys = map[string]bool{
	"site_name":                true,
	"site_url":                 true,
	"subscription_name":        true,
	"subscription_domain":      true,
	"support_email":            true,
	"support_telegram":         true,
	"seo_title":                true,
	"seo_description":          true,
	"seo_keywords":             true,
	"allow_register":           true,
	"require_email_verify":     true,
	"captcha_provider":         true,
	"captcha_site_key":         true,
	"captcha_enabled_register": true,
	"captcha_enabled_login":    true,
	"captcha_enabled_forgot":   true,
	"captcha_enabled_ticket":   true,
	"turnstile_mode":           true,
}

type updateSettingsBody struct {
	Settings map[string]string `json:"settings" binding:"required"`
}

func publicSettings(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		all, err := d.Store.ListSettings(c.Request.Context())
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		out := defaultPublicSettings()
		for key, value := range all {
			if publicSettingKeys[key] {
				out[key] = value
			}
		}
		httpx.OK(c, gin.H{"settings": out})
	}
}

func adminGetSettings(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		settings := defaultAdminSettings()
		rows, err := d.Store.ListSettings(c.Request.Context())
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		for key, value := range rows {
			if _, ok := settings[key]; ok {
				settings[key] = value
			}
		}
		httpx.OK(c, gin.H{"settings": settings})
	}
}

func adminUpdateSettings(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body updateSettingsBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		if err := d.Store.SetSettings(c.Request.Context(), body.Settings); err != nil {
			httpx.Fail(c, err)
			return
		}
		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "settings.update", ResourceType: "settings",
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, gin.H{"ok": true})
	}
}

func defaultPublicSettings() map[string]string {
	return map[string]string{
		"site_name":            "Zboard",
		"site_url":             "",
		"subscription_name":    "Zboard",
		"subscription_domain":  "",
		"support_email":        "",
		"support_telegram":     "",
		"seo_title":            "Zboard",
		"seo_description":      "",
		"seo_keywords":         "",
		"allow_register":       "1",
		"require_email_verify": "0",
	}
}

func defaultAdminSettings() map[string]string {
	settings := defaultPublicSettings()
	for key, value := range map[string]string{
		"default_language":          "zh-CN",
		"trial_traffic_gb":          "0",
		"trial_days":                "0",
		"user_default_device_limit": "3",
		"require_email_verify":      "0",
		"clash_enabled":             "1",
		"singbox_enabled":           "1",
		"v2rayn_enabled":            "1",
		"smtp_host":                 "",
		"smtp_port":                 "587",
		"smtp_user":                 "",
		"smtp_pass":                 "",
		"smtp_from_name":            "",
		"smtp_from_email":           "",
		"smtp_encryption":           "starttls",
		"captcha_provider":          "none",
		"captcha_site_key":          "",
		"captcha_secret_key":        "",
		"captcha_enabled_register":  "0",
		"captcha_enabled_login":     "0",
		"captcha_enabled_forgot":    "0",
		"captcha_enabled_ticket":    "0",
		"turnstile_mode":            "managed",
		"admin_path":                "/admin",
		"email_domain_whitelist":    "",
	} {
		settings[key] = value
	}
	return settings
}
