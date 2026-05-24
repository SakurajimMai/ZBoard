package server

import (
	"strconv"
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
