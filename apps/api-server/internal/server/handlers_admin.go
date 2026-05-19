package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/store"
)

type adminBootstrapBody struct {
	SetupToken string `json:"setup_token" binding:"required"`
	Email      string `json:"email" binding:"required"`
	Password   string `json:"password" binding:"required"`
}

func adminBootstrap(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body adminBootstrapBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		id, err := d.Auth.BootstrapAdmin(c.Request.Context(), body.SetupToken, body.Email, body.Password)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(id),
			Action: "admin.bootstrap", IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.Created(c, gin.H{"admin_id": id})
	}
}

func adminLogin(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body credentialsBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		token, a, err := d.Auth.LoginAdmin(c.Request.Context(), body.Email, body.Password)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "admin.login", IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, gin.H{"token": token, "admin": adminView(a)})
	}
}

func adminMe(_ Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		httpx.OK(c, gin.H{"admin": adminView(a)})
	}
}

func adminLogout(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		tok, _ := c.Get(ctxAdminTok)
		_ = d.Auth.LogoutAdmin(c.Request.Context(), tok.(string))
		httpx.OK(c, gin.H{"ok": true})
	}
}

func adminAuditLogs(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := d.Store.ListAuditLogs(c.Request.Context(), 100)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"items": rows})
	}
}

func adminView(a *store.AdminUser) gin.H {
	return gin.H{
		"id":     a.ID,
		"email":  a.Email,
		"role":   a.Role,
		"status": a.Status,
	}
}
