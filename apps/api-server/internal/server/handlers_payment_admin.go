package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/store"
)

// ===== Admin Payment Provider CRUD =====

func adminListPaymentProviders(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := d.Store.ListPaymentProviders(c.Request.Context())
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		// Mask sensitive config fields in the list view.
		view := make([]gin.H, 0, len(rows))
		for _, r := range rows {
			view = append(view, gin.H{
				"id":            r.ID,
				"name":          r.Name,
				"display_name":  r.DisplayName,
				"provider_type": r.ProviderType,
				"config_json":   maskConfig(r.ConfigJSON),
				"enabled":       r.Enabled,
				"sort":          r.Sort,
				"created_at":    r.CreatedAt,
				"updated_at":    r.UpdatedAt,
			})
		}
		httpx.OK(c, gin.H{"items": view})
	}
}

type createPaymentProviderBody struct {
	Name         string `json:"name" binding:"required"`
	DisplayName  string `json:"display_name"`
	ProviderType string `json:"provider_type" binding:"required"`
	ConfigJSON   string `json:"config_json" binding:"required"`
	Enabled      *int   `json:"enabled"`
	Sort         int    `json:"sort"`
}

func adminCreatePaymentProvider(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body createPaymentProviderBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		// Validate config_json is valid JSON.
		if !json.Valid([]byte(body.ConfigJSON)) {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "config_json 不是合法 JSON"))
			return
		}
		// Validate provider_type is known.
		switch body.ProviderType {
		case "epay", "creem", "nowpayments":
		default:
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request",
				"provider_type 必须是 epay / creem / nowpayments"))
			return
		}
		enabled := 1
		if body.Enabled != nil {
			enabled = *body.Enabled
		}
		id, err := d.Store.CreatePaymentProvider(c.Request.Context(), store.CreatePaymentProviderInput{
			Name:         body.Name,
			DisplayName:  body.DisplayName,
			ProviderType: body.ProviderType,
			ConfigJSON:   body.ConfigJSON,
			Enabled:      enabled,
			Sort:         body.Sort,
		})
		if err != nil {
			if store.IsUniqueViolation(err) {
				httpx.Fail(c, httpx.NewError(http.StatusConflict, "name_taken", "渠道名称已存在"))
				return
			}
			httpx.Fail(c, err)
			return
		}
		// Reload the registry so the new provider takes effect immediately.
		_ = d.Payments.Reload(c.Request.Context())

		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "payment_provider.create", ResourceType: "payment_provider",
			ResourceID: body.Name, IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.Created(c, gin.H{"id": id})
	}
}

type updatePaymentProviderBody struct {
	DisplayName string `json:"display_name"`
	ConfigJSON  string `json:"config_json"`
	Enabled     *int   `json:"enabled"`
	Sort        int    `json:"sort"`
}

func adminUpdatePaymentProvider(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "id 不合法"))
			return
		}
		var body updatePaymentProviderBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		if body.ConfigJSON != "" && !json.Valid([]byte(body.ConfigJSON)) {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "config_json 不是合法 JSON"))
			return
		}
		enabled := 1
		if body.Enabled != nil {
			enabled = *body.Enabled
		}
		if err := d.Store.UpdatePaymentProvider(c.Request.Context(), id, body.DisplayName, body.ConfigJSON, enabled, body.Sort); err != nil {
			httpx.Fail(c, err)
			return
		}
		// Reload registry.
		_ = d.Payments.Reload(c.Request.Context())

		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "payment_provider.update", ResourceType: "payment_provider",
			ResourceID: strconv.FormatInt(id, 10), IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, gin.H{"ok": true})
	}
}

func adminDeletePaymentProvider(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "id 不合法"))
			return
		}
		if err := d.Store.DeletePaymentProvider(c.Request.Context(), id); err != nil {
			httpx.Fail(c, err)
			return
		}
		_ = d.Payments.Reload(c.Request.Context())

		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "payment_provider.delete", ResourceType: "payment_provider",
			ResourceID: strconv.FormatInt(id, 10), IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, gin.H{"ok": true})
	}
}

// maskConfig hides sensitive values in config_json for list display.
// Shows keys but replaces values longer than 4 chars with "****".
func maskConfig(raw string) string {
	var obj map[string]any
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return "{}"
	}
	for k, v := range obj {
		if s, ok := v.(string); ok && len(s) > 4 {
			obj[k] = s[:2] + "****" + s[len(s)-2:]
		}
	}
	b, _ := json.Marshal(obj)
	return string(b)
}

// ===== User-facing: list available payment methods =====

func listAvailablePaymentMethods(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := d.Store.ListEnabledPaymentProviders(c.Request.Context())
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		methods := make([]gin.H, 0, len(rows))
		for _, r := range rows {
			methods = append(methods, gin.H{
				"name":          r.Name,
				"display_name":  r.DisplayName,
				"provider_type": r.ProviderType,
			})
		}
		httpx.OK(c, gin.H{"methods": methods})
	}
}
