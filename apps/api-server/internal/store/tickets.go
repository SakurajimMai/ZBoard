package store

import (
	"context"
	"time"
)

type Ticket struct {
	ID        int64      `db:"id" json:"id"`
	TicketNo  string     `db:"ticket_no" json:"ticket_no"`
	UserID    int64      `db:"user_id" json:"user_id"`
	Subject   string     `db:"subject" json:"subject"`
	Category  string     `db:"category" json:"category"`
	Status    string     `db:"status" json:"status"`
	Priority  string     `db:"priority" json:"priority"`
	ClosedAt  *time.Time `db:"closed_at" json:"closed_at"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
}

type TicketMessage struct {
	ID         int64     `db:"id" json:"id"`
	TicketID   int64     `db:"ticket_id" json:"ticket_id"`
	SenderType string    `db:"sender_type" json:"sender_type"` // "user" | "admin"
	SenderID   int64     `db:"sender_id" json:"sender_id"`
	Content    string    `db:"content" json:"content"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

func (s *Store) CreateTicket(ctx context.Context, ticketNo string, userID int64, subject, category, content string) (int64, error) {
	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var ticketID int64
	insertTicket := `INSERT INTO tickets(ticket_no, user_id, subject, category) VALUES (?, ?, ?, ?)`
	if s.Dialect == "postgres" {
		q := s.Rebind(insertTicket + " RETURNING id")
		if err := tx.QueryRowxContext(ctx, q, ticketNo, userID, subject, category).Scan(&ticketID); err != nil {
			return 0, err
		}
	} else {
		res, err := tx.ExecContext(ctx, s.Rebind(insertTicket), ticketNo, userID, subject, category)
		if err != nil {
			return 0, err
		}
		ticketID, _ = res.LastInsertId()
	}

	insertMsg := s.Rebind(`INSERT INTO ticket_messages(ticket_id, sender_type, sender_id, content) VALUES (?, ?, ?, ?)`)
	if _, err := tx.ExecContext(ctx, insertMsg, ticketID, "user", userID, content); err != nil {
		return 0, err
	}

	return ticketID, tx.Commit()
}

func (s *Store) ListTicketsByUser(ctx context.Context, userID int64) ([]Ticket, error) {
	q := s.Rebind(`SELECT id, ticket_no, user_id, subject, category, status, priority, closed_at, created_at, updated_at
		FROM tickets WHERE user_id = ? ORDER BY id DESC`)
	var rows []Ticket
	if err := s.DB.SelectContext(ctx, &rows, q, userID); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) ListAllTickets(ctx context.Context, status string) ([]Ticket, error) {
	var q string
	var args []any
	if status != "" && status != "all" {
		q = s.Rebind(`SELECT id, ticket_no, user_id, subject, category, status, priority, closed_at, created_at, updated_at
			FROM tickets WHERE status = ? ORDER BY id DESC`)
		args = []any{status}
	} else {
		q = `SELECT id, ticket_no, user_id, subject, category, status, priority, closed_at, created_at, updated_at
			FROM tickets ORDER BY id DESC`
	}
	var rows []Ticket
	if err := s.DB.SelectContext(ctx, &rows, q, args...); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) FindTicketByNo(ctx context.Context, ticketNo string) (*Ticket, error) {
	q := s.Rebind(`SELECT id, ticket_no, user_id, subject, category, status, priority, closed_at, created_at, updated_at
		FROM tickets WHERE ticket_no = ?`)
	var t Ticket
	if err := s.DB.GetContext(ctx, &t, q, ticketNo); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) FindTicketByID(ctx context.Context, id int64) (*Ticket, error) {
	q := s.Rebind(`SELECT id, ticket_no, user_id, subject, category, status, priority, closed_at, created_at, updated_at
		FROM tickets WHERE id = ?`)
	var t Ticket
	if err := s.DB.GetContext(ctx, &t, q, id); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) ListTicketMessages(ctx context.Context, ticketID int64) ([]TicketMessage, error) {
	q := s.Rebind(`SELECT id, ticket_id, sender_type, sender_id, content, created_at
		FROM ticket_messages WHERE ticket_id = ? ORDER BY id ASC`)
	var rows []TicketMessage
	if err := s.DB.SelectContext(ctx, &rows, q, ticketID); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) AddTicketMessage(ctx context.Context, ticketID int64, senderType string, senderID int64, content string) error {
	q := s.Rebind(`INSERT INTO ticket_messages(ticket_id, sender_type, sender_id, content) VALUES (?, ?, ?, ?)`)
	_, err := s.DB.ExecContext(ctx, q, ticketID, senderType, senderID, content)
	return err
}

func (s *Store) UpdateTicketStatus(ctx context.Context, ticketID int64, status string) error {
	q := s.Rebind(`UPDATE tickets SET status = ? WHERE id = ?`)
	_, err := s.DB.ExecContext(ctx, q, status, ticketID)
	return err
}

func (s *Store) CloseTicket(ctx context.Context, ticketID int64) error {
	q := s.Rebind(`UPDATE tickets SET status = 'closed', closed_at = ? WHERE id = ?`)
	_, err := s.DB.ExecContext(ctx, q, Now(), ticketID)
	return err
}

func (s *Store) CountTicketMessages(ctx context.Context, ticketID int64) (int, error) {
	q := s.Rebind(`SELECT COUNT(*) FROM ticket_messages WHERE ticket_id = ?`)
	var n int
	if err := s.DB.GetContext(ctx, &n, q, ticketID); err != nil {
		return 0, err
	}
	return n, nil
}
