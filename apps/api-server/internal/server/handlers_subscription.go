package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zboard/api-server/internal/authx"
	"github.com/zboard/api-server/internal/httpx"
	"github.com/zboard/api-server/internal/store"
	"github.com/zboard/api-server/internal/subrender"
	"github.com/zboard/api-server/internal/subtoken"
)

const subscriptionDeviceOnlineWindow = 10 * time.Minute
const storedSubTokenPrefix = subtoken.StoredPrefix

func subToken(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.MustGet(ctxUserIDKey).(int64)
		t, err := d.Store.FindActiveSubTokenByUser(c.Request.Context(), uid)
		if err != nil && !store.IsNoRows(err) {
			httpx.Fail(c, err)
			return
		}
		if t == nil {
			tok, storedToken, err := newSubTokenForUser(uid, d)
			if err != nil {
				httpx.Fail(c, err)
				return
			}
			if _, err := d.Store.CreateSubToken(c.Request.Context(), uid, storedToken, hashSubToken(tok)); err != nil {
				httpx.Fail(c, err)
				return
			}
			httpx.OK(c, gin.H{"token": tok})
			return
		}
		tok, err := materializeSubToken(uid, t.Token, d)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"token": tok})
	}
}

func subResetToken(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.MustGet(ctxUserIDKey).(int64)
		tok, storedToken, err := newSubTokenForUser(uid, d)
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		if err := d.Store.RotateSubToken(c.Request.Context(), uid, storedToken, hashSubToken(tok)); err != nil {
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
		target, ok := resolveSubscriptionTarget(c.Query("target"), c.Request.UserAgent())
		if !ok {
			_ = d.Store.LogSubAccess(c.Request.Context(), nil, hash, c.Query("target"), c.ClientIP(), c.Request.UserAgent(), "deny", "target_disabled")
			httpx.Fail(c, httpx.NewError(http.StatusForbidden, "subscription_target_disabled", "该订阅格式已关闭"))
			return
		}
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
		nodes, err := d.Store.ListActiveNodes(c.Request.Context())
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		if healed, err := ensureSubscriptionNodeUsers(c.Request.Context(), d.Store, u, nodes, nodeUsers); err != nil {
			httpx.Fail(c, err)
			return
		} else if healed {
			nodeUsers, err = d.Store.ListNodeUsersByUser(c.Request.Context(), t.UserID)
			if err != nil {
				httpx.Fail(c, err)
				return
			}
			if d.Nodes != nil {
				for _, n := range nodes {
					_, _, _ = d.Nodes.GenerateSyncTask(c.Request.Context(), n.ID)
				}
			}
		}
		if err := trackSubscriptionDevice(c.Request.Context(), d.Store, t.UserID, nodeUsers, c.ClientIP(), c.Request.UserAgent()); err != nil {
			_ = d.Store.LogSubAccess(c.Request.Context(), &t.UserID, hash, target, c.ClientIP(), c.Request.UserAgent(), "deny", "device_tracking_failed")
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
	case "clash":
		key = "clash_enabled"
	case "sing-box":
		key = "singbox_enabled"
	case "base64":
		key = "v2rayn_enabled"
	default:
		return false
	}
	enabled, err := st.BoolSetting(ctx, key, true)
	return err == nil && enabled
}

func resolveSubscriptionTarget(rawTarget, userAgent string) (string, bool) {
	target := strings.ToLower(strings.TrimSpace(rawTarget))
	if target == "" || target == "auto" {
		return inferSubscriptionTarget(userAgent), true
	}
	return canonicalSubscriptionTarget(target)
}

func inferSubscriptionTarget(userAgent string) string {
	ua := strings.ToLower(userAgent)
	switch {
	case strings.Contains(ua, "clash") || strings.Contains(ua, "mihomo") || strings.Contains(ua, "stash"):
		return "clash"
	case strings.Contains(ua, "sing-box") || strings.Contains(ua, "singbox") ||
		strings.Contains(ua, "hiddify") || strings.Contains(ua, "furious"):
		return "sing-box"
	default:
		return "base64"
	}
}

func canonicalSubscriptionTarget(target string) (string, bool) {
	key := strings.NewReplacer("_", "-", " ", "-", ".", "-").Replace(target)
	switch key {
	case "base64", "v2ray", "v2rayn", "v2rayng", "shadowrocket", "passwall", "general", "uri", "uris":
		return "base64", true
	case "clash", "clash-meta", "clash-verge", "mihomo", "stash":
		return "clash", true
	case "sing-box", "singbox", "hiddify", "hiddify-next", "furious":
		return "sing-box", true
	default:
		return "", false
	}
}

func ensureSubscriptionNodeUsers(ctx context.Context, st *store.Store, u *store.User, nodes []store.Node, nodeUsers []store.NodeUser) (bool, error) {
	if u == nil || u.Status != "active" {
		return false, nil
	}
	now := time.Now().UTC()
	if u.ExpiredAt != nil && !u.ExpiredAt.After(now) {
		return false, nil
	}
	if u.TrafficLimit > 0 && u.TrafficUsed >= u.TrafficLimit {
		return false, nil
	}
	byNode := make(map[int64]store.NodeUser, len(nodeUsers))
	clientID := ""
	for _, nu := range nodeUsers {
		byNode[nu.NodeID] = nu
		if clientID == "" && nu.ClientID != "" {
			clientID = nu.ClientID
		}
	}
	if clientID == "" {
		var err error
		clientID, err = newClientIDForServer()
		if err != nil {
			return false, err
		}
	}
	deviceLimit := 0
	if u.PlanID != nil {
		if plan, err := st.FindPlanByID(ctx, *u.PlanID); err == nil {
			deviceLimit = plan.DeviceLimit
		} else if !store.IsNoRows(err) {
			return false, err
		}
	}
	changed := false
	for _, n := range nodes {
		if _, ok := byNode[n.ID]; ok {
			continue
		}
		if err := st.EnsureNodeUserWithLimits(ctx, u.ID, n.ID, clientID, n.Protocol, 0, deviceLimit); err != nil {
			return false, err
		}
		changed = true
	}
	return changed, nil
}

func trackSubscriptionDevice(ctx context.Context, st *store.Store, userID int64, nodeUsers []store.NodeUser, ip, ua string) error {
	limit := subscriptionDeviceLimit(nodeUsers)
	if limit <= 0 {
		return st.TouchUserDevice(ctx, userID, subscriptionDeviceFingerprint(ip, ua), ip, ua)
	}
	fp := subscriptionDeviceFingerprint(ip, ua)
	activeSince := time.Now().UTC().Add(-subscriptionDeviceOnlineWindow)
	exists, err := st.HasActiveUserDevice(ctx, userID, fp, activeSince)
	if err != nil {
		return err
	}
	if !exists {
		count, err := st.CountActiveUserDevices(ctx, userID, activeSince)
		if err != nil {
			return err
		}
		if count >= limit {
			return nil
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

func newSubTokenForUser(userID int64, d Deps) (publicToken, storedToken string, err error) {
	salt, err := authx.NewToken(24)
	if err != nil {
		return "", "", err
	}
	storedToken = storedSubTokenPrefix + salt
	publicToken, err = subtoken.PublicToken(userID, salt, d.TokenSecret)
	if err != nil {
		return "", "", err
	}
	return publicToken, storedToken, nil
}

func materializeSubToken(userID int64, storedToken string, d Deps) (string, error) {
	return subtoken.Materialize(userID, storedToken, d.TokenSecret)
}

func hashSubToken(t string) string {
	return subtoken.Hash(t)
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
