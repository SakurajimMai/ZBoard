// Package runtime builds minimal viable Xray / sing-box configurations from a
// node + its enabled node_users. Both runtimes are configured to expose a
// per-user stats gRPC API on 127.0.0.1:10085 for Xray so the Node Agent can
// scrape uplink/downlink counters with reset-on-read semantics.
package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/zboard/api-server/internal/store"
)

// StatsAPIPort is the local-only port the runtime exposes for per-user stats.
// The Node Agent dials 127.0.0.1:StatsAPIPort over gRPC. We use the Xray
// `xray.app.stats.command.StatsService` API; sing-box exposes the same wire
// through experimental.v2ray_api when the runtime binary is built with
// with_v2ray_api.
const StatsAPIPort = 10085

const (
	defaultQUICCertificatePath = "/etc/zboard-agent/tls/server.crt"
	defaultQUICKeyPath         = "/etc/zboard-agent/tls/server.key"
)

// Build returns (configJSON, sha256, version) for the given runtime type.
func Build(node *store.Node, users []store.NodeUser, version string) (string, string, error) {
	if err := ValidateNode(node); err != nil {
		return "", "", err
	}
	var doc any
	switch node.Protocol {
	case "hysteria2":
		doc = singBoxHysteria2(node, users)
	case "tuic":
		doc = singBoxTUIC(node, users)
	default:
		switch node.RuntimeType {
		case "sing-box", "singbox":
			doc = singBox(node, users)
		default:
			doc = xray(node, users)
		}
	}
	body, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256(body)
	return string(body), hex.EncodeToString(sum[:]), nil
}

// ValidateNode blocks incomplete runtime inputs before the agent writes a
// config file that Xray/sing-box cannot start.
func ValidateNode(node *store.Node) error {
	if node == nil {
		return fmt.Errorf("node is nil")
	}
	protocol := strings.ToLower(strings.TrimSpace(node.Protocol))
	if protocol == "" {
		protocol = "vless"
	}
	transport := strings.ToLower(strings.TrimSpace(node.Transport))
	if transport == "" {
		transport = "tcp"
	}
	runtimeType := strings.ToLower(strings.TrimSpace(node.RuntimeType))
	if runtimeType == "" {
		runtimeType = "xray"
	}
	security := strings.ToLower(strings.TrimSpace(node.Security))
	if protocol == "hysteria2" || protocol == "tuic" {
		if transport != "udp" {
			return fmt.Errorf("%s nodes require udp transport", protocol)
		}
		if protocol == "hysteria2" {
			if _, _, err := parsePortRange(node.PortRange); err != nil {
				return err
			}
		}
	} else if runtimeType == "sing-box" || runtimeType == "singbox" {
		switch transport {
		case "tcp", "ws", "grpc":
		default:
			return fmt.Errorf("sing-box runtime does not support %s transport for %s nodes", transport, protocol)
		}
	} else {
		switch transport {
		case "tcp", "kcp", "mkcp", "ws", "grpc", "httpupgrade", "xhttp":
		default:
			return fmt.Errorf("xray runtime does not support %s transport for %s nodes", transport, protocol)
		}
	}
	if security != "reality" {
		return nil
	}
	if protocol != "vless" {
		return fmt.Errorf("reality security only supports vless nodes")
	}
	// Xray Reality server mode is accepted only on RAW TCP, XHTTP and gRPC.
	// Keeping this guard here avoids generating configs the agent cannot start.
	if runtimeType != "sing-box" && runtimeType != "singbox" {
		switch transport {
		case "tcp", "xhttp", "grpc":
		default:
			return fmt.Errorf("xray reality nodes only support tcp, xhttp or grpc transport")
		}
	}

	missing := make([]string, 0, 3)
	if strings.TrimSpace(node.RealityServerName) == "" {
		missing = append(missing, "reality_server_name")
	}
	if strings.TrimSpace(node.RealityPublicKey) == "" {
		missing = append(missing, "reality_public_key")
	}
	if strings.TrimSpace(node.RealityPrivateKey) == "" {
		missing = append(missing, "reality_private_key")
	}
	if len(missing) > 0 {
		return fmt.Errorf("reality node requires %s", strings.Join(missing, ", "))
	}
	return nil
}

// xrayUserEmail is the canonical "email" tag we set on every Xray client and
// the agent uses as the lookup key when parsing stats names. Must match the
// pattern emitted by Xray: `user>>>{email}>>>traffic>>>{uplink|downlink}`.
func xrayUserEmail(userID int64) string { return fmt.Sprintf("u%d@zboard", userID) }

func xray(node *store.Node, users []store.NodeUser) map[string]any {
	clients := make([]map[string]any, 0, len(users))
	for _, u := range users {
		if u.Enabled == 0 {
			continue
		}
		entry := map[string]any{"email": xrayUserEmail(u.UserID), "level": 0}
		switch node.Protocol {
		case "vless":
			entry["id"] = u.ClientID
			if node.Flow != "" {
				entry["flow"] = node.Flow
			}
		case "vmess":
			entry["id"] = u.ClientID
		case "trojan", "ss", "shadowsocks":
			entry["password"] = u.ClientID
		default:
			entry["id"] = u.ClientID
		}
		clients = append(clients, entry)
	}

	transport := xrayTransportNetwork(node.Transport)
	stream := map[string]any{
		"network":  transport,
		"security": node.Security,
	}
	switch node.Security {
	case "tls":
		tls := map[string]any{}
		if node.SNI != "" {
			tls["serverName"] = node.SNI
		}
		if node.Fingerprint != "" {
			tls["fingerprint"] = node.Fingerprint
		}
		if alpn := splitALPN(node.ALPN); len(alpn) > 0 {
			tls["alpn"] = alpn
		}
		stream["tlsSettings"] = tls
	case "reality":
		// Xray Reality inbound runs in server mode when dest/privateKey are set;
		// that branch expects serverNames and shortIds lists.
		dest := node.RealityDest
		if dest == "" {
			dest = defaultStr(node.RealityServerName, node.Host) + ":443"
		}
		reality := map[string]any{
			"show":        false,
			"dest":        dest,
			"serverNames": []string{node.RealityServerName},
			"shortIds":    []string{node.RealityShortID},
		}
		reality["privateKey"] = node.RealityPrivateKey
		if node.Fingerprint != "" {
			reality["fingerprint"] = node.Fingerprint
		}
		stream["realitySettings"] = reality
	}
	switch transport {
	case "mkcp":
		stream["kcpSettings"] = map[string]any{}
	case "ws":
		ws := map[string]any{"path": defaultStr(node.WSPath, "/")}
		if node.WSHost != "" {
			ws["headers"] = map[string]any{"Host": node.WSHost}
		}
		stream["wsSettings"] = ws
	case "grpc":
		stream["grpcSettings"] = map[string]any{"serviceName": node.GRPCServiceName}
	case "httpupgrade":
		stream["httpupgradeSettings"] = httpTransportSettings(node)
	case "xhttp":
		settings := httpTransportSettings(node)
		settings["mode"] = "auto"
		stream["xhttpSettings"] = settings
	}

	settings := map[string]any{
		"clients":    clients,
		"decryption": "none",
	}
	// SS-2022 lives under inbound `settings.method`/`password`. We map the
	// node's per-user client_id into the password field; the same per-user key
	// is what the subscription renderer hands clients.
	if node.Protocol == "ss" || node.Protocol == "shadowsocks" {
		settings = map[string]any{
			"method":  defaultStr(node.SSMethod, "2022-blake3-aes-128-gcm"),
			"clients": clients, // method+password per-client
		}
	}

	mainInbound := map[string]any{
		"tag":            "in",
		"listen":         "0.0.0.0",
		"port":           node.Port,
		"protocol":       node.Protocol,
		"settings":       settings,
		"streamSettings": stream,
	}
	if node.MuxEnabled != 0 {
		mainInbound["multiplexing"] = map[string]any{"enabled": true}
	}

	// Stats / API inbound: dokodemo-door listening locally; the api block
	// intercepts traffic tagged "api" to expose StatsService.
	apiInbound := map[string]any{
		"tag":      "api",
		"listen":   "127.0.0.1",
		"port":     StatsAPIPort,
		"protocol": "dokodemo-door",
		"settings": map[string]any{"address": "127.0.0.1"},
	}

	return map[string]any{
		"log":   map[string]any{"loglevel": "warning", "access": "none"},
		"stats": map[string]any{},
		"api": map[string]any{
			"tag":      "api",
			"services": []string{"StatsService"},
		},
		"policy": map[string]any{
			"levels": map[string]any{
				"0": map[string]any{
					"statsUserUplink":   true,
					"statsUserDownlink": true,
				},
			},
			"system": map[string]any{
				"statsInboundUplink":   true,
				"statsInboundDownlink": true,
			},
		},
		"inbounds": []any{mainInbound, apiInbound},
		"outbounds": []any{
			map[string]any{"protocol": "freedom", "tag": "direct"},
			map[string]any{"protocol": "blackhole", "tag": "blocked"},
		},
		"routing": map[string]any{
			"rules": []any{
				map[string]any{
					"type":        "field",
					"inboundTag":  []string{"api"},
					"outboundTag": "api",
				},
			},
		},
	}
}

func xrayTransportNetwork(transport string) string {
	switch strings.ToLower(strings.TrimSpace(transport)) {
	case "kcp", "mkcp":
		return "mkcp"
	case "":
		return "tcp"
	default:
		return transport
	}
}

func httpTransportSettings(node *store.Node) map[string]any {
	settings := map[string]any{"path": defaultStr(node.WSPath, "/")}
	if node.WSHost != "" {
		settings["host"] = node.WSHost
	}
	return settings
}

// singBoxUserName is the canonical user name we set on every sing-box user.
func singBoxUserName(userID int64) string { return fmt.Sprintf("u%d", userID) }

func singBoxStatsUsers(users []store.NodeUser) []string {
	out := make([]string, 0, len(users))
	for _, u := range users {
		if u.Enabled == 0 {
			continue
		}
		out = append(out, singBoxUserName(u.UserID))
	}
	return out
}

func withSingBoxStats(doc map[string]any, users []store.NodeUser) map[string]any {
	statUsers := singBoxStatsUsers(users)
	if len(statUsers) == 0 {
		return doc
	}
	doc["experimental"] = map[string]any{
		"v2ray_api": map[string]any{
			"listen": "127.0.0.1:" + strconv.Itoa(StatsAPIPort),
			"stats": map[string]any{
				"enabled": true,
				"users":   statUsers,
			},
		},
	}
	return doc
}

func singBox(node *store.Node, users []store.NodeUser) map[string]any {
	users2 := make([]map[string]any, 0, len(users))
	for _, u := range users {
		if u.Enabled == 0 {
			continue
		}
		name := singBoxUserName(u.UserID)
		entry := map[string]any{"name": name}
		switch node.Protocol {
		case "vless", "vmess":
			entry["uuid"] = u.ClientID
			if node.Protocol == "vless" && node.Flow != "" {
				entry["flow"] = node.Flow
			}
		case "trojan", "ss", "shadowsocks":
			entry["password"] = u.ClientID
		}
		users2 = append(users2, entry)
	}
	inbound := map[string]any{
		"type":        node.Protocol,
		"tag":         "in",
		"listen":      "0.0.0.0",
		"listen_port": node.Port,
		"users":       users2,
	}
	// SS-2022 inbounds carry method on the inbound, not per-user. sing-box
	// expects `method` + per-user passwords matching the SS-2022 cipher length.
	if node.Protocol == "ss" || node.Protocol == "shadowsocks" {
		inbound["type"] = "shadowsocks"
		inbound["method"] = defaultStr(node.SSMethod, "2022-blake3-aes-128-gcm")
	}
	switch node.Security {
	case "tls":
		tls := map[string]any{
			"enabled":     true,
			"server_name": defaultStr(node.SNI, node.Host),
		}
		if node.Fingerprint != "" {
			tls["utls"] = map[string]any{"enabled": true, "fingerprint": node.Fingerprint}
		}
		if alpn := splitALPN(node.ALPN); len(alpn) > 0 {
			tls["alpn"] = alpn
		}
		inbound["tls"] = tls
	case "reality":
		realityHandshake := map[string]any{}
		if node.RealityDest != "" {
			realityHandshake["server"] = node.RealityDest
		}
		reality := map[string]any{
			"enabled":   true,
			"short_id":  []string{node.RealityShortID},
			"handshake": realityHandshake,
		}
		if node.RealityPrivateKey != "" {
			reality["private_key"] = node.RealityPrivateKey
		}
		tls := map[string]any{
			"enabled":     true,
			"server_name": node.RealityServerName,
			"reality":     reality,
		}
		if node.Fingerprint != "" {
			tls["utls"] = map[string]any{"enabled": true, "fingerprint": node.Fingerprint}
		}
		inbound["tls"] = tls
	}
	switch node.Transport {
	case "ws":
		t := map[string]any{"type": "ws", "path": defaultStr(node.WSPath, "/")}
		if node.WSHost != "" {
			t["headers"] = map[string]any{"Host": node.WSHost}
		}
		inbound["transport"] = t
	case "grpc":
		inbound["transport"] = map[string]any{"type": "grpc", "service_name": node.GRPCServiceName}
	}

	return withSingBoxStats(map[string]any{
		"log":      map[string]any{"level": "warn"},
		"inbounds": []any{inbound},
		"outbounds": []any{
			map[string]any{"type": "direct", "tag": "direct"},
		},
	}, users)
}

func defaultStr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

// quicTLSBlock returns the sing-box `tls` block for a QUIC-based protocol.
// Hysteria2 / TUIC always run over QUIC, so TLS is mandatory and ALPN
// defaults to ["h3"].
func quicTLSBlock(node *store.Node) map[string]any {
	tls := map[string]any{
		"enabled":          true,
		"server_name":      defaultStr(node.SNI, node.Host),
		"certificate_path": defaultQUICCertificatePath,
		"key_path":         defaultQUICKeyPath,
	}
	alpn := splitALPN(node.ALPN)
	if len(alpn) == 0 {
		alpn = []string{"h3"}
	}
	tls["alpn"] = alpn
	return tls
}

func singBoxHysteria2(node *store.Node, users []store.NodeUser) map[string]any {
	users2 := make([]map[string]any, 0, len(users))
	for _, u := range users {
		if u.Enabled == 0 {
			continue
		}
		name := singBoxUserName(u.UserID)
		users2 = append(users2, map[string]any{
			"name":     name,
			"password": u.ClientID, // Hysteria2 user password = per-user secret
		})
	}
	inbound := map[string]any{
		"type":        "hysteria2",
		"tag":         "in",
		"listen":      "0.0.0.0",
		"listen_port": node.Port,
		"users":       users2,
		"tls":         quicTLSBlock(node),
	}
	if node.UpMbps > 0 {
		inbound["up_mbps"] = node.UpMbps
	}
	if node.DownMbps > 0 {
		inbound["down_mbps"] = node.DownMbps
	}
	// Salamander obfs is the only obfs Hysteria2 currently defines.
	if node.ObfsPassword != "" {
		inbound["obfs"] = map[string]any{
			"type":     "salamander",
			"password": node.ObfsPassword,
		}
	}

	doc := map[string]any{
		"log":      map[string]any{"level": "warn"},
		"inbounds": []any{inbound},
		"outbounds": []any{
			map[string]any{"type": "direct", "tag": "direct"},
		},
	}

	// Port hopping metadata is NOT embedded in the runtime config (sing-box
	// rejects unknown top-level fields). Instead, it's returned separately via
	// BuildPortHoppingMeta() and the nodesvc layer includes it in the task
	// payload — the agent reads it from there, not from the config JSON.

	return withSingBoxStats(doc, users)
}

// PortHoppingMeta returns the iptables setup/teardown commands for a Hysteria2
// node with port hopping. Returns nil when port_range is empty.
func PortHoppingMeta(node *store.Node) map[string]any {
	if node.PortRange == "" || node.Protocol != "hysteria2" {
		return nil
	}
	start, end, err := parsePortRange(node.PortRange)
	if err != nil {
		return nil
	}
	// iptables --dport uses colon syntax for ranges: 20000:40000
	iptablesRange := fmt.Sprintf("%d:%d", start, end)
	return map[string]any{
		"enabled":     true,
		"listen_port": node.Port,
		"port_range":  node.PortRange,
		"setup_cmds": []string{
			fmt.Sprintf("iptables -t nat -A PREROUTING -p udp --dport %s -j DNAT --to-destination :%d", iptablesRange, node.Port),
			fmt.Sprintf("ip6tables -t nat -A PREROUTING -p udp --dport %s -j DNAT --to-destination :%d", iptablesRange, node.Port),
		},
		"teardown_cmds": []string{
			fmt.Sprintf("iptables -t nat -D PREROUTING -p udp --dport %s -j DNAT --to-destination :%d", iptablesRange, node.Port),
			fmt.Sprintf("ip6tables -t nat -D PREROUTING -p udp --dport %s -j DNAT --to-destination :%d", iptablesRange, node.Port),
		},
	}
}

func parsePortRange(raw string) (int, int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, 0, nil
	}
	parts := strings.Split(raw, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("hysteria2 port_range must use start-end format")
	}
	start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("hysteria2 port_range start is invalid")
	}
	end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, fmt.Errorf("hysteria2 port_range end is invalid")
	}
	if start < 1 || start > 65535 || end < 1 || end > 65535 || start > end {
		return 0, 0, fmt.Errorf("hysteria2 port_range must be within 1-65535 and start <= end")
	}
	return start, end, nil
}

func singBoxTUIC(node *store.Node, users []store.NodeUser) map[string]any {
	users2 := make([]map[string]any, 0, len(users))
	for _, u := range users {
		if u.Enabled == 0 {
			continue
		}
		name := singBoxUserName(u.UserID)
		// TUIC clients authenticate with (uuid, password). We reuse client_id
		// as the uuid and obfs_password (or fallback to client_id) as the
		// password so the agent only needs to ship one secret per user.
		password := node.ObfsPassword
		if password == "" {
			password = u.ClientID
		}
		users2 = append(users2, map[string]any{
			"name":     name,
			"uuid":     u.ClientID,
			"password": password,
		})
	}
	inbound := map[string]any{
		"type":               "tuic",
		"tag":                "in",
		"listen":             "0.0.0.0",
		"listen_port":        node.Port,
		"users":              users2,
		"congestion_control": defaultStr(node.CongestionControl, "bbr"),
		"tls":                quicTLSBlock(node),
	}
	return withSingBoxStats(map[string]any{
		"log":      map[string]any{"level": "warn"},
		"inbounds": []any{inbound},
		"outbounds": []any{
			map[string]any{"type": "direct", "tag": "direct"},
		},
	}, users)
}

// splitALPN turns a comma-separated ALPN list (e.g. "h2,http/1.1") into a
// trimmed string slice. Empty input → nil so callers omit the JSON field.
func splitALPN(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
