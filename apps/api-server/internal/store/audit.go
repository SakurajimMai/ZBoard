package store

import (
	"context"
	"unicode/utf8"
)

// AuditDetailMaxBytes caps the audit `detail` column. The schema column is
// VARCHAR/TEXT depending on dialect; we truncate at the application layer so a
// 100KB error string from a payment provider, a stack trace, or a malicious
// callback can't bloat the table or push the row over MySQL's row-size limit.
const AuditDetailMaxBytes = 4096

const auditTruncMarker = "…[truncated]"

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
	e.Detail = truncateAuditDetail(e.Detail)
	q := s.Rebind(`INSERT INTO audit_logs(actor_type, actor_id, action, resource_type,
		resource_id, ip, user_agent, detail) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	_, err := s.DB.ExecContext(ctx, q, e.ActorType, e.ActorID, e.Action,
		e.ResourceType, e.ResourceID, e.IP, e.UserAgent, e.Detail)
	return err
}

// truncateAuditDetail caps detail to AuditDetailMaxBytes without slicing through
// a multi-byte UTF-8 codepoint. A naive byte slice (`s[:n]`) can leave a partial
// rune at the cut, producing invalid UTF-8 that MySQL's utf8mb4 columns reject
// in strict mode — which would silently drop the entire audit row. We back the
// cut up to the nearest rune boundary, then append a marker.
func truncateAuditDetail(s string) string {
	if len(s) <= AuditDetailMaxBytes {
		return s
	}
	budget := AuditDetailMaxBytes - len(auditTruncMarker)
	if budget <= 0 {
		return auditTruncMarker
	}
	cut := budget
	// utf8.RuneStart reports whether s[cut] could begin an encoded rune. If it's
	// a continuation byte we're mid-codepoint, so step back until s[:cut] ends
	// on a complete rune.
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + auditTruncMarker
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

// ListAuditLogsPage returns one page of audit rows plus the total row count.
// Pagination keeps the admin UI responsive once the table grows past tens of
// thousands of rows; the unbounded `ListAuditLogs` would otherwise stream the
// full first page into memory on every request.
func (s *Store) ListAuditLogsPage(ctx context.Context, p PageParams) ([]AuditRow, int64, error) {
	p = NormalizePage(p)
	var total int64
	if err := s.DB.GetContext(ctx, &total, s.Rebind(`SELECT COUNT(*) FROM audit_logs`)); err != nil {
		return nil, 0, err
	}
	q := s.Rebind(`SELECT id, actor_type, actor_id, action, resource_type, resource_id,
		ip, user_agent, detail, created_at FROM audit_logs
		ORDER BY id DESC LIMIT ? OFFSET ?`)
	var rows []AuditRow
	if err := s.DB.SelectContext(ctx, &rows, q, p.PageSize, p.Offset()); err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
