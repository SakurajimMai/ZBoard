package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/store"
)

// ===== Plans =====

func listPlans(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		plans, err := d.Biz.ListActivePlans(c.Request.Context())
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"items": plans})
	}
}

func adminListPlans(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		plans, err := d.Biz.ListAllPlans(c.Request.Context())
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"items": plans})
	}
}

type createPlanBody struct {
	Name         string  `json:"name" binding:"required"`
	Price        string  `json:"price" binding:"required"`
	DurationDays int     `json:"duration_days" binding:"required"`
	TrafficLimit int64   `json:"traffic_limit"`
	DeviceLimit  int     `json:"device_limit"`
	SpeedLimit   int     `json:"speed_limit"`
	NodeGroupID  *int64  `json:"node_group_id"`
	Sort         int     `json:"sort"`
}

func adminCreatePlan(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body createPlanBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		if body.DeviceLimit == 0 {
			body.DeviceLimit = 3
		}
		id, err := d.Biz.CreatePlan(c.Request.Context(), store.CreatePlanInput{
			Name:         body.Name,
			Price:        body.Price,
			DurationDays: body.DurationDays,
			TrafficLimit: body.TrafficLimit,
			DeviceLimit:  body.DeviceLimit,
			SpeedLimit:   body.SpeedLimit,
			NodeGroupID:  body.NodeGroupID,
			Sort:         body.Sort,
		})
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "plan.create", ResourceType: "plan", ResourceID: strconv.FormatInt(id, 10),
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.Created(c, gin.H{"plan_id": id})
	}
}

// ===== Orders =====

type createOrderBody struct {
	PlanID int64 `json:"plan_id" binding:"required"`
}

func createOrder(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body createOrderBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		uid := c.MustGet(ctxUserIDKey).(int64)
		key := c.GetHeader("Idempotency-Key")
		res, err := d.Biz.CreateOrder(c.Request.Context(), uid, body.PlanID, key)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "user", ActorID: ptrInt64(uid),
			Action: "order.create", ResourceType: "order", ResourceID: res.Order.OrderNo,
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		status := http.StatusCreated
		if res.Existing {
			status = http.StatusOK
		}
		c.JSON(status, gin.H{"existing": res.Existing, "order": res.Order})
	}
}

func payOrder(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		orderNo := c.Param("order_no")
		uid := c.MustGet(ctxUserIDKey).(int64)
		key := c.GetHeader("Idempotency-Key")
		res, err := d.Biz.StartPayment(c.Request.Context(), uid, orderNo, key)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "user", ActorID: ptrInt64(uid),
			Action: "payment.start", ResourceType: "order", ResourceID: orderNo,
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, gin.H{
			"existing":   res.Existing,
			"payment":    res.Payment,
			"order_no":   res.OrderNo,
			"pay_url":    res.PayURL,
		})
	}
}

// ===== Mock callback =====

type mockCallbackBody struct {
	EventID   string `json:"event_id" binding:"required"`
	OrderNo   string `json:"order_no" binding:"required"`
	PaymentNo string `json:"payment_no" binding:"required"`
}

func mockPaymentCallback(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body mockCallbackBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		raw, _ := c.GetRawData()
		if err := d.Biz.HandleMockCallback(c.Request.Context(), body.EventID, body.OrderNo, body.PaymentNo, "", string(raw)); err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"ok": true})
	}
}

// ===== Admin views =====

func adminListOrders(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := d.Store.ListAllOrders(c.Request.Context(), 200)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"items": rows})
	}
}

func adminListPayments(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := d.Store.ListPayments(c.Request.Context(), 200)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"items": rows})
	}
}

func adminListPaymentCallbacks(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := d.Store.ListPaymentCallbacks(c.Request.Context(), 200)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"items": rows})
	}
}

func adminListUsers(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := d.Store.ListUsers(c.Request.Context(), 200)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		view := make([]gin.H, 0, len(rows))
		for _, u := range rows {
			view = append(view, gin.H{
				"id":            u.ID,
				"email":         u.Email,
				"plan_id":       u.PlanID,
				"expired_at":    u.ExpiredAt,
				"traffic_limit": u.TrafficLimit,
				"traffic_used":  u.TrafficUsed,
				"status":        u.Status,
				"created_at":    u.CreatedAt,
			})
		}
		httpx.OK(c, gin.H{"items": view})
	}
}

func adminUserDisable(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "id 不合法"))
			return
		}
		if err := d.Store.SetUserStatus(c.Request.Context(), id, "disabled"); err != nil {
			httpx.Fail(c, err)
			return
		}
		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "user.disable", ResourceType: "user", ResourceID: strconv.FormatInt(id, 10),
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, gin.H{"ok": true})
	}
}

func adminUserEnable(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "id 不合法"))
			return
		}
		if err := d.Store.SetUserStatus(c.Request.Context(), id, "active"); err != nil {
			httpx.Fail(c, err)
			return
		}
		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "user.enable", ResourceType: "user", ResourceID: strconv.FormatInt(id, 10),
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, gin.H{"ok": true})
	}
}
