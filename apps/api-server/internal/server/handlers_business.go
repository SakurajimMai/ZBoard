package server

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/authx"
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
		params := paginationFromQuery(c)
		plans, total, err := d.Store.ListAllPlansPage(c.Request.Context(), params)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{
			"items":     plans,
			"page":      params.Page,
			"page_size": params.PageSize,
			"total":     total,
		})
	}
}

type createPlanBody struct {
	Name              string   `json:"name" binding:"required"`
	Price             string   `json:"price" binding:"required"`
	ResetTrafficPrice string   `json:"reset_traffic_price"`
	DurationDays      int      `json:"duration_days" binding:"required"`
	TrafficLimit      int64    `json:"traffic_limit"`
	DeviceLimit       int      `json:"device_limit"`
	Features          []string `json:"features"`
	NodeGroupID       *int64   `json:"node_group_id"`
	Sort              int      `json:"sort"`
}

type updatePlanBody struct {
	Name              string   `json:"name" binding:"required"`
	Price             string   `json:"price" binding:"required"`
	ResetTrafficPrice string   `json:"reset_traffic_price"`
	DurationDays      int      `json:"duration_days" binding:"required"`
	TrafficLimit      int64    `json:"traffic_limit"`
	DeviceLimit       int      `json:"device_limit"`
	Features          []string `json:"features"`
	NodeGroupID       *int64   `json:"node_group_id"`
	Status            string   `json:"status"`
	Sort              int      `json:"sort"`
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
			Name:              body.Name,
			Price:             body.Price,
			ResetTrafficPrice: body.ResetTrafficPrice,
			DurationDays:      body.DurationDays,
			TrafficLimit:      body.TrafficLimit,
			DeviceLimit:       body.DeviceLimit,
			Features:          normalizeFeatureList(body.Features),
			NodeGroupID:       body.NodeGroupID,
			Sort:              body.Sort,
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

func adminUpdatePlan(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "id 不合法"))
			return
		}
		var body updatePlanBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		status := strings.TrimSpace(body.Status)
		if status == "" {
			status = "active"
		}
		if status != "active" && status != "inactive" {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "套餐状态不合法"))
			return
		}
		if body.DeviceLimit == 0 {
			body.DeviceLimit = 3
		}
		if err := d.Store.UpdatePlan(c.Request.Context(), id, store.UpdatePlanInput{
			Name:              body.Name,
			Price:             body.Price,
			ResetTrafficPrice: body.ResetTrafficPrice,
			DurationDays:      body.DurationDays,
			TrafficLimit:      body.TrafficLimit,
			DeviceLimit:       body.DeviceLimit,
			Features:          normalizeFeatureList(body.Features),
			NodeGroupID:       body.NodeGroupID,
			Status:            status,
			Sort:              body.Sort,
		}); err != nil {
			httpx.Fail(c, err)
			return
		}
		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "plan.update", ResourceType: "plan", ResourceID: strconv.FormatInt(id, 10),
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, gin.H{"ok": true})
	}
}

func normalizeFeatureList(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	return out
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
			"existing": res.Existing,
			"payment":  res.Payment,
			"order_no": res.OrderNo,
			"pay_url":  res.PayURL,
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
		params := paginationFromQuery(c)
		rows, total, err := d.Store.ListAllOrdersPage(c.Request.Context(), params)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"items": rows, "page": params.Page, "page_size": params.PageSize, "total": total})
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
		params := paginationFromQuery(c)
		filter, err := userFilterFromQuery(c)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		rows, total, err := d.Store.ListUsersPageFiltered(c.Request.Context(), params, filter)
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
		httpx.OK(c, gin.H{"items": view, "page": params.Page, "page_size": params.PageSize, "total": total})
	}
}

func userFilterFromQuery(c *gin.Context) (store.UserFilter, error) {
	var f store.UserFilter
	f.Email = c.Query("email")
	f.Status = c.Query("status")
	f.Expires = c.Query("expires")
	if raw := strings.TrimSpace(c.Query("plan_id")); raw != "" && raw != "all" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return f, httpx.NewError(http.StatusBadRequest, "bad_request", "套餐筛选不合法")
		}
		f.PlanID = &id
	}
	if raw := strings.TrimSpace(c.Query("traffic_min")); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return f, httpx.NewError(http.StatusBadRequest, "bad_request", "最小流量不合法")
		}
		f.TrafficMin = &n
	}
	if raw := strings.TrimSpace(c.Query("traffic_max")); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return f, httpx.NewError(http.StatusBadRequest, "bad_request", "最大流量不合法")
		}
		f.TrafficMax = &n
	}
	return f, nil
}

type adminCreateUserBody struct {
	Email        string  `json:"email" binding:"required"`
	Password     string  `json:"password" binding:"required"`
	Balance      string  `json:"balance"`
	PlanID       *int64  `json:"plan_id"`
	ExpiredAt    *string `json:"expired_at"`
	TrafficLimit int64   `json:"traffic_limit"`
	TrafficUsed  int64   `json:"traffic_used"`
	Status       string  `json:"status"`
}

type adminUpdateUserBody struct {
	Email        string  `json:"email" binding:"required"`
	Balance      string  `json:"balance"`
	PlanID       *int64  `json:"plan_id"`
	ExpiredAt    *string `json:"expired_at"`
	TrafficLimit int64   `json:"traffic_limit"`
	TrafficUsed  int64   `json:"traffic_used"`
	Status       string  `json:"status"`
}

func adminCreateUser(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body adminCreateUserBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		email := strings.TrimSpace(strings.ToLower(body.Email))
		if len(body.Password) < 6 {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "密码至少 6 位"))
			return
		}
		status := normalizeUserStatus(body.Status)
		if status == "" {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "用户状态不合法"))
			return
		}
		expiredAt, err := parseOptionalTime(body.ExpiredAt)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "到期时间格式不合法"))
			return
		}
		plan, err := planForUserInput(c.Request.Context(), d.Store, body.PlanID)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		hash, err := authx.HashPassword(body.Password)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		id, err := d.Store.AdminCreateUser(c.Request.Context(), store.AdminCreateUserInput{
			Email:        email,
			PasswordHash: hash,
			Balance:      defaultString(body.Balance, "0.00"),
			PlanID:       body.PlanID,
			ExpiredAt:    expiredAt,
			TrafficLimit: body.TrafficLimit,
			TrafficUsed:  body.TrafficUsed,
			Status:       status,
		})
		if err != nil {
			if store.IsUniqueViolation(err) {
				httpx.Fail(c, httpx.NewError(http.StatusConflict, "email_taken", "邮箱已注册"))
				return
			}
			httpx.Fail(c, err)
			return
		}
		if status == "active" {
			if err := provisionUserOnActiveNodes(c.Request.Context(), d.Store, id, plan); err != nil {
				httpx.Fail(c, err)
				return
			}
		}
		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "user.create", ResourceType: "user", ResourceID: strconv.FormatInt(id, 10),
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.Created(c, gin.H{"user_id": id})
	}
}

func adminUpdateUser(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "id 不合法"))
			return
		}
		var body adminUpdateUserBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		status := normalizeUserStatus(body.Status)
		if status == "" {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "用户状态不合法"))
			return
		}
		expiredAt, err := parseOptionalTime(body.ExpiredAt)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "到期时间格式不合法"))
			return
		}
		plan, err := planForUserInput(c.Request.Context(), d.Store, body.PlanID)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		if err := d.Store.AdminUpdateUser(c.Request.Context(), id, store.AdminUpdateUserInput{
			Email:        strings.TrimSpace(strings.ToLower(body.Email)),
			Balance:      defaultString(body.Balance, "0.00"),
			PlanID:       body.PlanID,
			ExpiredAt:    expiredAt,
			TrafficLimit: body.TrafficLimit,
			TrafficUsed:  body.TrafficUsed,
			Status:       status,
		}); err != nil {
			if store.IsUniqueViolation(err) {
				httpx.Fail(c, httpx.NewError(http.StatusConflict, "email_taken", "邮箱已注册"))
				return
			}
			httpx.Fail(c, err)
			return
		}
		if status == "active" {
			if err := provisionUserOnActiveNodes(c.Request.Context(), d.Store, id, plan); err != nil {
				httpx.Fail(c, err)
				return
			}
			_ = d.Store.SetNodeUserEnabledForUser(c.Request.Context(), id, 1)
		} else {
			_ = d.Store.SetNodeUserEnabledForUser(c.Request.Context(), id, 0)
		}
		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "user.update", ResourceType: "user", ResourceID: strconv.FormatInt(id, 10),
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, gin.H{"ok": true})
	}
}

func normalizeUserStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "", "active":
		return "active"
	case "disabled", "inactive":
		return "disabled"
	default:
		return ""
	}
}

func parseOptionalTime(raw *string) (*time.Time, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	v := strings.TrimSpace(*raw)
	layouts := []string{time.RFC3339, "2006-01-02"}
	for _, layout := range layouts {
		t, err := time.Parse(layout, v)
		if err == nil {
			return &t, nil
		}
	}
	return nil, httpx.NewError(http.StatusBadRequest, "bad_request", "时间格式不合法")
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func planForUserInput(ctx context.Context, st *store.Store, planID *int64) (*store.Plan, error) {
	if planID == nil {
		return nil, nil
	}
	plan, err := st.FindPlanByID(ctx, *planID)
	if err != nil {
		if store.IsNoRows(err) {
			return nil, httpx.NewError(http.StatusBadRequest, "plan_not_found", "套餐不存在")
		}
		return nil, err
	}
	return plan, nil
}

func provisionUserOnActiveNodes(ctx context.Context, st *store.Store, userID int64, plan *store.Plan) error {
	nodes, err := st.ListActiveNodes(ctx)
	if err != nil {
		return err
	}
	clientID, err := newClientIDForServer()
	if err != nil {
		return err
	}
	deviceLimit := 0
	if plan != nil {
		deviceLimit = plan.DeviceLimit
	}
	for _, n := range nodes {
		if err := st.EnsureNodeUserWithLimits(ctx, userID, n.ID, clientID, n.Protocol, 0, deviceLimit); err != nil {
			return err
		}
	}
	return nil
}

func enqueueActiveNodeSync(ctx context.Context, d Deps) error {
	if d.Nodes == nil {
		return httpx.NewError(http.StatusInternalServerError, "node_service_unavailable", "节点服务不可用")
	}
	results, err := d.Nodes.GenerateSyncTaskAll(ctx)
	if err != nil {
		return err
	}
	for _, result := range results {
		if result.Error != "" {
			return httpx.NewError(http.StatusInternalServerError, "node_sync_failed", result.Error)
		}
	}
	return nil
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

type adminBatchUsersBody struct {
	Action  string  `json:"action" binding:"required"`
	UserIDs []int64 `json:"user_ids" binding:"required"`
	Subject string  `json:"subject"`
	Content string  `json:"content"`
}

func adminBatchUsers(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body adminBatchUsersBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		if len(body.UserIDs) == 0 || len(body.UserIDs) > 500 {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "用户数量不合法"))
			return
		}
		action := strings.TrimSpace(body.Action)
		for _, id := range body.UserIDs {
			if id <= 0 {
				httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "用户 ID 不合法"))
				return
			}
			switch action {
			case "enable":
				if err := d.Store.SetUserStatus(c.Request.Context(), id, "active"); err != nil {
					httpx.Fail(c, err)
					return
				}
				_ = d.Store.SetNodeUserEnabledForUser(c.Request.Context(), id, 1)
			case "disable":
				if err := d.Store.SetUserStatus(c.Request.Context(), id, "disabled"); err != nil {
					httpx.Fail(c, err)
					return
				}
				_ = d.Store.SetNodeUserEnabledForUser(c.Request.Context(), id, 0)
			case "reset_subscription":
				tok, err := newSubToken()
				if err != nil {
					httpx.Fail(c, err)
					return
				}
				if err := d.Store.RotateSubToken(c.Request.Context(), id, tok, hashSubToken(tok)); err != nil {
					httpx.Fail(c, err)
					return
				}
			case "send_email":
				if strings.TrimSpace(body.Subject) == "" || strings.TrimSpace(body.Content) == "" {
					httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "邮件标题和内容不能为空"))
					return
				}
				u, err := d.Store.FindUserByID(c.Request.Context(), id)
				if err != nil {
					httpx.Fail(c, err)
					return
				}
				if d.Auth.Mailer == nil {
					httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "mailer_not_configured", "邮件服务未配置"))
					return
				}
				if err := d.Auth.Mailer.SendText(u.Email, body.Subject, body.Content); err != nil {
					httpx.Fail(c, err)
					return
				}
			default:
				httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "批量操作不支持"))
				return
			}
		}
		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "user.batch." + action, ResourceType: "user",
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, gin.H{"ok": true, "count": len(body.UserIDs)})
	}
}

func adminResetUserSubscription(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "id 不合法"))
			return
		}
		tok, err := newSubToken()
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		if err := d.Store.RotateSubToken(c.Request.Context(), id, tok, hashSubToken(tok)); err != nil {
			httpx.Fail(c, err)
			return
		}
		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "user.reset_subscription", ResourceType: "user", ResourceID: strconv.FormatInt(id, 10),
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, gin.H{"token": tok})
	}
}

func adminGetUserSubscription(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "id 不合法"))
			return
		}
		t, err := d.Store.FindActiveSubTokenByUser(c.Request.Context(), id)
		if err != nil && !store.IsNoRows(err) {
			httpx.Fail(c, err)
			return
		}
		if t == nil {
			tok, err := newSubToken()
			if err != nil {
				httpx.Fail(c, err)
				return
			}
			if _, err := d.Store.CreateSubToken(c.Request.Context(), id, tok, hashSubToken(tok)); err != nil {
				httpx.Fail(c, err)
				return
			}
			httpx.OK(c, gin.H{"token": tok})
			return
		}
		httpx.OK(c, gin.H{"token": t.Token})
	}
}

func adminResetUserUUID(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "id 不合法"))
			return
		}
		newID, err := newUserClientID()
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		if err := d.Store.RotateUserClientID(c.Request.Context(), id, newID); err != nil {
			httpx.Fail(c, err)
			return
		}
		nodes, err := d.Store.ListActiveNodes(c.Request.Context())
		if err == nil {
			for _, n := range nodes {
				_, _, _ = d.Nodes.GenerateSyncTask(c.Request.Context(), n.ID)
			}
		}
		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "user.reset_uuid", ResourceType: "user", ResourceID: strconv.FormatInt(id, 10),
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, gin.H{"ok": true, "client_id": newID})
	}
}

func adminResetUserIdentity(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "id 不合法"))
			return
		}
		newID, err := newUserClientID()
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		tok, err := newSubToken()
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		if err := d.Store.RotateUserClientID(c.Request.Context(), id, newID); err != nil {
			httpx.Fail(c, err)
			return
		}
		if err := d.Store.RotateSubToken(c.Request.Context(), id, tok, hashSubToken(tok)); err != nil {
			httpx.Fail(c, err)
			return
		}
		nodes, err := d.Store.ListActiveNodes(c.Request.Context())
		if err == nil {
			for _, n := range nodes {
				_, _, _ = d.Nodes.GenerateSyncTask(c.Request.Context(), n.ID)
			}
		}
		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "user.reset_identity", ResourceType: "user", ResourceID: strconv.FormatInt(id, 10),
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, gin.H{"token": tok, "client_id": newID})
	}
}

func adminListUserOrders(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "id 不合法"))
			return
		}
		rows, err := d.Store.ListOrdersByUser(c.Request.Context(), id, 100)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"items": rows})
	}
}

func adminListUserTrafficLogs(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "id 不合法"))
			return
		}
		rows, err := d.Store.ListTrafficLogsByUser(c.Request.Context(), id, 100)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"items": rows})
	}
}
