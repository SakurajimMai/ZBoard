package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/store"
)

// userTrafficSnapshot returns the current user's upload/download traffic breakdown.
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

// userTrafficLogs returns recent traffic log entries for the current user.
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

// userNodeView is a sanitized node view for end users — no secrets, no private keys.
type userNodeView struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Region          *string    `json:"region"`
	Protocol        string     `json:"protocol"`
	Transport       string     `json:"transport"`
	Status          string     `json:"status"`
	LastHeartbeatAt *time.Time `json:"last_heartbeat_at"`
}

// userNodes returns the list of active nodes visible to the current user.
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

// userTrafficDaily returns per-day upload/download/total bytes over the last
// `days` days (default 30, capped 365).
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

// userResetTraffic charges the user's plan-defined reset_traffic_price from
// users.balance, then zeroes the traffic counters. Users without a plan, or
// plans with price 0, are rejected with 403 to prevent free abuse.
func userResetTraffic(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.MustGet(ctxUserIDKey).(int64)
		u, err := d.Store.FindUserByID(c.Request.Context(), uid)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		if u.PlanID == nil || *u.PlanID == 0 {
			httpx.Fail(c, httpx.NewError(http.StatusForbidden, "plan_required", "请先订阅套餐后再使用此功能"))
			return
		}
		plan, err := d.Store.FindPlanByID(c.Request.Context(), *u.PlanID)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		price := strings.TrimSpace(plan.ResetTrafficPrice)
		if price == "" || price == "0" || price == "0.00" {
			httpx.Fail(c, httpx.NewError(http.StatusForbidden, "reset_disabled", "当前套餐未开放流量重置"))
			return
		}
		ok, err := d.Store.DeductBalanceAtomic(c.Request.Context(), uid, price)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		if !ok {
			httpx.Fail(c, httpx.NewError(http.StatusPaymentRequired, "insufficient_balance",
				fmt.Sprintf("余额不足，本次重置需要 %s", price)))
			return
		}
		if err := d.Store.ResetUserTraffic(c.Request.Context(), uid); err != nil {
			httpx.Fail(c, err)
			return
		}
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "user", ActorID: ptrInt64(uid),
			Action: "user.reset_traffic", Detail: "charged=" + price,
			IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, gin.H{"ok": true, "charged": price})
	}
}

// userResetUUID rotates the user's client_id across every node_users row and
// regenerates sync_config tasks so the new id reaches each node's runtime.
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
			ActorType: "user", ActorID: ptrInt64(uid),
			Action: "user.reset_uuid", IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
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
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	), nil
}
