package store

import (
	"context"
	"time"
)

type NodeUser struct {
	ID          int64      `db:"id" json:"id"`
	UserID      int64      `db:"user_id" json:"user_id"`
	NodeID      int64      `db:"node_id" json:"node_id"`
	ClientID    string     `db:"client_id" json:"client_id"`
	Protocol    string     `db:"protocol" json:"protocol"`
	Enabled     int        `db:"enabled" json:"enabled"`
	Upload      int64      `db:"upload" json:"upload"`
	Download    int64      `db:"download" json:"download"`
	SpeedLimit  int        `db:"speed_limit" json:"speed_limit"`
	DeviceLimit int        `db:"device_limit" json:"device_limit"`
	LastSyncAt  *time.Time `db:"last_sync_at" json:"last_sync_at"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

// EnsureNodeUser inserts a node_users row if missing. Idempotent on (user_id, node_id).
func (s *Store) EnsureNodeUser(ctx context.Context, userID, nodeID int64, clientID, protocol string) error {
	return s.EnsureNodeUserWithLimits(ctx, userID, nodeID, clientID, protocol, 0, 0)
}

func (s *Store) EnsureNodeUserWithLimits(ctx context.Context, userID, nodeID int64, clientID, protocol string, speedLimit, deviceLimit int) error {
	speedLimit = 0
	_, err := s.DB.ExecContext(ctx,
		s.Rebind(`INSERT INTO node_users(user_id, node_id, client_id, protocol, enabled, speed_limit, device_limit)
			VALUES (?, ?, ?, ?, 1, ?, ?)`),
		userID, nodeID, clientID, protocol, speedLimit, deviceLimit,
	)
	if err != nil && IsUniqueViolation(err) {
		q := s.Rebind(`UPDATE node_users SET protocol = ?, enabled = 1,
			speed_limit = ?, device_limit = ?, updated_at = CURRENT_TIMESTAMP
			WHERE user_id = ? AND node_id = ?`)
		_, err = s.DB.ExecContext(ctx, q, protocol, speedLimit, deviceLimit, userID, nodeID)
	}
	return err
}

func (s *Store) ApplyPlanLimitsToNodeUsers(ctx context.Context, userID int64, plan *Plan) error {
	if plan == nil {
		return nil
	}
	q := s.Rebind(`UPDATE node_users SET speed_limit = ?, device_limit = ?,
		updated_at = CURRENT_TIMESTAMP WHERE user_id = ?`)
	_, err := s.DB.ExecContext(ctx, q, 0, plan.DeviceLimit, userID)
	return err
}

func (s *Store) FindNodeUser(ctx context.Context, userID, nodeID int64) (*NodeUser, error) {
	q := s.Rebind(`SELECT id, user_id, node_id, client_id, protocol, enabled, upload, download,
		speed_limit, device_limit, last_sync_at, created_at, updated_at
		FROM node_users WHERE user_id = ? AND node_id = ?`)
	var nu NodeUser
	if err := s.DB.GetContext(ctx, &nu, q, userID, nodeID); err != nil {
		return nil, err
	}
	return &nu, nil
}

func (s *Store) ListNodeUsersByUser(ctx context.Context, userID int64) ([]NodeUser, error) {
	q := s.Rebind(`SELECT id, user_id, node_id, client_id, protocol, enabled, upload, download,
		speed_limit, device_limit, last_sync_at, created_at, updated_at
		FROM node_users WHERE user_id = ? ORDER BY node_id ASC`)
	var rows []NodeUser
	if err := s.DB.SelectContext(ctx, &rows, q, userID); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) ListNodeUsersByNode(ctx context.Context, nodeID int64) ([]NodeUser, error) {
	q := s.Rebind(`SELECT id, user_id, node_id, client_id, protocol, enabled, upload, download,
		speed_limit, device_limit, last_sync_at, created_at, updated_at
		FROM node_users WHERE node_id = ? ORDER BY user_id ASC`)
	var rows []NodeUser
	if err := s.DB.SelectContext(ctx, &rows, q, nodeID); err != nil {
		return nil, err
	}
	return rows, nil
}

// ListEnabledNodeUserIDsByNode returns the set of user IDs that are *currently
// enabled* on the node. The traffic-report path uses this as an allowlist:
// a disabled account (closed by the worker for expiry/over-quota) must never be
// chargeable again, otherwise a malicious or compromised agent could keep
// attributing traffic to it — wedging the audit trail and re-tripping quotas.
func (s *Store) ListEnabledNodeUserIDsByNode(ctx context.Context, nodeID int64) (map[int64]struct{}, error) {
	q := s.Rebind(`SELECT user_id FROM node_users WHERE node_id = ? AND enabled = 1`)
	var ids []int64
	if err := s.DB.SelectContext(ctx, &ids, q, nodeID); err != nil {
		return nil, err
	}
	out := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		out[id] = struct{}{}
	}
	return out, nil
}

// SetNodeUserEnabled toggles the enabled flag on all node_users rows for a user.
func (s *Store) SetNodeUserEnabledForUser(ctx context.Context, userID int64, enabled int) error {
	q := s.Rebind(`UPDATE node_users SET enabled = ? WHERE user_id = ?`)
	_, err := s.DB.ExecContext(ctx, q, enabled, userID)
	return err
}
