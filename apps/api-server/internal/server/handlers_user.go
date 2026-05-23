package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/captchasvc"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/store"
)

type credentialsBody struct {
	Email        string `json:"email" binding:"required"`
	Password     string `json:"password" binding:"required"`
	CaptchaToken string `json:"captcha_token"`
}

func registerUser(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		allowRegister, err := d.Store.BoolSetting(c.Request.Context(), "allow_register", true)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		if !allowRegister {
			httpx.Fail(c, httpx.NewError(http.StatusForbidden, "register_disabled", "当前站点已关闭用户注册"))
			return
		}
		requireEmailVerify, err := d.Store.BoolSetting(c.Request.Context(), "require_email_verify", false)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		if requireEmailVerify {
			httpx.Fail(c, httpx.NewError(http.StatusForbidden, "email_verify_required", "当前站点要求邮箱验证码注册"))
			return
		}
		var body credentialsBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		if err := d.Captcha.Verify(c.Request.Context(), captchasvc.SceneRegister, body.CaptchaToken, c.ClientIP()); err != nil {
			httpx.Fail(c, err)
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
		if err := d.Captcha.Verify(c.Request.Context(), captchasvc.SceneLogin, body.CaptchaToken, c.ClientIP()); err != nil {
			httpx.Fail(c, err)
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
