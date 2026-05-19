// Package runtime builds minimal viable Xray / sing-box configurations from a
// node + its enabled node_users. Both runtimes are configured to expose a
// per-user stats gRPC API on 127.0.0.1:10085 so the Node Agent can scrape
// uplink/downlink counters with reset-on-read semantics.
package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zboard/api-server/internal/store"
)

// StatsAPIPort is the local-only port the runtime exposes for per-user stats.
// The Node Agent dials 127.0.0.1:StatsAPIPort over gRPC. We use the Xray
// `xray.app.stats.command.StatsService` API; sing-box exposes the same wire
// protocol via `experimental.v2ray_api`.
const StatsAPIPort = 10085

// Build returns (configJSON, sha256, version) for the given runtime type.
func Build(node *store.Node, users []store.NodeUser, version string) (string, string, error) {
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
		entry := map[string]any{"email": xrayUserEmail(u.UserID)}
		switch node.Protocol {
		case "vless":
			entry["id"] = u.ClientID
			// Per-client flow only set when caller picked one. xtls-rprx-vision
			// is the GFW-resilient default Reality nodes get on create.
			entry["flow"] = node.Flow
		case "vmess":
			entry["id"] = u.ClientID
		case "trojan", "ss", "shadowsocks":
			entry["password"] = u.ClientID
		default:
			entry["id"] = u.ClientID
		}
		clients = append(clients, entry)
	}

	stream := map[string]any{
		"network":  node.Transport,
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
		reality := map[string]any{
			"show":        false,
			"serverNames": []string{node.RealityServerName},
		}
		if node.RealityDest != "" {
			reality["dest"] = node.RealityDest
		}
		// Reality requires the *server's* private key on the inbound; the
		// public key is exposed to clients. The private key must never leak
		// to the subscription renderer.
		if node.RealityPrivateKey != "" {
			reality["privateKey"] = node.RealityPrivateKey
		}
		if node.RealityShortID != "" {
			reality["shortIds"] = []string{node.RealityShortID}
		}
		if node.Fingerprint != "" {
			reality["fingerprint"] = node.Fingerprint
		}
		stream["realitySettings"] = reality
	}
	switch node.Transport {
	case "ws":
		ws := map[string]any{"path": defaultStr(node.WSPath, "/")}
		if node.WSHost != "" {
			ws["headers"] = map[string]any{"Host": node.WSHost}
		}
		stream["wsSettings"] = ws
	case "grpc":
		stream["grpcSettings"] = map[string]any{"serviceName": node.GRPCServiceName}
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
		"log":   map[string]any{"loglevel": "warning"},
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

// singBoxUserName is the canonical user name we set on every sing-box user.
// sing-box's v2ray_api exposes stats keyed by this name.
func singBoxUserName(userID int64) string { return fmt.Sprintf("u%d", userID) }

func singBox(node *store.Node, users []store.NodeUser) map[string]any {
	users2 := make([]map[string]any, 0, len(users))
	names := make([]string, 0, len(users))
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
		names = append(names, name)
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

	return map[string]any{
		"log":      map[string]any{"level": "warn"},
		"inbounds": []any{inbound},
		"outbounds": []any{
			map[string]any{"type": "direct", "tag": "direct"},
			map[string]any{"type": "block", "tag": "block"},
		},
		"experimental": map[string]any{
			"v2ray_api": map[string]any{
				"listen": fmt.Sprintf("127.0.0.1:%d", StatsAPIPort),
				"stats": map[string]any{
					"enabled": true,
					"users":   names,
				},
			},
		},
	}
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
		"enabled":     true,
		"server_name": defaultStr(node.SNI, node.Host),
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
	names := make([]string, 0, len(users))
	for _, u := range users {
		if u.Enabled == 0 {
			continue
		}
		name := singBoxUserName(u.UserID)
		users2 = append(users2, map[string]any{
			"name":     name,
			"password": u.ClientID, // Hysteria2 user password = per-user secret
		})
		names = append(names, name)
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
	return map[string]any{
		"log":      map[string]any{"level": "warn"},
		"inbounds": []any{inbound},
		"outbounds": []any{
			map[string]any{"type": "direct", "tag": "direct"},
			map[string]any{"type": "block", "tag": "block"},
		},
		"experimental": map[string]any{
			"v2ray_api": map[string]any{
				"listen": fmt.Sprintf("127.0.0.1:%d", StatsAPIPort),
				"stats":  map[string]any{"enabled": true, "users": names},
			},
		},
	}
}

func singBoxTUIC(node *store.Node, users []store.NodeUser) map[string]any {
	users2 := make([]map[string]any, 0, len(users))
	names := make([]string, 0, len(users))
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
		names = append(names, name)
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
	return map[string]any{
		"log":      map[string]any{"level": "warn"},
		"inbounds": []any{inbound},
		"outbounds": []any{
			map[string]any{"type": "direct", "tag": "direct"},
			map[string]any{"type": "block", "tag": "block"},
		},
		"experimental": map[string]any{
			"v2ray_api": map[string]any{
				"listen": fmt.Sprintf("127.0.0.1:%d", StatsAPIPort),
				"stats":  map[string]any{"enabled": true, "users": names},
			},
		},
	}
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
