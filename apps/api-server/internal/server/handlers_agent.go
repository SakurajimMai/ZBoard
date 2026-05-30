package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/agentauth"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/store"
)

// agentRegister marks node_agents as active and updates version/os info.
type agentRegisterBody struct {
	AgentVersion string `json:"agent_version"`
	OSInfo       string `json:"os_info"`
	RuntimeInfo  string `json:"runtime_info"`
}

func agentRegister(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		body := getAgentBody(c)
		var in agentRegisterBody
		if len(body) > 0 {
			if err := json.Unmarshal(body, &in); err != nil {
				httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
				return
			}
		}
		nodeID := c.MustGet(agentauth.NodeIDCtxKey).(int64)
		if err := d.Store.MarkAgentRegistered(c.Request.Context(), nodeID, in.AgentVersion, in.OSInfo, in.RuntimeInfo); err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"ok": true})
	}
}

type agentHeartbeatBody struct {
	AgentVersion  string `json:"agent_version"`
	RuntimeStatus string `json:"runtime_status"`
	RuntimeInfo   string `json:"runtime_info"`
	SystemLoad    string `json:"system_load"`
}

func agentHeartbeat(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		body := getAgentBody(c)
		var in agentHeartbeatBody
		if len(body) > 0 {
			if err := json.Unmarshal(body, &in); err != nil {
				httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
				return
			}
		}
		nodeID := c.MustGet(agentauth.NodeIDCtxKey).(int64)
		if err := d.Store.RecordHeartbeat(c.Request.Context(), store.HeartbeatInput{
			NodeID:        nodeID,
			AgentVersion:  in.AgentVersion,
			RuntimeStatus: in.RuntimeStatus,
			RuntimeInfo:   in.RuntimeInfo,
			SystemLoad:    in.SystemLoad,
			ReportedAt:    time.Now().UTC(),
		}); err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"ok": true})
	}
}

func agentPullTasks(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		nodeID := c.MustGet(agentauth.NodeIDCtxKey).(int64)
		tasks, err := d.Store.PullTasksForNode(c.Request.Context(), nodeID, 10)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		// Inline runtime config bodies for sync_config tasks so the agent doesn't
		// need a separate fetch.
		out := make([]gin.H, 0, len(tasks))
		for _, t := range tasks {
			row := gin.H{
				"task_id":   t.TaskID,
				"task_type": t.TaskType,
				"payload":   json.RawMessage(t.Payload),
				"retry":     t.RetryCount,
			}
			if t.TaskType == "sync_config" {
				var p struct {
					Version string `json:"version"`
				}
				if err := json.Unmarshal([]byte(t.Payload), &p); err == nil && p.Version != "" {
					rc, err := d.Store.FindRuntimeConfigByVersion(c.Request.Context(), p.Version)
					if err == nil {
						row["runtime_config"] = json.RawMessage(rc.ConfigJSON)
					}
				}
			}
			out = append(out, row)
		}
		httpx.OK(c, gin.H{"tasks": out})
	}
}

type agentTaskResultBody struct {
	Status       string `json:"status"`
	FailedReason string `json:"failed_reason"`
}

func agentTaskResult(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		taskID := c.Param("task_id")
		nodeID := c.MustGet(agentauth.NodeIDCtxKey).(int64)
		body := getAgentBody(c)
		var in agentTaskResultBody
		if len(body) > 0 {
			if err := json.Unmarshal(body, &in); err != nil {
				httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
				return
			}
		}
		if in.Status != "success" && in.Status != "failed" {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "status 必须是 success 或 failed"))
			return
		}
		// HMAC 只能证明请求由哪个节点签名，URL 里的 task_id 仍需校验归属。
		// 否则节点 A 可以完成节点 B 的任务，把对方任务静默改成成功或失败。
		task, findErr := d.Store.FindNodeTaskByTaskID(c.Request.Context(), taskID)
		if findErr != nil {
			if store.IsNoRows(findErr) {
				httpx.Fail(c, httpx.NewError(http.StatusNotFound, "task_not_found", "任务不存在"))
				return
			}
			httpx.Fail(c, findErr)
			return
		}
		if task.NodeID != nodeID {
			httpx.Fail(c, httpx.NewError(http.StatusForbidden, "task_node_mismatch", "任务不属于当前节点"))
			return
		}
		if err := d.Store.CompleteTask(c.Request.Context(), taskID, in.Status, in.FailedReason); err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"ok": true})
	}
}

type agentTrafficReportBody struct {
	Items []struct {
		UserID        int64 `json:"user_id"`
		UploadDelta   int64 `json:"upload_delta"`
		DownloadDelta int64 `json:"download_delta"`
	} `json:"items"`
}

func agentTrafficReport(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		body := getAgentBody(c)
		var in agentTrafficReportBody
		if err := json.Unmarshal(body, &in); err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", err.Error()))
			return
		}
		nodeID := c.MustGet(agentauth.NodeIDCtxKey).(int64)
		// 只接受已分配且仍启用的用户流量，避免恶意或失陷 agent 把任意用户的
		// 配额刷掉，或给已被 worker 因到期/超额禁用的账号继续记账。
		allowed, err := d.Store.ListEnabledNodeUserIDsByNode(c.Request.Context(), nodeID)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		deltas := make([]store.TrafficDelta, 0, len(in.Items))
		for _, it := range in.Items {
			if it.UploadDelta < 0 || it.DownloadDelta < 0 {
				continue
			}
			if _, ok := allowed[it.UserID]; !ok {
				continue
			}
			deltas = append(deltas, store.TrafficDelta{
				UserID:        it.UserID,
				NodeID:        nodeID,
				UploadDelta:   it.UploadDelta,
				DownloadDelta: it.DownloadDelta,
			})
		}
		if len(deltas) == 0 && len(in.Items) > 0 {
			// 全部条目都被拒绝时返回错误，避免配置错误或攻击请求得到静默 200。
			httpx.Fail(c, httpx.NewError(http.StatusForbidden, "traffic_user_not_assigned", "上报用户不在该节点下"))
			return
		}
		if err := d.Store.RecordTraffic(c.Request.Context(), deltas); err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"ok": true, "accepted": len(deltas)})
	}
}

func getAgentBody(c *gin.Context) []byte {
	if v, ok := c.Get(agentauth.BodyCtxKey); ok {
		if b, ok := v.([]byte); ok {
			return b
		}
	}
	return nil
}

// ===== Admin runtime-config endpoints =====

func adminSyncNodeConfig(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		nodeID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "节点 ID 不合法"))
			return
		}
		taskID, version, err := d.Nodes.GenerateSyncTask(c.Request.Context(), nodeID)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "node.sync_config", ResourceType: "node", ResourceID: idStr,
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, gin.H{"task_id": taskID, "version": version})
	}
}

func adminListRuntimeConfigs(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		nodeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			httpx.Fail(c, httpx.NewError(http.StatusBadRequest, "bad_request", "节点 ID 不合法"))
			return
		}
		rows, err := d.Store.ListRuntimeConfigs(c.Request.Context(), nodeID)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		// Strip the bulky JSON body for list view; clients fetch a single
		// version when they want the body.
		view := make([]gin.H, 0, len(rows))
		for _, r := range rows {
			view = append(view, gin.H{
				"id":          r.ID,
				"node_id":     r.NodeID,
				"version":     r.Version,
				"config_hash": r.ConfigHash,
				"status":      r.Status,
				"applied_at":  r.AppliedAt,
				"created_at":  r.CreatedAt,
			})
		}
		httpx.OK(c, gin.H{"items": view})
	}
}

func adminRollbackRuntimeConfig(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		version := c.Param("version")
		taskID, err := d.Nodes.Rollback(c.Request.Context(), version)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "runtime_config.rollback", ResourceType: "runtime_config", ResourceID: version,
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, gin.H{"task_id": taskID})
	}
}

func adminSyncAllNodeConfigs(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		results, err := d.Nodes.GenerateSyncTaskAll(c.Request.Context())
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		ok, failed := 0, 0
		for _, r := range results {
			if r.Error == "" {
				ok++
			} else {
				failed++
			}
		}
		a := c.MustGet(ctxAdminKey).(*store.AdminUser)
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "admin", ActorID: ptrInt64(a.ID),
			Action: "node.sync_config_all", ResourceType: "node",
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, gin.H{"results": results, "ok": ok, "failed": failed, "total": len(results)})
	}
}

func adminListNodeTasks(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := d.Store.ListNodeTasks(c.Request.Context(), 200)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"items": rows})
	}
}

func adminListTrafficSnapshots(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := d.Store.ListTrafficSnapshots(c.Request.Context(), 200)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"items": rows})
	}
}

func adminListTrafficLogs(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := d.Store.ListTrafficLogs(c.Request.Context(), 200)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"items": rows})
	}
}
