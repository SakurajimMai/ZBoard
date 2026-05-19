package store

import (
	"context"
	"time"
)

type NodeTask struct {
	ID            int64      `db:"id" json:"id"`
	TaskID        string     `db:"task_id" json:"task_id"`
	NodeID        int64      `db:"node_id" json:"node_id"`
	TaskType      string     `db:"task_type" json:"task_type"`
	Payload       string     `db:"payload" json:"payload"`
	Status        string     `db:"status" json:"status"`
	RetryCount    int        `db:"retry_count" json:"retry_count"`
	MaxRetryCount int        `db:"max_retry_count" json:"max_retry_count"`
	LockedAt      *time.Time `db:"locked_at" json:"locked_at"`
	ExecutedAt    *time.Time `db:"executed_at" json:"executed_at"`
	FailedReason  *string    `db:"failed_reason" json:"failed_reason"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at" json:"updated_at"`
}

func (s *Store) CreateNodeTask(ctx context.Context, taskID string, nodeID int64, taskType, payload string) error {
	_, err := s.DB.ExecContext(ctx, s.Rebind(
		`INSERT INTO node_tasks(task_id, node_id, task_type, payload) VALUES (?, ?, ?, ?)`),
		taskID, nodeID, taskType, payload)
	return err
}

// PullTasksForNode locks up to `limit` pending tasks by setting status=running.
func (s *Store) PullTasksForNode(ctx context.Context, nodeID int64, limit int) ([]NodeTask, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var rows []NodeTask
	q := s.Rebind(`SELECT id, task_id, node_id, task_type, payload, status, retry_count,
		max_retry_count, locked_at, executed_at, failed_reason, created_at, updated_at
		FROM node_tasks WHERE node_id = ? AND status = 'pending' ORDER BY id ASC LIMIT ?`)
	if err := tx.SelectContext(ctx, &rows, q, nodeID, limit); err != nil {
		return nil, err
	}
	now := Now()
	for i := range rows {
		if _, err := tx.ExecContext(ctx, s.Rebind(
			`UPDATE node_tasks SET status = 'running', locked_at = ? WHERE id = ?`),
			now, rows[i].ID); err != nil {
			return nil, err
		}
		rows[i].Status = "running"
		rows[i].LockedAt = &now
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) FindNodeTaskByTaskID(ctx context.Context, taskID string) (*NodeTask, error) {
	q := s.Rebind(`SELECT id, task_id, node_id, task_type, payload, status, retry_count,
		max_retry_count, locked_at, executed_at, failed_reason, created_at, updated_at
		FROM node_tasks WHERE task_id = ?`)
	var t NodeTask
	if err := s.DB.GetContext(ctx, &t, q, taskID); err != nil {
		return nil, err
	}
	return &t, nil
}

// CompleteTask marks success or applies retry logic on failure.
func (s *Store) CompleteTask(ctx context.Context, taskID, status, failedReason string) error {
	t, err := s.FindNodeTaskByTaskID(ctx, taskID)
	if err != nil {
		return err
	}
	now := Now()
	if status == "success" {
		_, err := s.DB.ExecContext(ctx, s.Rebind(
			`UPDATE node_tasks SET status = 'success', executed_at = ? WHERE task_id = ?`),
			now, taskID)
		_ = s.recordTaskLog(ctx, t.NodeID, taskID, "success", "", "")
		return err
	}
	// failed branch
	if t.RetryCount+1 < t.MaxRetryCount {
		_, err := s.DB.ExecContext(ctx, s.Rebind(
			`UPDATE node_tasks SET status = 'pending', retry_count = retry_count + 1,
				failed_reason = ?, locked_at = NULL WHERE task_id = ?`),
			failedReason, taskID)
		_ = s.recordTaskLog(ctx, t.NodeID, taskID, "retry", failedReason, "")
		return err
	}
	_, err = s.DB.ExecContext(ctx, s.Rebind(
		`UPDATE node_tasks SET status = 'failed', failed_reason = ?, executed_at = ? WHERE task_id = ?`),
		failedReason, now, taskID)
	_ = s.recordTaskLog(ctx, t.NodeID, taskID, "failed", failedReason, "")
	return err
}

func (s *Store) recordTaskLog(ctx context.Context, nodeID int64, taskID, status, message, detail string) error {
	q := s.Rebind(`INSERT INTO node_task_logs(task_id, node_id, status, message, detail, reported_at)
		VALUES (?, ?, ?, ?, ?, ?)`)
	_, err := s.DB.ExecContext(ctx, q, taskID, nodeID, status, message, detail, Now())
	return err
}

// SweepTimeoutTasks moves running tasks that exceeded `older` back to failed.
func (s *Store) SweepTimeoutTasks(ctx context.Context, olderThan time.Time) (int64, error) {
	res, err := s.DB.ExecContext(ctx, s.Rebind(
		`UPDATE node_tasks SET status = 'failed', failed_reason = '任务执行超时'
		 WHERE status = 'running' AND locked_at < ?`), olderThan)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) ListNodeTasks(ctx context.Context, limit int) ([]NodeTask, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := s.Rebind(`SELECT id, task_id, node_id, task_type, payload, status, retry_count,
		max_retry_count, locked_at, executed_at, failed_reason, created_at, updated_at
		FROM node_tasks ORDER BY id DESC LIMIT ?`)
	var rows []NodeTask
	if err := s.DB.SelectContext(ctx, &rows, q, limit); err != nil {
		return nil, err
	}
	return rows, nil
}
