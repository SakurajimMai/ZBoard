package store

import (
	"context"
	"time"
)

type Node struct {
	ID                int64      `db:"id" json:"id"`
	NodeCode          string     `db:"node_code" json:"node_code"`
	Name              string     `db:"name" json:"name"`
	Region            *string    `db:"region" json:"region"`
	Host              string     `db:"host" json:"host"`
	Port              int        `db:"port" json:"port"`
	Protocol          string     `db:"protocol" json:"protocol"`
	Transport         string     `db:"transport" json:"transport"`
	Security          string     `db:"security" json:"security"`
	RuntimeType       string     `db:"runtime_type" json:"runtime_type"`
	AgentVersion      *string    `db:"agent_version" json:"agent_version"`
	Status            string     `db:"status" json:"status"`
	LastHeartbeatAt   *time.Time `db:"last_heartbeat_at" json:"last_heartbeat_at"`
	CreatedAt         time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at" json:"updated_at"`
	WSPath            string     `db:"ws_path" json:"ws_path"`
	WSHost            string     `db:"ws_host" json:"ws_host"`
	GRPCServiceName   string     `db:"grpc_service_name" json:"grpc_service_name"`
	SNI               string     `db:"sni" json:"sni"`
	Fingerprint       string     `db:"fingerprint" json:"fingerprint"`
	RealityPublicKey  string     `db:"reality_public_key" json:"reality_public_key"`
	RealityShortID    string     `db:"reality_short_id" json:"reality_short_id"`
	RealityServerName string     `db:"reality_server_name" json:"reality_server_name"`
	Flow              string     `db:"flow" json:"flow"`
	ALPN              string     `db:"alpn" json:"alpn"` // comma-separated, e.g. "h2,http/1.1"
	MuxEnabled        int        `db:"mux_enabled" json:"mux_enabled"`
	SSMethod          string     `db:"ss_method" json:"ss_method"` // e.g. 2022-blake3-aes-128-gcm
	RealityPrivateKey string     `db:"reality_private_key" json:"reality_private_key"`
	RealityDest       string     `db:"reality_dest" json:"reality_dest"`             // e.g. www.cloudflare.com:443
	ObfsPassword      string     `db:"obfs_password" json:"obfs_password"`           // hysteria2 obfs password
	CongestionControl string     `db:"congestion_control" json:"congestion_control"` // hysteria2 / tuic, e.g. "bbr"
	UpMbps            int        `db:"up_mbps" json:"up_mbps"`                       // hysteria2 advertised upload bandwidth
	DownMbps          int        `db:"down_mbps" json:"down_mbps"`                   // hysteria2 advertised download bandwidth
	PortRange         string     `db:"port_range" json:"port_range"`                 // hysteria2 port hopping, e.g. "20000-40000"
}

type NodeView struct {
	Node
	ActiveUserCount int64  `db:"active_user_count" json:"active_user_count"`
	RuntimeStatus   string `db:"runtime_status" json:"runtime_status"`
	HealthStatus    string `db:"-" json:"health_status"`
	HealthLabel     string `db:"-" json:"health_label"`
}

type UpdateNodeInput struct {
	Name              string
	Region            string
	Host              string
	Port              int
	Protocol          string
	Transport         string
	Security          string
	RuntimeType       string
	Status            string
	WSPath            string
	WSHost            string
	GRPCServiceName   string
	SNI               string
	Fingerprint       string
	RealityPublicKey  string
	RealityShortID    string
	RealityServerName string
	Flow              string
	ALPN              string
	MuxEnabled        int
	SSMethod          string
	RealityPrivateKey string
	RealityDest       string
	ObfsPassword      string
	CongestionControl string
	UpMbps            int
	DownMbps          int
	PortRange         string
}

const nodeColumns = `id, node_code, name, region, host, port, protocol, transport, security,
	runtime_type, agent_version, status, last_heartbeat_at, created_at, updated_at,
	ws_path, ws_host, grpc_service_name, sni, fingerprint,
	reality_public_key, reality_short_id, reality_server_name,
	flow, alpn, mux_enabled, ss_method, reality_private_key, reality_dest,
	obfs_password, congestion_control, up_mbps, down_mbps, port_range`

func (s *Store) ListActiveNodes(ctx context.Context) ([]Node, error) {
	q := `SELECT ` + nodeColumns + ` FROM nodes WHERE status = 'active' ORDER BY id ASC`
	var rows []Node
	if err := s.DB.SelectContext(ctx, &rows, q); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Store) ListAllNodes(ctx context.Context) ([]Node, error) {
	rows, _, err := s.ListAllNodesPage(ctx, PageParams{Page: 1, PageSize: 500})
	return rows, err
}

func (s *Store) ListAllNodesPage(ctx context.Context, p PageParams) ([]Node, int64, error) {
	p = NormalizePage(p)
	var total int64
	if err := s.DB.GetContext(ctx, &total, `SELECT COUNT(*) FROM nodes`); err != nil {
		return nil, 0, err
	}
	q := `SELECT ` + nodeColumns + ` FROM nodes ORDER BY id ASC`
	q = s.Rebind(q + ` LIMIT ? OFFSET ?`)
	var rows []Node
	if err := s.DB.SelectContext(ctx, &rows, q, p.PageSize, p.Offset()); err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (s *Store) ListAllNodeViewsPage(ctx context.Context, p PageParams, offlineThresholdSeconds int) ([]NodeView, int64, error) {
	p = NormalizePage(p)
	if offlineThresholdSeconds <= 0 {
		offlineThresholdSeconds = 120
	}
	var total int64
	if err := s.DB.GetContext(ctx, &total, `SELECT COUNT(*) FROM nodes`); err != nil {
		return nil, 0, err
	}
	q := `SELECT ` + prefixedNodeColumns("n") + `,
		COALESCE(nu.active_user_count, 0) AS active_user_count,
		COALESCE(ah.runtime_status, '') AS runtime_status
		FROM nodes n
		LEFT JOIN (
			SELECT node_id, COUNT(*) AS active_user_count
			FROM node_users
			WHERE enabled = 1
			GROUP BY node_id
		) nu ON nu.node_id = n.id
		LEFT JOIN (
			SELECT h.node_id, h.runtime_status
			FROM agent_heartbeats h
			INNER JOIN (
				SELECT node_id, MAX(reported_at) AS reported_at
				FROM agent_heartbeats
				GROUP BY node_id
			) latest ON latest.node_id = h.node_id AND latest.reported_at = h.reported_at
		) ah ON ah.node_id = n.id
		ORDER BY n.id ASC`
	q = s.Rebind(q + ` LIMIT ? OFFSET ?`)
	var rows []NodeView
	if err := s.DB.SelectContext(ctx, &rows, q, p.PageSize, p.Offset()); err != nil {
		return nil, 0, err
	}
	now := Now()
	for i := range rows {
		rows[i].HealthStatus, rows[i].HealthLabel = nodeHealth(rows[i], now, offlineThresholdSeconds)
	}
	return rows, total, nil
}

func prefixedNodeColumns(alias string) string {
	cols := []string{
		"id", "node_code", "name", "region", "host", "port", "protocol", "transport", "security",
		"runtime_type", "agent_version", "status", "last_heartbeat_at", "created_at", "updated_at",
		"ws_path", "ws_host", "grpc_service_name", "sni", "fingerprint",
		"reality_public_key", "reality_short_id", "reality_server_name",
		"flow", "alpn", "mux_enabled", "ss_method", "reality_private_key", "reality_dest",
		"obfs_password", "congestion_control", "up_mbps", "down_mbps", "port_range",
	}
	out := ""
	for i, col := range cols {
		if i > 0 {
			out += ", "
		}
		out += alias + "." + col + " AS " + col
	}
	return out
}

func nodeHealth(n NodeView, now time.Time, offlineThresholdSeconds int) (string, string) {
	if n.Status != "active" {
		return "red", "异常"
	}
	if n.LastHeartbeatAt == nil || now.Sub(*n.LastHeartbeatAt) > time.Duration(offlineThresholdSeconds)*time.Second {
		return "red", "异常"
	}
	if n.RuntimeStatus != "" && n.RuntimeStatus != "running" {
		return "red", "异常"
	}
	if n.ActiveUserCount > 0 {
		return "green", "使用中"
	}
	return "yellow", "空闲"
}

func (s *Store) FindNodeByID(ctx context.Context, id int64) (*Node, error) {
	q := s.Rebind(`SELECT ` + nodeColumns + ` FROM nodes WHERE id = ?`)
	var n Node
	if err := s.DB.GetContext(ctx, &n, q, id); err != nil {
		return nil, err
	}
	return &n, nil
}

func (s *Store) UpdateNode(ctx context.Context, id int64, in UpdateNodeInput) error {
	q := s.Rebind(`UPDATE nodes SET name = ?, region = ?, host = ?, port = ?,
		protocol = ?, transport = ?, security = ?, runtime_type = ?, status = ?,
		ws_path = ?, ws_host = ?, grpc_service_name = ?, sni = ?, fingerprint = ?,
		reality_public_key = ?, reality_short_id = ?, reality_server_name = ?,
		flow = ?, alpn = ?, mux_enabled = ?, ss_method = ?, reality_private_key = ?, reality_dest = ?,
		obfs_password = ?, congestion_control = ?, up_mbps = ?, down_mbps = ?, port_range = ?,
		updated_at = CURRENT_TIMESTAMP WHERE id = ?`)
	region := in.Region
	_, err := s.DB.ExecContext(ctx, q,
		in.Name, region, in.Host, in.Port,
		in.Protocol, in.Transport, in.Security, in.RuntimeType, in.Status,
		in.WSPath, in.WSHost, in.GRPCServiceName, in.SNI, in.Fingerprint,
		in.RealityPublicKey, in.RealityShortID, in.RealityServerName,
		in.Flow, in.ALPN, in.MuxEnabled, in.SSMethod, in.RealityPrivateKey, in.RealityDest,
		in.ObfsPassword, in.CongestionControl, in.UpMbps, in.DownMbps, in.PortRange,
		id,
	)
	return err
}
