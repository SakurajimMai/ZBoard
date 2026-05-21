package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/authx"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/store"
	"github.com/zboard/api-server/internal/subrender"
)

func subToken(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.MustGet(ctxUserIDKey).(int64)
		t, err := d.Store.FindActiveSubTokenByUser(c.Request.Context(), uid)
		if err != nil && !store.IsNoRows(err) {
			httpx.Fail(c, err)
			return
		}
		if t == nil {
			tok, err := newSubToken()
			if err != nil {
				httpx.Fail(c, err)
				return
			}
			if _, err := d.Store.CreateSubToken(c.Request.Context(), uid, tok, hashSubToken(tok)); err != nil {
				httpx.Fail(c, err)
				return
			}
			httpx.OK(c, gin.H{"token": tok})
			return
		}
		httpx.OK(c, gin.H{"token": t.Token})
	}
}

func subResetToken(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.MustGet(ctxUserIDKey).(int64)
		tok, err := newSubToken()
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		if err := d.Store.RotateSubToken(c.Request.Context(), uid, tok, hashSubToken(tok)); err != nil {
			httpx.Fail(c, err)
			return
		}
		_ = d.Store.WriteAudit(c.Request.Context(), store.AuditEntry{
			ActorType: "user", ActorID: ptrInt64(uid),
			Action: "subscription.reset_token", IP: c.ClientIP(), UserAgent: c.Request.UserAgent(),
		})
		httpx.OK(c, gin.H{"token": tok})
	}
}

func subRender(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Param("token")
		hash := hashSubToken(token)
		target := c.DefaultQuery("target", "base64")
		if !subscriptionTargetEnabled(c.Request.Context(), d.Store, target) {
			_ = d.Store.LogSubAccess(c.Request.Context(), nil, hash, target, c.ClientIP(), c.Request.UserAgent(), "deny", "target_disabled")
			httpx.Fail(c, httpx.NewError(http.StatusForbidden, "subscription_target_disabled", "该订阅格式已关闭"))
			return
		}

		t, err := d.Store.FindActiveSubTokenByHash(c.Request.Context(), hash)
		if err != nil {
			_ = d.Store.LogSubAccess(c.Request.Context(), nil, hash, target, c.ClientIP(), c.Request.UserAgent(), "deny", "token_not_found")
			httpx.Fail(c, httpx.NewError(http.StatusNotFound, "sub_token_invalid", "订阅 token 无效"))
			return
		}
		u, err := d.Store.FindUserByID(c.Request.Context(), t.UserID)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		if u.Status != "active" {
			_ = d.Store.LogSubAccess(c.Request.Context(), &t.UserID, hash, target, c.ClientIP(), c.Request.UserAgent(), "deny", "user_disabled")
			httpx.Fail(c, httpx.NewError(http.StatusForbidden, "user_disabled", "账号已禁用"))
			return
		}
		if u.ExpiredAt != nil && !u.ExpiredAt.After(time.Now().UTC()) {
			_ = d.Store.LogSubAccess(c.Request.Context(), &t.UserID, hash, target, c.ClientIP(), c.Request.UserAgent(), "deny", "plan_expired")
			httpx.Fail(c, httpx.NewError(http.StatusForbidden, "plan_expired", "套餐已到期"))
			return
		}
		if u.TrafficLimit > 0 && u.TrafficUsed >= u.TrafficLimit {
			_ = d.Store.LogSubAccess(c.Request.Context(), &t.UserID, hash, target, c.ClientIP(), c.Request.UserAgent(), "deny", "traffic_exceeded")
			httpx.Fail(c, httpx.NewError(http.StatusForbidden, "traffic_exceeded", "流量已用尽"))
			return
		}
		nodeUsers, err := d.Store.ListNodeUsersByUser(c.Request.Context(), t.UserID)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		if err := checkSubscriptionDeviceLimit(c.Request.Context(), d.Store, t.UserID, nodeUsers, c.ClientIP(), c.Request.UserAgent()); err != nil {
			_ = d.Store.LogSubAccess(c.Request.Context(), &t.UserID, hash, target, c.ClientIP(), c.Request.UserAgent(), "deny", "device_limit_exceeded")
			httpx.Fail(c, err)
			return
		}
		nodes, err := d.Store.ListActiveNodes(c.Request.Context())
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		items := subrender.Build(nodes, nodeUsers)

		var body, contentType string
		switch target {
		case "clash", "clash-meta":
			body = subrender.ClashMeta(items)
			contentType = "text/yaml; charset=utf-8"
		case "sing-box", "singbox":
			body = subrender.SingBox(items)
			contentType = "application/json; charset=utf-8"
		default:
			body = subrender.Base64(items)
			contentType = "text/plain; charset=utf-8"
		}

		_ = d.Store.TouchSubTokenAccess(c.Request.Context(), t.ID, c.ClientIP(), c.Request.UserAgent())
		_ = d.Store.LogSubAccess(c.Request.Context(), &t.UserID, hash, target, c.ClientIP(), c.Request.UserAgent(), "allow", "")
		c.Header("Subscription-Userinfo", subUserInfo(c.Request.Context(), d.Store, u))
		c.Data(http.StatusOK, contentType, []byte(body))
	}
}

func subscriptionTargetEnabled(ctx context.Context, st *store.Store, target string) bool {
	key := ""
	switch target {
	case "clash", "clash-meta":
		key = "clash_enabled"
	case "sing-box", "singbox":
		key = "singbox_enabled"
	case "base64", "v2rayn", "":
		key = "v2rayn_enabled"
	default:
		return false
	}
	enabled, err := st.BoolSetting(ctx, key, true)
	return err == nil && enabled
}

func checkSubscriptionDeviceLimit(ctx context.Context, st *store.Store, userID int64, nodeUsers []store.NodeUser, ip, ua string) error {
	limit := subscriptionDeviceLimit(nodeUsers)
	if limit <= 0 {
		return st.TouchUserDevice(ctx, userID, subscriptionDeviceFingerprint(ip, ua), ip, ua)
	}
	fp := subscriptionDeviceFingerprint(ip, ua)
	exists, err := st.HasUserDevice(ctx, userID, fp)
	if err != nil {
		return err
	}
	if !exists {
		count, err := st.CountUserDevices(ctx, userID)
		if err != nil {
			return err
		}
		if count >= limit {
			return httpx.NewError(http.StatusForbidden, "device_limit_exceeded", fmt.Sprintf("套餐最多允许 %d 台设备使用订阅", limit))
		}
	}
	return st.TouchUserDevice(ctx, userID, fp, ip, ua)
}

func subscriptionDeviceLimit(nodeUsers []store.NodeUser) int {
	limit := 0
	for _, nu := range nodeUsers {
		if nu.Enabled == 0 || nu.DeviceLimit <= 0 {
			continue
		}
		if limit == 0 || nu.DeviceLimit < limit {
			limit = nu.DeviceLimit
		}
	}
	return limit
}

func subscriptionDeviceFingerprint(ip, ua string) string {
	sum := sha256.Sum256([]byte(ip + "\n" + ua))
	return hex.EncodeToString(sum[:])
}

func newSubToken() (string, error) { return authx.NewToken(24) }

func hashSubToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

func subUserInfo(ctx context.Context, st *store.Store, u *store.User) string {
	expire := int64(0)
	if u.ExpiredAt != nil {
		expire = u.ExpiredAt.Unix()
	}
	upload := int64(0)
	download := u.TrafficUsed
	if snap, err := st.FindTrafficSnapshotByUser(ctx, u.ID); err == nil && snap != nil {
		upload = snap.UploadTotal
		download = snap.DownloadTotal
	}
	total := u.TrafficLimit
	return "upload=" + i64(upload) + "; download=" + i64(download) + "; total=" + i64(total) + "; expire=" + i64(expire)
}

func i64(n int64) string {
	return formatInt(n)
}

// formatInt avoids importing strconv in handler signature lines.
func formatInt(n int64) string {
	const digits = "0123456789"
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = digits[n%10]
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
