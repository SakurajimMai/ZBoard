package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/zboard/api-server/internal/agentauth"
	"github.com/zboard/api-server/internal/authsvc"
	"github.com/zboard/api-server/internal/bizsvc"
	"github.com/zboard/api-server/internal/captchasvc"
	"github.com/zboard/api-server/internal/nodesvc"
	"github.com/zboard/api-server/internal/payment/registry"
	"github.com/zboard/api-server/internal/store"
	"github.com/zboard/api-server/internal/worker"
)

// Deps bundles everything the HTTP layer needs.
type Deps struct {
	DB          *sqlx.DB
	Store       *store.Store
	Auth        *authsvc.Service
	Biz         *bizsvc.Service
	Nodes       *nodesvc.Service
	Worker      *worker.Service
	Payments    *registry.Registry
	Captcha     *captchasvc.Service
	CORSOrigins []string
}

func New(d Deps) *gin.Engine {
	if d.Captcha == nil && d.Store != nil {
		d.Captcha = captchasvc.New(d.Store)
	}
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors(d.CORSOrigins))

	r.GET("/health", func(c *gin.Context) {
		if err := d.DB.PingContext(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "down", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	{
		api.POST("/auth/register", registerUser(d))
		api.POST("/auth/send-email-code", sendEmailCode(d))
		api.POST("/auth/register-with-code", registerUserWithCode(d))
		api.POST("/auth/reset-password", resetPassword(d))
		api.POST("/auth/login", loginUser(d))
		api.GET("/settings", publicSettings(d))
		api.GET("/announcements", listActiveAnnouncements(d))
		api.GET("/plans", listPlans(d))
		api.POST("/payments/mock-callback", mockPaymentCallback(d))
		api.POST("/payments/:provider/callback", paymentCallback(d))
		api.GET("/payment-methods", listAvailablePaymentMethods(d))

		authed := api.Group("")
		authed.Use(userAuth(d.Auth))
		{
			authed.GET("/me", currentUser(d))
			authed.POST("/me/password", changeUserPassword(d))
			authed.DELETE("/me", deleteUserAccount(d))
			authed.POST("/auth/logout", logoutUser(d))
			authed.POST("/orders", createOrder(d))
			authed.POST("/orders/:order_no/pay", createPaymentWithProvider(d, d.Payments))
			authed.GET("/subscription", subToken(d))
			authed.POST("/subscription/reset-token", subResetToken(d))
			authed.GET("/traffic/snapshot", userTrafficSnapshot(d))
			authed.GET("/traffic/logs", userTrafficLogs(d))
			authed.GET("/traffic/daily", userTrafficDaily(d))
			authed.POST("/traffic/reset", userResetTraffic(d))
			authed.POST("/uuid/reset", userResetUUID(d))
			authed.GET("/nodes", userNodes(d))
			authed.GET("/tickets", listUserTickets(d))
			authed.POST("/tickets", createTicket(d))
			authed.GET("/tickets/:ticket_no", getUserTicketDetail(d))
			authed.POST("/tickets/:ticket_no/reply", replyUserTicket(d))
			authed.GET("/notifications", listNotifications(d))
			authed.GET("/notifications/unread", countUnreadNotifications(d))
			authed.POST("/notifications/:id/read", markNotificationRead(d))
			authed.POST("/notifications/read-all", markAllNotificationsRead(d))
		}
	}

	r.GET("/api/sub/:token", subRender(d))

	admin := r.Group("/api/admin/v1")
	{
		admin.POST("/auth/bootstrap", adminBootstrap(d))
		admin.POST("/auth/login", adminLogin(d))

		authed := admin.Group("")
		authed.Use(adminAuth(d.Auth))
		{
			authed.GET("/auth/me", adminMe(d))
			authed.POST("/auth/logout", adminLogout(d))
			authed.GET("/overview", adminOverview(d))
			authed.GET("/audit-logs", adminAuditLogs(d))
			authed.GET("/users", adminListUsers(d))
			authed.POST("/users/batch", adminBatchUsers(d))
			authed.POST("/users", adminCreateUser(d))
			authed.PUT("/users/:id", adminUpdateUser(d))
			authed.POST("/users/:id/disable", adminUserDisable(d))
			authed.POST("/users/:id/enable", adminUserEnable(d))
			authed.GET("/users/:id/subscription", adminGetUserSubscription(d))
			authed.POST("/users/:id/reset-subscription", adminResetUserSubscription(d))
			authed.POST("/users/:id/reset-uuid", adminResetUserUUID(d))
			authed.POST("/users/:id/reset-identity", adminResetUserIdentity(d))
			authed.GET("/users/:id/orders", adminListUserOrders(d))
			authed.GET("/users/:id/traffic-logs", adminListUserTrafficLogs(d))
			authed.GET("/announcements", adminListAnnouncements(d))
			authed.POST("/announcements", adminCreateAnnouncement(d))
			authed.PUT("/announcements/:id", adminUpdateAnnouncement(d))
			authed.DELETE("/announcements/:id", adminDeleteAnnouncement(d))
			authed.GET("/plans", adminListPlans(d))
			authed.POST("/plans", adminCreatePlan(d))
			authed.PUT("/plans/:id", adminUpdatePlan(d))
			authed.GET("/settings", adminGetSettings(d))
			authed.PUT("/settings", adminUpdateSettings(d))
			authed.POST("/settings/test-email", adminSendTestEmail(d))
			authed.GET("/orders", adminListOrders(d))
			authed.GET("/payments", adminListPayments(d))
			authed.GET("/payment-callbacks", adminListPaymentCallbacks(d))
			authed.GET("/nodes", adminListNodes(d))
			authed.POST("/reality/generate", adminGenerateRealityConfig())
			authed.POST("/nodes", adminCreateNode(d))
			authed.PUT("/nodes/:id", adminUpdateNode(d))
			authed.POST("/nodes/:id/sync-config", adminSyncNodeConfig(d))
			authed.POST("/nodes/sync-config-all", adminSyncAllNodeConfigs(d))
			authed.GET("/nodes/:id/runtime-configs", adminListRuntimeConfigs(d))
			authed.POST("/runtime-configs/:version/rollback", adminRollbackRuntimeConfig(d))
			authed.GET("/node-tasks", adminListNodeTasks(d))
			authed.GET("/traffic/users", adminListTrafficSnapshots(d))
			authed.GET("/traffic/logs", adminListTrafficLogs(d))
			authed.POST("/workers/maintenance/run", adminRunMaintenance(d))
			authed.GET("/payment-providers", adminListPaymentProviders(d))
			authed.POST("/payment-providers", adminCreatePaymentProvider(d))
			authed.PUT("/payment-providers/:id", adminUpdatePaymentProvider(d))
			authed.DELETE("/payment-providers/:id", adminDeletePaymentProvider(d))
			authed.GET("/tickets", adminListTickets(d))
			authed.GET("/tickets/:id", adminGetTicketDetail(d))
			authed.POST("/tickets/:id/reply", adminReplyTicket(d))
			authed.POST("/tickets/:id/close", adminCloseTicket(d))
		}
	}

	agent := r.Group("/api/agent/v1")
	agent.Use(agentauth.HMAC(d.Store))
	{
		agent.POST("/register", agentRegister(d))
		agent.POST("/heartbeat", agentHeartbeat(d))
		agent.POST("/tasks/pull", agentPullTasks(d))
		agent.POST("/tasks/:task_id/result", agentTaskResult(d))
		agent.POST("/traffic/report", agentTrafficReport(d))
	}
	return r
}
