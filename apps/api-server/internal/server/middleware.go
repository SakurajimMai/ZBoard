package server

import (
	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/authsvc"
	"github.com/zboard/api-server/internal/httpx"
)

const (
	ctxUserIDKey = "zb_user_id"
	ctxAdminKey  = "zb_admin"
	ctxAdminTok  = "zb_admin_tok"
	ctxUserTok   = "zb_user_tok"
)

func userAuth(svc *authsvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := authsvc.ExtractBearer(c.GetHeader("Authorization"))
		uid, err := svc.ResolveUserToken(c.Request.Context(), token)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		c.Set(ctxUserIDKey, uid)
		c.Set(ctxUserTok, token)
		c.Next()
	}
}

func adminAuth(svc *authsvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := authsvc.ExtractBearer(c.GetHeader("Authorization"))
		a, err := svc.ResolveAdminToken(c.Request.Context(), token)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		c.Set(ctxAdminKey, a)
		c.Set(ctxAdminTok, token)
		c.Next()
	}
}
