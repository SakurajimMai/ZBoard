// Package worker contains the maintenance pipeline that disables expired and
// over-quota users, generates disable_user tasks for the affected nodes, and
// sweeps stale tasks. It runs in-process and is triggered manually today; cron
// integration is a follow-up.
package worker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/zboard/api-server/internal/store"
)

const (
	// TaskRunTimeout is how long a `running` task may stay locked before the
	// sweep moves it to `failed`.
	TaskRunTimeout = 10 * time.Minute
)

type Service struct {
	Store *store.Store
}

func New(s *store.Store) *Service { return &Service{Store: s} }

// Result reports counts for a single maintenance run.
type Result struct {
	ExpiredUsers   int   `json:"expired_users"`
	OverQuotaUsers int   `json:"over_quota_users"`
	DisabledUsers  int   `json:"disabled_users"`
	DisableTasks   int   `json:"disable_tasks"`
	TimedOutTasks  int64 `json:"timed_out_tasks"`
}

// Run executes a full maintenance pass.
func (s *Service) Run(ctx context.Context) (*Result, error) {
	now := time.Now().UTC()
	res := &Result{}

	// 1. Expired users
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

	// 2. Over-quota users
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

	// 3. Sweep stale running tasks
	swept, err := s.Store.SweepTimeoutTasks(ctx, now.Add(-TaskRunTimeout))
	if err != nil {
		return nil, err
	}
	res.TimedOutTasks = swept

	return res, nil
}

// disableUser flips status, disables node_users, and emits one disable_user
// task per node the user has access to. Returns the number of tasks created.
func (s *Service) disableUser(ctx context.Context, userID int64, reason string) (int, error) {
	if err := s.Store.SetUserStatus(ctx, userID, "disabled"); err != nil {
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
	// Notify user about account suspension
	switch reason {
	case "expired":
		s.Store.NotifyUser(ctx, userID, "plan_expired",
			"套餐已到期", "您的套餐已到期，服务已暂停。请续费以恢复使用。",
			"/dashboard/billing")
	case "traffic_exceeded":
		s.Store.NotifyUser(ctx, userID, "plan_expired",
			"流量已用尽", "您的流量已超出限额，服务已暂停。请购买新套餐或流量包。",
			"/dashboard/billing")
	}
	return len(nodeIDs), nil
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
