// Package subrender renders subscription content for Clash Meta, sing-box and
// a Base64-wrapped URI list. All three formats share a normalized node model so
// fields don't drift between targets.
package subrender

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/zboard/api-server/internal/runtime"
	"github.com/zboard/api-server/internal/store"
)

// Item is a single normalized node-user record used by all three renderers.
type Item struct {
	NodeID            int64
	Name              string // "地区 + 节点名"，dedup-suffixed when colliding
	Region            string
	Host              string
	Port              int
	Protocol          string // vless / vmess / trojan / shadowsocks / ss / hysteria2 / tuic
	Transport         string // tcp / ws / grpc / udp
	Security          string // tls / none / reality
	UUID              string // for vless / vmess / tuic
	Password          string // for trojan / ss / hysteria2 / tuic
	Path              string // ws path
	WSHost            string // ws Host header
	Service           string // grpc serviceName
	SNI               string
	Fingerprint       string // utls fingerprint, e.g. "chrome"
	RealityPublicKey  string
	RealityShortID    string
	RealityServerName string
	Flow              string   // vless flow, e.g. "xtls-rprx-vision"
	ALPN              []string // e.g. ["h2","http/1.1"]
	SSMethod          string   // SS / SS-2022 cipher; never bleeds Reality private key
	ObfsPassword      string   // hysteria2 salamander obfs password (NOT Reality private key)
	CongestionControl string   // tuic / hysteria2 congestion control
	UpMbps            int      // hysteria2 client-advertised upload bandwidth
	DownMbps          int      // hysteria2 client-advertised download bandwidth
	PortRange         string   // hysteria2 port hopping range, e.g. "20000-40000"
}

// Build merges nodes + node_users into a deduplicated, ordered Item slice.
func Build(nodes []store.Node, nodeUsers []store.NodeUser) []Item {
	byNode := make(map[int64]store.NodeUser, len(nodeUsers))
	for _, nu := range nodeUsers {
		byNode[nu.NodeID] = nu
	}
	out := make([]Item, 0, len(nodes))
	for _, n := range nodes {
		nu, ok := byNode[n.ID]
		if !ok || nu.Enabled == 0 {
			continue
		}
		if runtime.ValidateNode(&n) != nil {
			continue
		}
		region := ""
		if n.Region != nil {
			region = *n.Region
		}
		display := strings.TrimSpace(strings.TrimSpace(region) + " " + strings.TrimSpace(n.Name))
		if display == "" {
			display = n.NodeCode
		}
		// Defaults: SNI -> host, ws path -> "/" — these are now real columns
		// on the node, but we tolerate empty strings for backward compatibility.
		sni := n.SNI
		if sni == "" {
			sni = n.Host
		}
		wsPath := n.WSPath
		if wsPath == "" {
			wsPath = "/"
		}
		wsHost := n.WSHost
		if wsHost == "" {
			wsHost = n.Host
		}

		out = append(out, Item{
			NodeID:            n.ID,
			Name:              display,
			Region:            region,
			Host:              n.Host,
			Port:              n.Port,
			Protocol:          strings.ToLower(n.Protocol),
			Transport:         strings.ToLower(n.Transport),
			Security:          strings.ToLower(n.Security),
			UUID:              nu.ClientID,
			Password:          nu.ClientID,
			Path:              wsPath,
			WSHost:            wsHost,
			Service:           n.GRPCServiceName,
			SNI:               sni,
			Fingerprint:       n.Fingerprint,
			RealityPublicKey:  n.RealityPublicKey,
			RealityShortID:    n.RealityShortID,
			RealityServerName: n.RealityServerName,
			Flow:              n.Flow,
			ALPN:              splitALPN(n.ALPN),
			SSMethod:          n.SSMethod,
			ObfsPassword:      n.ObfsPassword,
			CongestionControl: n.CongestionControl,
			UpMbps:            n.UpMbps,
			DownMbps:          n.DownMbps,
			PortRange:         n.PortRange,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NodeID < out[j].NodeID })
	dedupNames(out)
	return out
}

func dedupNames(items []Item) {
	count := map[string]int{}
	for i, it := range items {
		count[it.Name]++
		if count[it.Name] > 1 {
			items[i].Name = fmt.Sprintf("%s #%d", it.Name, count[it.Name])
		}
	}
}

// Base64 returns a base64-encoded URI list, one URI per node.
func Base64(items []Item) string {
	lines := make([]string, 0, len(items))
	for _, it := range items {
		if uri := uriFor(it); uri != "" {
			lines = append(lines, uri)
		}
	}
	joined := strings.Join(lines, "\n")
	return base64.StdEncoding.EncodeToString([]byte(joined))
}

func uriFor(it Item) string {
	switch it.Protocol {
	case "vless":
		q := url.Values{}
		q.Set("encryption", "none")
		q.Set("type", uriTransportType(it.Transport))
		if it.Security != "" {
			q.Set("security", it.Security)
		}
		if it.Flow != "" {
			q.Set("flow", it.Flow)
		}
		if it.SNI != "" {
			q.Set("sni", it.SNI)
		}
		if it.Fingerprint != "" {
			q.Set("fp", it.Fingerprint)
		}
		if len(it.ALPN) > 0 {
			q.Set("alpn", strings.Join(it.ALPN, ","))
		}
		if it.Security == "reality" {
			if it.RealityPublicKey != "" {
				q.Set("pbk", it.RealityPublicKey)
			}
			if it.RealityShortID != "" {
				q.Set("sid", it.RealityShortID)
			}
			if it.RealityServerName != "" {
				q.Set("sni", it.RealityServerName)
			}
		}
		switch it.Transport {
		case "ws", "httpupgrade", "xhttp":
			q.Set("path", it.Path)
			if it.WSHost != "" {
				q.Set("host", it.WSHost)
			}
			if it.Transport == "xhttp" {
				q.Set("mode", "auto")
			}
		case "grpc":
			if it.Service != "" {
				q.Set("serviceName", it.Service)
			}
		}
		host := net.JoinHostPort(it.Host, strconv.Itoa(it.Port))
		return fmt.Sprintf("vless://%s@%s?%s#%s", it.UUID, host, q.Encode(), url.QueryEscape(it.Name))
	case "trojan":
		host := net.JoinHostPort(it.Host, strconv.Itoa(it.Port))
		q := url.Values{}
		if it.SNI != "" {
			q.Set("sni", it.SNI)
		}
		if it.Fingerprint != "" {
			q.Set("fp", it.Fingerprint)
		}
		if len(it.ALPN) > 0 {
			q.Set("alpn", strings.Join(it.ALPN, ","))
		}
		if it.Transport == "ws" {
			q.Set("type", "ws")
			q.Set("path", it.Path)
			if it.WSHost != "" {
				q.Set("host", it.WSHost)
			}
		} else if it.Transport == "grpc" {
			q.Set("type", "grpc")
			if it.Service != "" {
				q.Set("serviceName", it.Service)
			}
		}
		return fmt.Sprintf("trojan://%s@%s?%s#%s", url.QueryEscape(it.Password), host, q.Encode(), url.QueryEscape(it.Name))
	case "ss", "shadowsocks":
		// ss://userinfo@host:port#name where userinfo = base64url("method:password").
		// SS-2022 ciphers (e.g. 2022-blake3-aes-128-gcm) require the method
		// to match what the inbound's `settings.method` was generated with.
		method := it.SSMethod
		if method == "" {
			method = "chacha20-ietf-poly1305"
		}
		userInfo := base64.RawURLEncoding.EncodeToString([]byte(method + ":" + it.Password))
		host := net.JoinHostPort(it.Host, strconv.Itoa(it.Port))
		return fmt.Sprintf("ss://%s@%s#%s", userInfo, host, url.QueryEscape(it.Name))
	case "hysteria2":
		// hysteria2://password@host:port?...#name
		// Port hopping must stay in mport. Putting a range in the authority
		// port makes the URI invalid for clients that use a strict URL parser.
		host := net.JoinHostPort(it.Host, strconv.Itoa(it.Port))
		q := url.Values{}
		sni := it.SNI
		if sni == "" {
			sni = it.Host
		}
		q.Set("sni", sni)
		if len(it.ALPN) > 0 {
			q.Set("alpn", strings.Join(it.ALPN, ","))
		} else {
			q.Set("alpn", "h3")
		}
		if it.ObfsPassword != "" {
			q.Set("obfs", "salamander")
			q.Set("obfs-password", it.ObfsPassword)
		}
		if it.UpMbps > 0 {
			q.Set("up", strconv.Itoa(it.UpMbps))
		}
		if it.DownMbps > 0 {
			q.Set("down", strconv.Itoa(it.DownMbps))
		}
		if it.Fingerprint != "" {
			q.Set("fp", it.Fingerprint)
		}
		// mport param for clients that support it (redundant with host:port-range
		// but some clients read mport explicitly).
		if it.PortRange != "" {
			q.Set("mport", it.PortRange)
		}
		return fmt.Sprintf("hysteria2://%s@%s?%s#%s", url.QueryEscape(it.Password), host, q.Encode(), url.QueryEscape(it.Name))
	case "tuic":
		// tuic://uuid:password@host:port?...#name
		host := net.JoinHostPort(it.Host, strconv.Itoa(it.Port))
		password := it.ObfsPassword
		if password == "" {
			password = it.Password
		}
		q := url.Values{}
		sni := it.SNI
		if sni == "" {
			sni = it.Host
		}
		q.Set("sni", sni)
		if len(it.ALPN) > 0 {
			q.Set("alpn", strings.Join(it.ALPN, ","))
		} else {
			q.Set("alpn", "h3")
		}
		q.Set("congestion_control", defaultStr(it.CongestionControl, "bbr"))
		if it.Fingerprint != "" {
			q.Set("fp", it.Fingerprint)
		}
		return fmt.Sprintf("tuic://%s:%s@%s?%s#%s",
			url.QueryEscape(it.UUID), url.QueryEscape(password), host,
			q.Encode(), url.QueryEscape(it.Name))
	default:
		return ""
	}
}

func defaultStr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func uriTransportType(transport string) string {
	if transport == "kcp" {
		return "mkcp"
	}
	return transport
}

// splitALPN turns a comma-separated string into a trimmed slice; mirrors the
// runtime package helper but kept package-local to avoid an import cycle.
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

// ClashMeta renders a Clash Meta YAML proxies list with a default group.
func ClashMeta(items []Item) string {
	var b strings.Builder
	b.WriteString("port: 7890\nsocks-port: 7891\nallow-lan: false\nmode: rule\nlog-level: info\n")
	b.WriteString("proxies:\n")
	names := make([]string, 0, len(items))
	for _, it := range items {
		names = append(names, it.Name)
		b.WriteString("  - name: ")
		b.WriteString(yamlQuote(it.Name))
		b.WriteByte('\n')
		switch it.Protocol {
		case "vless":
			b.WriteString("    type: vless\n")
		case "vmess":
			b.WriteString("    type: vmess\n")
		case "trojan":
			b.WriteString("    type: trojan\n")
		case "ss", "shadowsocks":
			cipher := it.SSMethod
			if cipher == "" {
				cipher = "chacha20-ietf-poly1305"
			}
			fmt.Fprintf(&b, "    type: ss\n    cipher: %s\n", cipher)
		case "hysteria2":
			b.WriteString("    type: hysteria2\n")
		case "tuic":
			b.WriteString("    type: tuic\n")
		default:
			b.WriteString("    type: vless\n")
		}
		fmt.Fprintf(&b, "    server: %s\n", it.Host)
		fmt.Fprintf(&b, "    port: %d\n", it.Port)
		switch it.Protocol {
		case "vless", "vmess":
			fmt.Fprintf(&b, "    uuid: %s\n", it.UUID)
		case "trojan", "ss", "shadowsocks":
			fmt.Fprintf(&b, "    password: %s\n", yamlQuote(it.Password))
		case "hysteria2":
			fmt.Fprintf(&b, "    password: %s\n", yamlQuote(it.Password))
			if it.ObfsPassword != "" {
				b.WriteString("    obfs: salamander\n")
				fmt.Fprintf(&b, "    obfs-password: %s\n", yamlQuote(it.ObfsPassword))
			}
			if it.UpMbps > 0 {
				fmt.Fprintf(&b, "    up: %d\n", it.UpMbps)
			}
			if it.DownMbps > 0 {
				fmt.Fprintf(&b, "    down: %d\n", it.DownMbps)
			}
			if it.PortRange != "" {
				fmt.Fprintf(&b, "    ports: %s\n", it.PortRange)
			}
		case "tuic":
			fmt.Fprintf(&b, "    uuid: %s\n", it.UUID)
			pw := it.ObfsPassword
			if pw == "" {
				pw = it.Password
			}
			fmt.Fprintf(&b, "    password: %s\n", yamlQuote(pw))
			cc := it.CongestionControl
			if cc == "" {
				cc = "bbr"
			}
			fmt.Fprintf(&b, "    congestion-controller: %s\n", cc)
		}
		if it.Protocol == "vless" && it.Flow != "" {
			fmt.Fprintf(&b, "    flow: %s\n", it.Flow)
		}
		if it.Security == "tls" {
			b.WriteString("    tls: true\n")
			if it.SNI != "" {
				fmt.Fprintf(&b, "    servername: %s\n", it.SNI)
			}
			if it.Fingerprint != "" {
				fmt.Fprintf(&b, "    client-fingerprint: %s\n", it.Fingerprint)
			}
			if len(it.ALPN) > 0 {
				b.WriteString("    alpn:\n")
				for _, a := range it.ALPN {
					fmt.Fprintf(&b, "      - %s\n", yamlQuote(a))
				}
			}
		} else if it.Security == "reality" {
			b.WriteString("    tls: true\n")
			if it.RealityServerName != "" {
				fmt.Fprintf(&b, "    servername: %s\n", it.RealityServerName)
			}
			b.WriteString("    reality-opts:\n")
			fmt.Fprintf(&b, "      public-key: %s\n", yamlQuote(it.RealityPublicKey))
			if it.RealityShortID != "" {
				fmt.Fprintf(&b, "      short-id: %s\n", yamlQuote(it.RealityShortID))
			}
			if it.Fingerprint != "" {
				fmt.Fprintf(&b, "    client-fingerprint: %s\n", it.Fingerprint)
			}
			if len(it.ALPN) > 0 {
				b.WriteString("    alpn:\n")
				for _, a := range it.ALPN {
					fmt.Fprintf(&b, "      - %s\n", yamlQuote(a))
				}
			}
		}
		if it.Transport == "ws" {
			b.WriteString("    network: ws\n    ws-opts:\n      path: ")
			b.WriteString(yamlQuote(it.Path))
			b.WriteByte('\n')
			if it.WSHost != "" {
				b.WriteString("      headers:\n        Host: ")
				b.WriteString(yamlQuote(it.WSHost))
				b.WriteByte('\n')
			}
		} else if it.Transport == "grpc" {
			b.WriteString("    network: grpc\n    grpc-opts:\n      grpc-service-name: ")
			b.WriteString(yamlQuote(it.Service))
			b.WriteByte('\n')
		} else if it.Transport == "httpupgrade" || it.Transport == "xhttp" {
			fmt.Fprintf(&b, "    network: %s\n", it.Transport)
		}
	}
	b.WriteString("proxy-groups:\n")
	b.WriteString("  - name: AUTO\n    type: select\n    proxies:\n")
	for _, n := range names {
		b.WriteString("      - ")
		b.WriteString(yamlQuote(n))
		b.WriteByte('\n')
	}
	b.WriteString("rules:\n  - MATCH,AUTO\n")
	return b.String()
}

// SingBox renders a sing-box outbound JSON document.
func SingBox(items []Item) string {
	type outbound map[string]any
	outs := []outbound{}
	for _, it := range items {
		o := outbound{"tag": it.Name, "server": it.Host, "server_port": it.Port}
		switch it.Protocol {
		case "vless":
			o["type"] = "vless"
			o["uuid"] = it.UUID
			o["packet_encoding"] = "xudp"
			if it.Flow != "" {
				o["flow"] = it.Flow
			}
		case "vmess":
			o["type"] = "vmess"
			o["uuid"] = it.UUID
			o["security"] = "auto"
		case "trojan":
			o["type"] = "trojan"
			o["password"] = it.Password
		case "ss", "shadowsocks":
			cipher := it.SSMethod
			if cipher == "" {
				cipher = "chacha20-ietf-poly1305"
			}
			o["type"] = "shadowsocks"
			o["method"] = cipher
			o["password"] = it.Password
		case "hysteria2":
			o["type"] = "hysteria2"
			o["password"] = it.Password
			if it.ObfsPassword != "" {
				o["obfs"] = map[string]any{"type": "salamander", "password": it.ObfsPassword}
			}
			if it.UpMbps > 0 {
				o["up_mbps"] = it.UpMbps
			}
			if it.DownMbps > 0 {
				o["down_mbps"] = it.DownMbps
			}
			if it.PortRange != "" {
				o["server_ports"] = it.PortRange
			}
		case "tuic":
			pw := it.ObfsPassword
			if pw == "" {
				pw = it.Password
			}
			o["type"] = "tuic"
			o["uuid"] = it.UUID
			o["password"] = pw
			cc := it.CongestionControl
			if cc == "" {
				cc = "bbr"
			}
			o["congestion_control"] = cc
		default:
			o["type"] = "vless"
			o["uuid"] = it.UUID
		}
		if it.Security == "tls" {
			tls := map[string]any{"enabled": true, "server_name": it.SNI}
			if it.Fingerprint != "" {
				tls["utls"] = map[string]any{"enabled": true, "fingerprint": it.Fingerprint}
			}
			if len(it.ALPN) > 0 {
				tls["alpn"] = it.ALPN
			}
			o["tls"] = tls
		} else if it.Security == "reality" {
			tls := map[string]any{
				"enabled":     true,
				"server_name": it.RealityServerName,
				"reality": map[string]any{
					"enabled":    true,
					"public_key": it.RealityPublicKey,
					"short_id":   it.RealityShortID,
				},
			}
			if it.Fingerprint != "" {
				tls["utls"] = map[string]any{"enabled": true, "fingerprint": it.Fingerprint}
			}
			if len(it.ALPN) > 0 {
				tls["alpn"] = it.ALPN
			}
			o["tls"] = tls
		}
		if it.Transport == "ws" {
			ws := map[string]any{"type": "ws", "path": it.Path}
			if it.WSHost != "" {
				ws["headers"] = map[string]any{"Host": it.WSHost}
			}
			o["transport"] = ws
		} else if it.Transport == "grpc" {
			o["transport"] = map[string]any{"type": "grpc", "service_name": it.Service}
		}
		outs = append(outs, o)
	}
	doc := map[string]any{
		"log":       map[string]any{"level": "info"},
		"inbounds":  []any{map[string]any{"type": "mixed", "listen": "127.0.0.1", "listen_port": 2080}},
		"outbounds": outs,
	}
	b, _ := json.MarshalIndent(doc, "", "  ")
	return string(b)
}

func yamlQuote(s string) string {
	if s == "" {
		return `""`
	}
	if strings.ContainsAny(s, ":#'\"\n\\") {
		return strconv.Quote(s)
	}
	return s
}
