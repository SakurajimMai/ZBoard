package server

import (
	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/httpx"
)

func adminOverview(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		overview, err := d.Store.AdminOverview(c.Request.Context())
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, overview)
	}
}
