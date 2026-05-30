// Package worker contains the maintenance pipeline that disables expired and
// over-quota users, sends subscription reminders, generates disable_user tasks
// for the affected nodes, and sweeps stale tasks.
package worker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/zboard/api-server/internal/agentauth"
	"github.com/zboard/api-server/internal/store"
)

const (
	// TaskRunTimeout is how long a `running` task may stay locked before the
	// sweep moves it to `failed`.
	TaskRunTimeout = 10 * time.Minute

	// AgentNonceRetention is how far back we keep nonces. The HMAC middleware
	// already rejects timestamps outside ±agentauth.WindowSeconds, so any nonce
	// older than that window cannot be replayed and is safe to delete. We keep
	// a 2× window of slack to absorb clock skew between the API and agents.
	AgentNonceRetention = time.Duration(2*agentauth.WindowSeconds) * time.Second
)

type Service struct {
	Store *store.Store
}

func New(s *store.Store) *Service { return &Service{Store: s} }

// Result reports counts for a single maintenance run.
type Result struct {
	ExpiredUsers     int   `json:"expired_users"`
	OverQuotaUsers   int   `json:"over_quota_users"`
	DisabledUsers    int   `json:"disabled_users"`
	DisableTasks     int   `json:"disable_tasks"`
	ExpiryReminders  int   `json:"expiry_reminders"`
	TrafficReminders int   `json:"traffic_reminders"`
	TimedOutTasks    int64 `json:"timed_out_tasks"`
	PurgedNonces     bool  `json:"purged_nonces"`
}

// Run executes a full maintenance pass.
func (s *Service) Run(ctx context.Context) (*Result, error) {
	now := time.Now().UTC()
	res := &Result{}

	expired, err := s.Store.FindExpiredActiveUserIDs(ctx, now)
	if err != nil {
		return nil, err
	}
	for _, uid := range expired {
		n, err := s.disableUser(ctx, uid, "expired")
		if err != nil {
			return nil, err
		}
		res.DisableTasks += n
	}
	res.ExpiredUsers = len(expired)

	overs, err := s.Store.FindOverQuotaUserIDs(ctx)
	if err != nil {
		return nil, err
	}
	for _, uid := range overs {
		n, err := s.disableUser(ctx, uid, "traffic_exceeded")
		if err != nil {
			return nil, err
		}
		res.DisableTasks += n
	}
	res.OverQuotaUsers = len(overs)
	res.DisabledUsers = len(expired) + len(overs)

	expiryReminders, err := s.sendExpiryReminders(ctx, now)
	if err != nil {
		return nil, err
	}
	res.ExpiryReminders = expiryReminders
	trafficReminders, err := s.sendTrafficReminders(ctx)
	if err != nil {
		return nil, err
	}
	res.TrafficReminders = trafficReminders

	swept, err := s.Store.SweepTimeoutTasks(ctx, now.Add(-TaskRunTimeout))
	if err != nil {
		return nil, err
	}
	res.TimedOutTasks = swept

	// Drop replay-window nonces. The middleware rejects anything older than
	// the timestamp window, so these can never be presented again.
	if err := s.Store.PurgeAgentNonces(ctx, now.Add(-AgentNonceRetention).Unix()); err != nil {
		return nil, err
	}
	res.PurgedNonces = true

	return res, nil
}

func (s *Service) sendExpiryReminders(ctx context.Context, now time.Time) (int, error) {
	enabled, err := s.Store.BoolSetting(ctx, "subscription_expire_reminder_enabled", true)
	if err != nil || !enabled {
		return 0, err
	}
	days, err := s.Store.IntSetting(ctx, "reminder_days_before", 3)
	if err != nil {
		return 0, err
	}
	if days <= 0 {
		days = 3
	}
	userIDs, err := s.Store.FindExpiringActiveUserIDs(ctx, now, now.AddDate(0, 0, days))
	if err != nil {
		return 0, err
	}
	count := 0
	for _, userID := range userIDs {
		exists, err := s.Store.HasUnreadNotificationType(ctx, userID, "plan_expiring")
		if err != nil {
			return count, err
		}
		if exists {
			continue
		}
		if err := s.Store.CreateNotification(ctx, userID, "plan_expiring",
			"订阅即将到期", fmt.Sprintf("您的订阅将在 %d 天内到期，请及时续费。", days), "/dashboard"); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (s *Service) sendTrafficReminders(ctx context.Context) (int, error) {
	enabled, err := s.Store.BoolSetting(ctx, "traffic_alert_enabled", true)
	if err != nil || !enabled {
		return 0, err
	}
	threshold, err := s.Store.IntSetting(ctx, "traffic_alert_threshold", 80)
	if err != nil {
		return 0, err
	}
	userIDs, err := s.Store.FindTrafficAlertUserIDs(ctx, threshold)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, userID := range userIDs {
		exists, err := s.Store.HasUnreadNotificationType(ctx, userID, "traffic_alert")
		if err != nil {
			return count, err
		}
		if exists {
			continue
		}
		if err := s.Store.CreateNotification(ctx, userID, "traffic_alert",
			"流量即将用尽", fmt.Sprintf("您的订阅流量已使用超过 %d%%，请留意剩余流量。", threshold), "/dashboard"); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// disableUser flips status, disables node_users, and emits one disable_user
// task per node the user has access to. Returns the number of tasks created.
func (s *Service) disableUser(ctx context.Context, userID int64, reason string) (int, error) {
	if err := s.Store.SetUserStatus(ctx, userID, "disabled"); err != nil {
		return 0, err
	}
	if err := s.Store.DeleteUserSessions(ctx, userID); err != nil {
		return 0, err
	}
	if err := s.Store.SetNodeUserEnabledForUser(ctx, userID, 0); err != nil {
		return 0, err
	}
	nodeIDs, err := s.Store.FindUserNodeIDs(ctx, userID)
	if err != nil {
		return 0, err
	}
	for _, nid := range nodeIDs {
		taskID := "task-disable-" + reason + "-" + randHex(6)
		payload := fmt.Sprintf(`{"user_id":%d,"reason":%q}`, userID, reason)
		if err := s.Store.CreateNodeTask(ctx, taskID, nid, "disable_user", payload); err != nil {
			return 0, err
		}
	}
	switch reason {
	case "expired":
		s.Store.NotifyUser(ctx, userID, "plan_expired",
			"套餐已到期", "您的套餐已到期，服务已暂停。请续费以恢复使用。",
			"/dashboard")
	case "traffic_exceeded":
		s.Store.NotifyUser(ctx, userID, "traffic_exceeded",
			"流量已用尽", "您的流量已超出限额，服务已暂停。请购买新套餐或联系管理员。",
			"/dashboard")
	}
	return len(nodeIDs), nil
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
