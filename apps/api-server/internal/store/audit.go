package store

import "context"

type AuditEntry struct {
	ActorType    string
	ActorID      *int64
	Action       string
	ResourceType string
	ResourceID   string
	IP           string
	UserAgent    string
	Detail       string
}

func (s *Store) WriteAudit(ctx context.Context, e AuditEntry) error {
	q := s.Rebind(`INSERT INTO audit_logs(actor_type, actor_id, action, resource_type,
		resource_id, ip, user_agent, detail) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	_, err := s.DB.ExecContext(ctx, q, e.ActorType, e.ActorID, e.Action,
		e.ResourceType, e.ResourceID, e.IP, e.UserAgent, e.Detail)
	return err
}

type AuditRow struct {
	ID           int64   `db:"id" json:"id"`
	ActorType    string  `db:"actor_type" json:"actor_type"`
	ActorID      *int64  `db:"actor_id" json:"actor_id"`
	Action       string  `db:"action" json:"action"`
	ResourceType *string `db:"resource_type" json:"resource_type"`
	ResourceID   *string `db:"resource_id" json:"resource_id"`
	IP           *string `db:"ip" json:"ip"`
	UserAgent    *string `db:"user_agent" json:"user_agent"`
	Detail       *string `db:"detail" json:"detail"`
	CreatedAt    string  `db:"created_at" json:"created_at"`
}

func (s *Store) ListAuditLogs(ctx context.Context, limit int) ([]AuditRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := s.Rebind(`SELECT id, actor_type, actor_id, action, resource_type, resource_id,
		ip, user_agent, detail, created_at FROM audit_logs ORDER BY id DESC LIMIT ?`)
	var rows []AuditRow
	if err := s.DB.SelectContext(ctx, &rows, q, limit); err != nil {
		return nil, err
	}
	return rows, nil
}
