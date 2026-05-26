package store

import (
	"context"
	"strings"
	"time"
)

type KnowledgeArticle struct {
	ID        int64     `db:"id" json:"id"`
	Slug      string    `db:"slug" json:"slug"`
	Title     string    `db:"title" json:"title"`
	Category  string    `db:"category" json:"category"`
	Summary   string    `db:"summary" json:"summary"`
	Content   string    `db:"content" json:"content"`
	Sort      int       `db:"sort_order" json:"sort"`
	Status    string    `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type KnowledgeInput struct {
	Slug     string
	Title    string
	Category string
	Summary  string
	Content  string
	Sort     int
	Status   string
}

func (s *Store) CreateKnowledgeArticle(ctx context.Context, in KnowledgeInput) (int64, error) {
	if strings.TrimSpace(in.Status) == "" {
		in.Status = "active"
	}
	return s.InsertReturningID(ctx,
		`INSERT INTO knowledge_articles(slug, title, category, summary, content, sort_order, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		in.Slug, in.Title, in.Category, in.Summary, in.Content, in.Sort, in.Status,
	)
}

func (s *Store) UpdateKnowledgeArticle(ctx context.Context, id int64, in KnowledgeInput) error {
	if strings.TrimSpace(in.Status) == "" {
		in.Status = "active"
	}
	q := s.Rebind(`UPDATE knowledge_articles SET title = ?, category = ?, summary = ?, content = ?,
		sort_order = ?, status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`)
	_, err := s.DB.ExecContext(ctx, q, in.Title, in.Category, in.Summary, in.Content, in.Sort, in.Status, id)
	return err
}

func (s *Store) DeleteKnowledgeArticle(ctx context.Context, id int64) error {
	q := s.Rebind(`DELETE FROM knowledge_articles WHERE id = ?`)
	_, err := s.DB.ExecContext(ctx, q, id)
	return err
}

func (s *Store) ListKnowledgeArticlesPage(ctx context.Context, p PageParams, category, status string) ([]KnowledgeArticle, int64, error) {
	p = NormalizePage(p)
	clauses := []string{"1 = 1"}
	args := []any{}
	if strings.TrimSpace(category) != "" {
		clauses = append(clauses, "category = ?")
		args = append(args, strings.TrimSpace(category))
	}
	if strings.TrimSpace(status) != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, strings.TrimSpace(status))
	}
	where := strings.Join(clauses, " AND ")
	countQ := s.Rebind(`SELECT COUNT(*) FROM knowledge_articles WHERE ` + where)
	var total int64
	if err := s.DB.GetContext(ctx, &total, countQ, args...); err != nil {
		return nil, 0, err
	}
	listArgs := append(append([]any{}, args...), p.PageSize, p.Offset())
	q := s.Rebind(`SELECT id, slug, title, category, summary, content, sort_order, status, created_at, updated_at
		FROM knowledge_articles WHERE ` + where + ` ORDER BY sort_order DESC, id DESC LIMIT ? OFFSET ?`)
	var rows []KnowledgeArticle
	if err := s.DB.SelectContext(ctx, &rows, q, listArgs...); err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (s *Store) ListActiveKnowledgeArticles(ctx context.Context, category string, limit int) ([]KnowledgeArticle, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	clauses := []string{"status = 'active'"}
	args := []any{}
	if strings.TrimSpace(category) != "" {
		clauses = append(clauses, "category = ?")
		args = append(args, strings.TrimSpace(category))
	}
	args = append(args, limit)
	q := s.Rebind(`SELECT id, slug, title, category, summary, content, sort_order, status, created_at, updated_at
		FROM knowledge_articles WHERE ` + strings.Join(clauses, " AND ") + ` ORDER BY sort_order DESC, id DESC LIMIT ?`)
	var rows []KnowledgeArticle
	if err := s.DB.SelectContext(ctx, &rows, q, args...); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) FindActiveKnowledgeArticleBySlug(ctx context.Context, slug string) (*KnowledgeArticle, error) {
	q := s.Rebind(`SELECT id, slug, title, category, summary, content, sort_order, status, created_at, updated_at
		FROM knowledge_articles WHERE slug = ? AND status = 'active'`)
	var article KnowledgeArticle
	if err := s.DB.GetContext(ctx, &article, q, slug); err != nil {
		return nil, err
	}
	return &article, nil
}
