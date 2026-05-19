package server

import (
	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/store"
)

func adminRunMaintenance(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		res, err := d.Worker.Run(c.Request.Context())
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "worker.maintenance", IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, res)
	}
}
