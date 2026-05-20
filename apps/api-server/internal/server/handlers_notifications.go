package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/httpx"
)

func listNotifications(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.MustGet(ctxUserIDKey).(int64)
		items, err := d.Store.ListNotifications(c.Request.Context(), uid, 50)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		unread, _ := d.Store.CountUnreadNotifications(c.Request.Context(), uid)
		httpx.OK(c, gin.H{"items": items, "unread": unread})
	}
}

func countUnreadNotifications(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.MustGet(ctxUserIDKey).(int64)
		count, err := d.Store.CountUnreadNotifications(c.Request.Context(), uid)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"unread": count})
	}
}

func markNotificationRead(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.MustGet(ctxUserIDKey).(int64)
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "id 不合法"))
			return
		}
		if err := d.Store.MarkNotificationRead(c.Request.Context(), id, uid); err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"ok": true})
	}
}

func markAllNotificationsRead(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.MustGet(ctxUserIDKey).(int64)
		if err := d.Store.MarkAllNotificationsRead(c.Request.Context(), uid); err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"ok": true})
	}
}
