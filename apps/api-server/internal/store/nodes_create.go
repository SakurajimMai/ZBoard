package store

import (
	"context"
	"strings"

	"github.com/zboard/api-server/internal/authx"
	"github.com/zboard/api-server/internal/config"
)

type CreateNodeInput struct {
	Name              string
	Region            string
	Host              string
	Port              int
	Protocol          string
	Transport         string
	Security          string
	RuntimeType       string
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
}

// CreateNode inserts a node and its node_agents shell row with a generated
// secret hash. Returns (nodeID, plaintextSecret) so the caller can hand the
// secret to the agent bootstrap response (and never persists it raw).
func (s *Store) CreateNode(ctx context.Context, in CreateNodeInput) (int64, string, error) {
	code, err := authx.NewToken(8)
	if err != nil {
		return 0, "", err
	}
	code = "node-" + code[:12]
	secret, err := authx.NewToken(32)
	if err != nil {
		return 0, "", err
	}

	if in.Protocol == "" {
		in.Protocol = "vless"
	}
	if in.Transport == "" {
		in.Transport = "tcp"
	}
	if in.Security == "" {
		in.Security = "tls"
	}
	if in.RuntimeType == "" {
		in.RuntimeType = "xray"
	}
	if in.WSPath == "" {
		in.WSPath = "/"
	}
	// SNI defaults to the host so we always have a usable TLS server name.
	if in.SNI == "" {
		in.SNI = in.Host
	}
	// SS-2022 default cipher when caller picks `ss` without specifying.
	if in.Protocol == "ss" || in.Protocol == "shadowsocks" {
		if in.SSMethod == "" {
			in.SSMethod = "2022-blake3-aes-128-gcm"
		}
	}
	// vless+vision is the GFW-resilient default when Reality is on.
	if in.Protocol == "vless" && in.Security == "reality" && in.Flow == "" {
		in.Flow = "xtls-rprx-vision"
	}
	// QUIC-based protocols: hysteria2 / tuic only run on sing-box. Force sane
	// defaults so admins don't have to know these wire details by heart.
	if in.Protocol == "hysteria2" || in.Protocol == "tuic" {
		in.RuntimeType = "sing-box"
		in.Transport = "udp"
		if in.Security == "" {
			in.Security = "tls"
		}
		if in.CongestionControl == "" {
			in.CongestionControl = "bbr"
		}
	}
	if in.Protocol == "hysteria2" {
		if in.UpMbps == 0 {
			in.UpMbps = 100
		}
		if in.DownMbps == 0 {
			in.DownMbps = 200
		}
	}

	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return 0, "", err
	}
	defer tx.Rollback()

	insertNode := `INSERT INTO nodes(node_code, name, region, host, port, protocol, transport, security, runtime_type,
		ws_path, ws_host, grpc_service_name, sni, fingerprint, reality_public_key, reality_short_id, reality_server_name,
		flow, alpn, mux_enabled, ss_method, reality_private_key, reality_dest,
		obfs_password, congestion_control, up_mbps, down_mbps)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	region := strings.TrimSpace(in.Region)
	args := []any{
		code, in.Name, region, in.Host, in.Port, in.Protocol, in.Transport, in.Security, in.RuntimeType,
		in.WSPath, in.WSHost, in.GRPCServiceName, in.SNI, in.Fingerprint,
		in.RealityPublicKey, in.RealityShortID, in.RealityServerName,
		in.Flow, in.ALPN, in.MuxEnabled, in.SSMethod, in.RealityPrivateKey, in.RealityDest,
		in.ObfsPassword, in.CongestionControl, in.UpMbps, in.DownMbps,
	}
	var nodeID int64
	if s.Dialect == config.DialectPostgres {
		q := s.Rebind(insertNode + " RETURNING id")
		if err := tx.QueryRowxContext(ctx, q, args...).Scan(&nodeID); err != nil {
			return 0, "", err
		}
	} else {
		res, err := tx.ExecContext(ctx, s.Rebind(insertNode), args...)
		if err != nil {
			return 0, "", err
		}
		nodeID, err = res.LastInsertId()
		if err != nil {
			return 0, "", err
		}
	}
	if _, err := tx.ExecContext(ctx, s.Rebind(
		`INSERT INTO node_agents(node_id, node_secret_hash) VALUES (?, ?)`),
		nodeID, authx.HashToken(secret),
	); err != nil {
		return 0, "", err
	}
	if err := tx.Commit(); err != nil {
		return 0, "", err
	}
	return nodeID, secret, nil
}

// ListUserIDsActive returns IDs of active users — used to provision node_users
// for a newly added node.
func (s *Store) ListUserIDsActive(ctx context.Context) ([]int64, error) {
	q := `SELECT id FROM users WHERE status = 'active'`
	var ids []int64
	if err := s.DB.SelectContext(ctx, &ids, q); err != nil {
		return nil, err
	}
	return ids, nil
}
