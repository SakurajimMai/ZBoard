package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/httpx"
)

type sendEmailCodeBody struct {
	Email   string `json:"email" binding:"required"`
	Purpose string `json:"purpose" binding:"required"` // "register" | "reset_password"
}

func sendEmailCode(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body sendEmailCodeBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
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
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	Code     string `json:"code" binding:"required"`
}

func registerUserWithCode(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body registerWithCodeBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
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
	Email       string `json:"email" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
	Code        string `json:"code" binding:"required"`
}

func resetPassword(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body resetPasswordBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		if err := d.Auth.ResetPasswordWithCode(c.Request.Context(), body.Email, body.NewPassword, body.Code); err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"ok": true})
	}
}
