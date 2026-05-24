package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/store"
)

type announcementBody struct {
	Title    string  `json:"title" binding:"required"`
	Content  string  `json:"content" binding:"required"`
	Popup    bool    `json:"popup"`
	Priority int     `json:"priority"`
	Status   string  `json:"status"`
	StartsAt *string `json:"starts_at"`
	EndsAt   *string `json:"ends_at"`
}

func adminListAnnouncements(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		params := paginationFromQuery(c)
		rows, total, err := d.Store.ListAnnouncementsPage(c.Request.Context(), params)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"items": rows, "page": params.Page, "page_size": params.PageSize, "total": total})
	}
}

func adminCreateAnnouncement(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		in, ok := bindAnnouncement(c)
		if !ok {
			return
		}
		id, err := d.Store.CreateAnnouncement(c.Request.Context(), in)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "announcement.create", ResourceType: "announcement", ResourceID: strconv.FormatInt(id, 10),
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.Created(c, gin.H{"id": id})
	}
}

func adminUpdateAnnouncement(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "id 不合法"))
			return
		}
		in, ok := bindAnnouncement(c)
		if !ok {
			return
		}
		if err := d.Store.UpdateAnnouncement(c.Request.Context(), id, in); err != nil {
			httpx.Fail(c, err)
			return
		}
		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "announcement.update", ResourceType: "announcement", ResourceID: strconv.FormatInt(id, 10),
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, gin.H{"ok": true})
	}
}

func adminDeleteAnnouncement(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "id 不合法"))
			return
		}
		if err := d.Store.DeleteAnnouncement(c.Request.Context(), id); err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"ok": true})
	}
}

func listActiveAnnouncements(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := d.Store.ListActiveAnnouncements(c.Request.Context(), 20)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"items": rows})
	}
}

func bindAnnouncement(c *gin.Context) (store.AnnouncementInput, bool) {
	var body announcementBody
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
		return store.AnnouncementInput{}, false
	}
	status := strings.TrimSpace(body.Status)
	if status == "" {
		status = "active"
	}
	if status != "active" && status != "inactive" {
		httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "公告状态不合法"))
		return store.AnnouncementInput{}, false
	}
	startsAt, err := parseOptionalTime(body.StartsAt)
	if err != nil {
		httpx.Fail(c, err)
		return store.AnnouncementInput{}, false
	}
	endsAt, err := parseOptionalTime(body.EndsAt)
	if err != nil {
		httpx.Fail(c, err)
		return store.AnnouncementInput{}, false
	}
	return store.AnnouncementInput{
		Title:    strings.TrimSpace(body.Title),
		Content:  strings.TrimSpace(body.Content),
		Popup:    body.Popup,
		Priority: body.Priority,
		Status:   status,
		StartsAt: startsAt,
		EndsAt:   endsAt,
	}, true
}
