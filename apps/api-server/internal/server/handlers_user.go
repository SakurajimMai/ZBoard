package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/store"
)

type credentialsBody struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func registerUser(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body credentialsBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		id, err := d.Auth.RegisterUser(c.Request.Context(), body.Email, body.Password)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "user", ActorID: ptrInt64(id),
			Action: "user.register", IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.Created(c, gin.H{"user_id": id})
	}
}

func loginUser(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body credentialsBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		token, u, err := d.Auth.LoginUser(c.Request.Context(), body.Email, body.Password)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "user", ActorID: ptrInt64(u.ID),
			Action: "user.login", IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, gin.H{"token": token, "user": userView(u)})
	}
}

func logoutUser(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, _ := c.Get(ctxUserTok)
		_ = d.Auth.LogoutUser(c.Request.Context(), token.(string))
		httpx.OK(c, gin.H{"ok": true})
	}
}

func currentUser(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.MustGet(ctxUserIDKey).(int64)
		u, err := d.Store.FindUserByID(c.Request.Context(), uid)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"user": userView(u)})
	}
}

func userView(u *store.User) gin.H {
	return gin.H{
		"id":            u.ID,
		"email":         u.Email,
		"plan_id":       u.PlanID,
		"expired_at":    u.ExpiredAt,
		"traffic_limit": u.TrafficLimit,
		"traffic_used":  u.TrafficUsed,
		"status":        u.Status,
	}
}

func ptrInt64(v int64) *int64 { return &v }
