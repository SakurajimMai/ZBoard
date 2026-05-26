package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
)

func TestAdminCanManageKnowledgeArticlesAndUsersReadPublishedTutorials(t *testing.T) {
	r, _, token := setupAdminCRUDRouter(t)

	create := adminJSON(t, r, token, http.MethodPost, "/api/admin/v1/knowledge", map[string]any{
		"title":    "V2rayN 使用教程",
		"category": "Windows",
		"summary":  "Windows 客户端导入节点订阅",
		"content":  "1. 复制节点订阅链接\n2. 在 V2rayN 中导入\n3. 更新订阅并启用节点",
		"sort":     10,
		"status":   "active",
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create knowledge status=%d body=%s", create.Code, create.Body.String())
	}
	var created struct {
		ID   int64  `json:"id"`
		Slug string `json:"slug"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.ID == 0 || created.Slug == "" {
		t.Fatalf("knowledge create response incomplete: %+v", created)
	}

	hidden := adminJSON(t, r, token, http.MethodPost, "/api/admin/v1/knowledge", map[string]any{
		"title":    "隐藏教程",
		"category": "Windows",
		"content":  "未发布",
		"sort":     20,
		"status":   "inactive",
	})
	if hidden.Code != http.StatusCreated {
		t.Fatalf("create hidden knowledge status=%d body=%s", hidden.Code, hidden.Body.String())
	}

	update := adminJSON(t, r, token, http.MethodPut, "/api/admin/v1/knowledge/"+strconv.FormatInt(created.ID, 10), map[string]any{
		"title":    "V2rayN 快速导入教程",
		"category": "Windows",
		"summary":  "更新后的摘要",
		"content":  "更新后的教程内容",
		"sort":     30,
		"status":   "active",
	})
	if update.Code != http.StatusOK {
		t.Fatalf("update knowledge status=%d body=%s", update.Code, update.Body.String())
	}

	adminList := adminJSON(t, r, token, http.MethodGet, "/api/admin/v1/knowledge", nil)
	if adminList.Code != http.StatusOK {
		t.Fatalf("admin list knowledge status=%d body=%s", adminList.Code, adminList.Body.String())
	}
	var adminGot struct {
		Total int64 `json:"total"`
		Items []struct {
			Title string `json:"title"`
			Sort  int    `json:"sort"`
		} `json:"items"`
	}
	if err := json.Unmarshal(adminList.Body.Bytes(), &adminGot); err != nil {
		t.Fatalf("decode admin knowledge list: %v", err)
	}
	if adminGot.Total != 2 || len(adminGot.Items) != 2 || adminGot.Items[0].Title != "V2rayN 快速导入教程" || adminGot.Items[0].Sort != 30 {
		t.Fatalf("unexpected admin knowledge list: %+v", adminGot)
	}

	publicList := adminJSON(t, r, "", http.MethodGet, "/api/v1/knowledge", nil)
	if publicList.Code != http.StatusOK {
		t.Fatalf("public list knowledge status=%d body=%s", publicList.Code, publicList.Body.String())
	}
	var publicGot struct {
		Items []struct {
			Title    string `json:"title"`
			Category string `json:"category"`
			Slug     string `json:"slug"`
		} `json:"items"`
	}
	if err := json.Unmarshal(publicList.Body.Bytes(), &publicGot); err != nil {
		t.Fatalf("decode public knowledge list: %v", err)
	}
	if len(publicGot.Items) != 1 || publicGot.Items[0].Title != "V2rayN 快速导入教程" || publicGot.Items[0].Slug == "" {
		t.Fatalf("unexpected public knowledge list: %+v", publicGot.Items)
	}

	detail := adminJSON(t, r, "", http.MethodGet, "/api/v1/knowledge/"+publicGot.Items[0].Slug, nil)
	if detail.Code != http.StatusOK {
		t.Fatalf("knowledge detail status=%d body=%s", detail.Code, detail.Body.String())
	}
	var detailGot struct {
		Article struct {
			Title   string `json:"title"`
			Content string `json:"content"`
		} `json:"article"`
	}
	if err := json.Unmarshal(detail.Body.Bytes(), &detailGot); err != nil {
		t.Fatalf("decode knowledge detail: %v", err)
	}
	if detailGot.Article.Content != "更新后的教程内容" {
		t.Fatalf("unexpected knowledge detail: %+v", detailGot.Article)
	}
}
