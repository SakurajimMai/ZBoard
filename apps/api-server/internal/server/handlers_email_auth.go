package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/captchasvc"
	"github.com/zboard/api-server/internal/httpx"
)

type sendEmailCodeBody struct {
	Email        string `json:"email" binding:"required"`
	Purpose      string `json:"purpose" binding:"required"` // "register" | "reset_password"
	CaptchaToken string `json:"captcha_token"`
}

func sendEmailCode(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body sendEmailCodeBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		if body.Purpose == "register" {
			allowRegister, err := d.Store.BoolSetting(c.Request.Context(), "allow_register", true)
			if err != nil {
				httpx.Fail(c, err)
				return
			}
			if !allowRegister {
				httpx.Fail(c, httpx.NewError(http.StatusForbidden, "register_disabled", "当前站点已关闭用户注册"))
				return
			}
		}
		if !emailVerifyAvailable(c.Request.Context(), d) {
			httpx.Fail(c, httpx.NewError(http.StatusForbidden, "email_verify_unavailable", "邮件服务未配置"))
			return
		}
		scene := captchasvc.SceneRegister
		if body.Purpose == "reset_password" {
			scene = captchasvc.SceneForgot
		}
		if err := d.Captcha.Verify(c.Request.Context(), scene, body.CaptchaToken, c.ClientIP()); err != nil {
			httpx.Fail(c, err)
			return
		}
		if err := d.Auth.SendEmailCode(c.Request.Context(), body.Email, body.Purpose); err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"ok": true})
	}
}

type registerWithCodeBody struct {
	Email        string `json:"email" binding:"required"`
	Password     string `json:"password" binding:"required"`
	Code         string `json:"code"`
	CaptchaToken string `json:"captcha_token"`
}

func registerUserWithCode(d Deps) gin.HandlerFunc {
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
		var body registerWithCodeBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		if err := d.Captcha.Verify(c.Request.Context(), captchasvc.SceneRegister, body.CaptchaToken, c.ClientIP()); err != nil {
			httpx.Fail(c, err)
			return
		}
		if !emailVerifyAvailable(c.Request.Context(), d) {
			id, err := d.Auth.RegisterUser(c.Request.Context(), body.Email, body.Password)
			if err != nil {
				httpx.Fail(c, err)
				return
			}
			httpx.Created(c, gin.H{"user_id": id})
			return
		}
		id, err := d.Auth.RegisterUserWithCode(c.Request.Context(), body.Email, body.Password, body.Code)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.Created(c, gin.H{"user_id": id})
	}
}

type resetPasswordBody struct {
	Email        string `json:"email" binding:"required"`
	NewPassword  string `json:"new_password" binding:"required"`
	Code         string `json:"code" binding:"required"`
	CaptchaToken string `json:"captcha_token"`
}

func resetPassword(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body resetPasswordBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		if err := d.Captcha.Verify(c.Request.Context(), captchasvc.SceneForgot, body.CaptchaToken, c.ClientIP()); err != nil {
			httpx.Fail(c, err)
			return
		}
		if err := d.Auth.ResetPasswordWithCode(c.Request.Context(), body.Email, body.NewPassword, body.Code); err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"ok": true})
	}
}
