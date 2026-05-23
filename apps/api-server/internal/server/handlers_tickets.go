package server

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/captchasvc"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/store"
)

// ===== User ticket endpoints =====

type createTicketBody struct {
	Subject      string `json:"subject" binding:"required"`
	Category     string `json:"category"`
	Content      string `json:"content" binding:"required"`
	CaptchaToken string `json:"captcha_token"`
}

func createTicket(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body createTicketBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		if err := d.Captcha.Verify(c.Request.Context(), captchasvc.SceneTicket, body.CaptchaToken, c.ClientIP()); err != nil {
			httpx.Fail(c, err)
			return
		}
		uid := c.MustGet(ctxUserIDKey).(int64)
		category := body.Category
		if category == "" {
			category = "general"
		}
		ticketNo := newTicketNo()
		id, err := d.Store.CreateTicket(c.Request.Context(), ticketNo, uid, body.Subject, category, body.Content)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "user", ActorID: ptrInt64(uid),
			Action: "ticket.create", ResourceType: "ticket", ResourceID: ticketNo,
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.Created(c, gin.H{"ticket_id": id, "ticket_no": ticketNo})
	}
}

func listUserTickets(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.MustGet(ctxUserIDKey).(int64)
		tickets, err := d.Store.ListTicketsByUser(c.Request.Context(), uid)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"items": tickets})
	}
}

func getUserTicketDetail(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.MustGet(ctxUserIDKey).(int64)
		ticketNo := c.Param("ticket_no")
		ticket, err := d.Store.FindTicketByNo(c.Request.Context(), ticketNo)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusNotFound, "not_found", "工单不存在"))
			return
		}
		if ticket.UserID != uid {
			httpx.Fail(c, httpx.NewError(http.StatusForbidden, "forbidden", "无权访问"))
			return
		}
		messages, err := d.Store.ListTicketMessages(c.Request.Context(), ticket.ID)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"ticket": ticket, "messages": messages})
	}
}

type replyTicketBody struct {
	Content string `json:"content" binding:"required"`
}

func replyUserTicket(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.MustGet(ctxUserIDKey).(int64)
		ticketNo := c.Param("ticket_no")
		var body replyTicketBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		ticket, err := d.Store.FindTicketByNo(c.Request.Context(), ticketNo)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusNotFound, "not_found", "工单不存在"))
			return
		}
		if ticket.UserID != uid {
			httpx.Fail(c, httpx.NewError(http.StatusForbidden, "forbidden", "无权访问"))
			return
		}
		if ticket.Status == "closed" {
			httpx.Fail(c, httpx.NewError(http.StatusConflict, "ticket_closed", "工单已关闭"))
			return
		}
		if err := d.Store.AddTicketMessage(c.Request.Context(), ticket.ID, "user", uid, body.Content); err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"ok": true})
	}
}

// ===== Admin ticket endpoints =====

func adminListTickets(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.DefaultQuery("status", "all")
		tickets, err := d.Store.ListAllTickets(c.Request.Context(), status)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"items": tickets})
	}
}

func adminGetTicketDetail(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "id 不合法"))
			return
		}
		ticket, err := d.Store.FindTicketByID(c.Request.Context(), id)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusNotFound, "not_found", "工单不存在"))
			return
		}
		messages, err := d.Store.ListTicketMessages(c.Request.Context(), ticket.ID)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"ticket": ticket, "messages": messages})
	}
}

func adminReplyTicket(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "id 不合法"))
			return
		}
		var body replyTicketBody
		if err := c.ShouldBindJSON(&body); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		ticket, err := d.Store.FindTicketByID(c.Request.Context(), id)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusNotFound, "not_found", "工单不存在"))
			return
		}
		if err := d.Store.AddTicketMessage(c.Request.Context(), ticket.ID, "admin", a.ID, body.Content); err != nil {
			httpx.Fail(c, err)
			return
		}
		// Notify the ticket owner that admin replied
		d.Store.NotifyUser(c.Request.Context(), ticket.UserID, "ticket_reply",
			"工单已回复", "您的工单「"+ticket.Subject+"」收到了新回复",
			"/dashboard/ticket")
		// Auto-reopen if closed
		if ticket.Status == "closed" {
			_ = d.Store.UpdateTicketStatus(c.Request.Context(), ticket.ID, "replied")
		} else {
			_ = d.Store.UpdateTicketStatus(c.Request.Context(), ticket.ID, "replied")
		}
		httpx.OK(c, gin.H{"ok": true})
	}
}

func adminCloseTicket(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "id 不合法"))
			return
		}
		if err := d.Store.CloseTicket(c.Request.Context(), id); err != nil {
			httpx.Fail(c, err)
			return
		}
		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "ticket.close", ResourceType: "ticket", ResourceID: strconv.FormatInt(id, 10),
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, gin.H{"ok": true})
	}
}

func newTicketNo() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return "TK-" + time.Now().UTC().Format("20060102") + "-" + hex.EncodeToString(b)
}
