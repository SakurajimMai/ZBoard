package store

import (
	"context"
	"time"
)

type RuntimeConfig struct {
	ID         int64      `db:"id" json:"id"`
	NodeID     int64      `db:"node_id" json:"node_id"`
	Version    string     `db:"version" json:"version"`
	ConfigHash string     `db:"config_hash" json:"config_hash"`
	ConfigJSON string     `db:"config_json" json:"config_json"`
	Status     string     `db:"status" json:"status"`
	AppliedAt  *time.Time `db:"applied_at" json:"applied_at"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
}

func (s *Store) CreateRuntimeConfig(ctx context.Context, nodeID int64, version, hash, json string) (int64, error) {
	return s.InsertReturningID(ctx,
		`INSERT INTO runtime_configs(node_id, version, config_hash, config_json) VALUES (?, ?, ?, ?)`,
		nodeID, version, hash, json,
	)
}

func (s *Store) ListRuntimeConfigs(ctx context.Context, nodeID int64) ([]RuntimeConfig, error) {
	q := s.Rebind(`SELECT id, node_id, version, config_hash, config_json, status, applied_at, created_at
		FROM runtime_configs WHERE node_id = ? ORDER BY id DESC`)
	var rows []RuntimeConfig
	if err := s.DB.SelectContext(ctx, &rows, q, nodeID); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) FindRuntimeConfigByVersion(ctx context.Context, version string) (*RuntimeConfig, error) {
	q := s.Rebind(`SELECT id, node_id, version, config_hash, config_json, status, applied_at, created_at
		FROM runtime_configs WHERE version = ?`)
	var r RuntimeConfig
	if err := s.DB.GetContext(ctx, &r, q, version); err != nil {
		return nil, err
	}
	return &r, nil
}
