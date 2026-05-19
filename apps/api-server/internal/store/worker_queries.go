package store

import (
	"context"
	"time"
)

// FindExpiredActiveUserIDs returns IDs of users still 'active' but past expiry.
func (s *Store) FindExpiredActiveUserIDs(ctx context.Context, now time.Time) ([]int64, error) {
	q := s.Rebind(`SELECT id FROM users WHERE status = 'active' AND expired_at IS NOT NULL AND expired_at <= ?`)
	var ids []int64
	if err := s.DB.SelectContext(ctx, &ids, q, now); err != nil {
		return nil, err
	}
	return ids, nil
}

// FindOverQuotaUserIDs returns IDs of users whose traffic_used exceeds traffic_limit (limit > 0).
func (s *Store) FindOverQuotaUserIDs(ctx context.Context) ([]int64, error) {
	q := `SELECT id FROM users WHERE status = 'active' AND traffic_limit > 0 AND traffic_used >= traffic_limit`
	var ids []int64
	if err := s.DB.SelectContext(ctx, &ids, q); err != nil {
		return nil, err
	}
	return ids, nil
}

// FindUserNodeIDs returns the node IDs that currently have a node_users row for this user.
func (s *Store) FindUserNodeIDs(ctx context.Context, userID int64) ([]int64, error) {
	q := s.Rebind(`SELECT node_id FROM node_users WHERE user_id = ?`)
	var ids []int64
	if err := s.DB.SelectContext(ctx, &ids, q, userID); err != nil {
		return nil, err
	}
	return ids, nil
}

// FindStaleRunningTaskIDs returns task IDs that have been 'running' since before t.
// Used by the timeout sweep.
func (s *Store) FindStaleRunningTaskIDs(ctx context.Context, before time.Time) ([]int64, error) {
	q := s.Rebind(`SELECT id FROM node_tasks WHERE status = 'running' AND locked_at IS NOT NULL AND locked_at < ?`)
	var ids []int64
	if err := s.DB.SelectContext(ctx, &ids, q, before); err != nil {
		return nil, err
	}
	return ids, nil
}
