package store

import "context"

// DeductBalanceAtomic atomically subtracts `amount` (decimal string, e.g. "1.50")
// from users.balance only when the current balance is sufficient. Returns true
// when the deduction happened, false when balance was insufficient.
func (s *Store) DeductBalanceAtomic(ctx context.Context, userID int64, amount string) (bool, error) {
	q := s.Rebind(`UPDATE users SET balance = balance - ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND balance >= ?`)
	res, err := s.DB.ExecContext(ctx, q, amount, userID, amount)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
