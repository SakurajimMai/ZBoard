package store

import (
	"context"
	"time"
)

func (s *Store) GetAgentSecretHash(ctx context.Context, nodeID int64) (string, error) {
	q := s.Rebind(`SELECT node_secret_hash FROM node_agents WHERE node_id = ?`)
	var h string
	if err := s.DB.GetContext(ctx, &h, q, nodeID); err != nil {
		return "", err
	}
	return h, nil
}

func (s *Store) InsertAgentNonce(ctx context.Context, nodeID int64, nonce string, ts int64) error {
	q := s.Rebind(`INSERT INTO agent_nonces(node_id, nonce, ts) VALUES (?, ?, ?)`)
	_, err := s.DB.ExecContext(ctx, q, nodeID, nonce, ts)
	return err
}

// PurgeAgentNonces deletes nonces older than the given timestamp. Hook this up
// to the worker for periodic cleanup.
func (s *Store) PurgeAgentNonces(ctx context.Context, olderThan int64) error {
	q := s.Rebind(`DELETE FROM agent_nonces WHERE ts < ?`)
	_, err := s.DB.ExecContext(ctx, q, olderThan)
	return err
}

func (s *Store) MarkAgentRegistered(ctx context.Context, nodeID int64, version, osInfo, runtimeInfo string) error {
	q := s.Rebind(`UPDATE node_agents SET status = 'active', version = ?, os_info = ?,
		runtime_info = ?, registered_at = ?, last_seen_at = ? WHERE node_id = ?`)
	now := Now()
	_, err := s.DB.ExecContext(ctx, q, version, osInfo, runtimeInfo, now, now, nodeID)
	return err
}

type HeartbeatInput struct {
	NodeID        int64
	AgentVersion  string
	RuntimeStatus string
	RuntimeInfo   string
	SystemLoad    string
	ReportedAt    time.Time
}

func (s *Store) RecordHeartbeat(ctx context.Context, in HeartbeatInput) error {
	if _, err := s.DB.ExecContext(ctx,
		s.Rebind(`INSERT INTO agent_heartbeats(node_id, agent_version, runtime_status, runtime_info, system_load, reported_at)
			VALUES (?, ?, ?, ?, ?, ?)`),
		in.NodeID, in.AgentVersion, in.RuntimeStatus, in.RuntimeInfo, in.SystemLoad, in.ReportedAt,
	); err != nil {
		return err
	}
	if _, err := s.DB.ExecContext(ctx,
		s.Rebind(`UPDATE nodes SET last_heartbeat_at = ?, agent_version = ? WHERE id = ?`),
		in.ReportedAt, in.AgentVersion, in.NodeID,
	); err != nil {
		return err
	}
	if _, err := s.DB.ExecContext(ctx,
		s.Rebind(`UPDATE node_agents SET last_seen_at = ?, status = 'active' WHERE node_id = ?`),
		in.ReportedAt, in.NodeID,
	); err != nil {
		return err
	}
	return nil
}
