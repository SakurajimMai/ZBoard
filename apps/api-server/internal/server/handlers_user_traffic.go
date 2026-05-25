package server

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/store"
)

// userTrafficSnapshot 返回当前用户的上传、下载和总流量快照。
func userTrafficSnapshot(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.MustGet(ctxUserIDKey).(int64)
		snap, err := d.Store.FindTrafficSnapshotByUser(c.Request.Context(), uid)
		if err != nil {
			if store.IsNoRows(err) {
				httpx.OK(c, gin.H{"snapshot": gin.H{
					"upload_total":   0,
					"download_total": 0,
					"total_used":     0,
					"traffic_limit":  0,
				}})
				return
			}
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"snapshot": snap})
	}
}

// userTrafficLogs 返回当前用户最近的流量明细。
func userTrafficLogs(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.MustGet(ctxUserIDKey).(int64)
		limit := 100
		if v := c.Query("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
				limit = n
			}
		}
		logs, err := d.Store.ListTrafficLogsByUser(c.Request.Context(), uid, limit)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"items": logs})
	}
}

type userNodeView struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Region          *string    `json:"region"`
	Protocol        string     `json:"protocol"`
	Transport       string     `json:"transport"`
	Status          string     `json:"status"`
	LastHeartbeatAt *time.Time `json:"last_heartbeat_at"`
}

// userNodes 返回用户端可见节点，过滤掉密钥和私有配置。
func userNodes(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		nodes, err := d.Store.ListActiveNodes(c.Request.Context())
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		now := time.Now().UTC()
		views := make([]gin.H, 0, len(nodes))
		for _, n := range nodes {
			health := "online"
			if n.LastHeartbeatAt == nil || now.Sub(*n.LastHeartbeatAt) > 120*time.Second {
				health = "offline"
			}
			views = append(views, gin.H{
				"id":                n.ID,
				"name":              n.Name,
				"region":            n.Region,
				"protocol":          n.Protocol,
				"transport":         n.Transport,
				"status":            health,
				"last_heartbeat_at": n.LastHeartbeatAt,
			})
		}
		httpx.OK(c, gin.H{"items": views})
	}
}

// userTrafficDaily 返回最近 days 天的每日流量聚合。
func userTrafficDaily(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.MustGet(ctxUserIDKey).(int64)
		days := 30
		if v := c.Query("days"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 365 {
				days = n
			}
		}
		rows, err := d.Store.ListDailyTrafficByUser(c.Request.Context(), uid, days)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"items": rows, "days": days})
	}
}

// userResetTraffic 创建流量重置支付订单；支付回调成功后才真正清零流量。
func userResetTraffic(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.MustGet(ctxUserIDKey).(int64)
		res, err := d.Biz.CreateTrafficResetOrder(c.Request.Context(), uid)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType:    "user",
			ActorID:      ptrInt64(uid),
			Action:       "user.reset_traffic_order",
			ResourceType: "order",
			ResourceID:   res.Order.OrderNo,
			Detail:       "amount=" + res.Order.Amount,
			IP:           c.ClientIP(),
			UserAgent:    c.Request.UserAgent(),
		})
		httpx.Created(c, gin.H{"order": res.Order})
	}
}

// userResetUUID 轮换用户在全部节点映射中的 client_id，并下发节点同步任务。
func userResetUUID(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.MustGet(ctxUserIDKey).(int64)
		newID, err := newUserClientID()
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		if err := d.Store.RotateUserClientID(c.Request.Context(), uid, newID); err != nil {
			httpx.Fail(c, err)
			return
		}
		nodes, err := d.Store.ListActiveNodes(c.Request.Context())
		if err == nil {
			for _, n := range nodes {
				_, _, _ = d.Nodes.GenerateSyncTask(c.Request.Context(), n.ID)
			}
		}
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "user",
			ActorID:   ptrInt64(uid),
			Action:    "user.reset_uuid",
			IP:        c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, gin.H{"ok": true, "client_id": newID})
	}
}

func newUserClientID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmtUUID(b), nil
}

func fmtUUID(b []byte) string {
	return hex.EncodeToString(b[0:4]) + "-" +
		hex.EncodeToString(b[4:6]) + "-" +
		hex.EncodeToString(b[6:8]) + "-" +
		hex.EncodeToString(b[8:10]) + "-" +
		hex.EncodeToString(b[10:16])
}
