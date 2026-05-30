package server

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/store"
)

type knowledgeBody struct {
	Title    string `json:"title" binding:"required"`
	Category string `json:"category"`
	Summary  string `json:"summary"`
	Content  string `json:"content" binding:"required"`
	Sort     int    `json:"sort"`
	Status   string `json:"status"`
}

func adminListKnowledge(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		params := paginationFromQuery(c)
		rows, total, err := d.Store.ListKnowledgeArticlesPage(
			c.Request.Context(),
			params,
			c.Query("category"),
			c.Query("status"),
		)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"items": rows, "page": params.Page, "page_size": params.PageSize, "total": total})
	}
}

func adminCreateKnowledge(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		in, ok := bindKnowledge(c)
		if !ok {
			return
		}
		var id int64
		var err error
		for i := 0; i < 3; i++ {
			in.Slug = newKnowledgeSlug()
			id, err = d.Store.CreateKnowledgeArticle(c.Request.Context(), in)
			if err == nil {
				break
			}
			if !store.IsUniqueViolation(err) {
				break
			}
		}
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "knowledge.create", ResourceType: "knowledge", ResourceID: strconv.FormatInt(id, 10),
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.Created(c, gin.H{"id": id, "slug": in.Slug})
	}
}

func adminUpdateKnowledge(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "id 不合法"))
			return
		}
		in, ok := bindKnowledge(c)
		if !ok {
			return
		}
		if err := d.Store.UpdateKnowledgeArticle(c.Request.Context(), id, in); err != nil {
			httpx.Fail(c, err)
			return
		}
		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "knowledge.update", ResourceType: "knowledge", ResourceID: strconv.FormatInt(id, 10),
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, gin.H{"ok": true})
	}
}

func adminDeleteKnowledge(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "id 不合法"))
			return
		}
		if err := d.Store.DeleteKnowledgeArticle(c.Request.Context(), id); err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"ok": true})
	}
}

func listActiveKnowledge(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := d.Store.ListActiveKnowledgeArticles(c.Request.Context(), c.Query("category"), 100)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"items": rows})
	}
}

func getActiveKnowledge(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		article, err := d.Store.FindActiveKnowledgeArticleBySlug(c.Request.Context(), c.Param("slug"))
		if err != nil {
			if store.IsNoRows(err) {
				httpx.Fail(c, httpx.NewError(http.StatusNotFound, "knowledge_not_found", "教程不存在或未发布"))
				return
			}
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"article": article})
	}
}

func bindKnowledge(c *gin.Context) (store.KnowledgeInput, bool) {
	var body knowledgeBody
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
		return store.KnowledgeInput{}, false
	}
	if !validateTextLen(c, "title", body.Title, maxTitleRunes) ||
		!validateTextLen(c, "summary", body.Summary, maxSummaryRunes) ||
		!validateTextLen(c, "content", body.Content, maxContentRunes) ||
		!validateTextLen(c, "category", body.Category, maxTitleRunes) {
		return store.KnowledgeInput{}, false
	}
	status := strings.TrimSpace(body.Status)
	if status == "" {
		status = "active"
	}
	if status != "active" && status != "inactive" {
		httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "教程状态不合法"))
		return store.KnowledgeInput{}, false
	}
	title := strings.TrimSpace(body.Title)
	content := strings.TrimSpace(body.Content)
	if title == "" || content == "" {
		httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "请填写标题和内容"))
		return store.KnowledgeInput{}, false
	}
	category := strings.TrimSpace(body.Category)
	if category == "" {
		category = "通用教程"
	}
	return store.KnowledgeInput{
		Title:    title,
		Category: category,
		Summary:  strings.TrimSpace(body.Summary),
		Content:  content,
		Sort:     body.Sort,
		Status:   status,
	}, true
}

func newKnowledgeSlug() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "kb-" + strconv.FormatInt(store.Now().UnixNano(), 36)
	}
	return "kb-" + hex.EncodeToString(b[:])
}
