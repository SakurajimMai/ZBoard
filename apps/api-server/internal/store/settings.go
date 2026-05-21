package store

import (
	"context"
	"strconv"
	"strings"
)

type AppSetting struct {
	Key   string `db:"key" json:"key"`
	Value string `db:"value" json:"value"`
}

func (s *Store) ListSettings(ctx context.Context) (map[string]string, error) {
	q := `SELECT key, value FROM app_settings ORDER BY key ASC`
	if s.Dialect == "mysql" {
		q = "SELECT `key`, `value` FROM app_settings ORDER BY `key` ASC"
	}
	var rows []AppSetting
	if err := s.DB.SelectContext(ctx, &rows, q); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, row := range rows {
		out[row.Key] = row.Value
	}
	return out, nil
}

func (s *Store) GetSetting(ctx context.Context, key, fallback string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return fallback, nil
	}
	q := s.Rebind(`SELECT value FROM app_settings WHERE key = ?`)
	if s.Dialect == "mysql" {
		q = s.Rebind("SELECT `value` FROM app_settings WHERE `key` = ?")
	}
	var value string
	if err := s.DB.GetContext(ctx, &value, q, key); err != nil {
		if IsNoRows(err) {
			return fallback, nil
		}
		return "", err
	}
	return value, nil
}

func (s *Store) SetSettings(ctx context.Context, values map[string]string) error {
	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var q string
	switch s.Dialect {
	case "postgres":
		q = `INSERT INTO app_settings(key, value, updated_at) VALUES ($1, $2, NOW())
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`
	case "mysql":
		q = "INSERT INTO app_settings(`key`, `value`) VALUES (?, ?) ON DUPLICATE KEY UPDATE `value` = VALUES(`value`)"
	default:
		q = `INSERT INTO app_settings(key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP`
	}

	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, q, key, value); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) BoolSetting(ctx context.Context, key string, fallback bool) (bool, error) {
	fallbackValue := "0"
	if fallback {
		fallbackValue = "1"
	}
	value, err := s.GetSetting(ctx, key, fallbackValue)
	if err != nil {
		return fallback, err
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return fallback, nil
	}
}

func (s *Store) IntSetting(ctx context.Context, key string, fallback int) (int, error) {
	value, err := s.GetSetting(ctx, key, strconv.Itoa(fallback))
	if err != nil {
		return fallback, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback, nil
	}
	return n, nil
}
