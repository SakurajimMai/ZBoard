package runtime_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/zboard/api-server/internal/runtime"
	"github.com/zboard/api-server/internal/store"
)

func TestBuildXrayWithReality(t *testing.T) {
	node := &store.Node{
		ID: 1, Name: "JP", Host: "jp.example.com", Port: 443,
		Protocol: "vless", Transport: "tcp", Security: "reality",
		RuntimeType:       "xray",
		Fingerprint:       "chrome",
		RealityPublicKey:  "PBK",
		RealityPrivateKey: "PRIVATE-KEY-HEX",
		RealityShortID:    "sid",
		RealityServerName: "www.cloudflare.com",
	}
	users := []store.NodeUser{
		{UserID: 1, ClientID: "uuid-1", Enabled: 1},
		{UserID: 2, ClientID: "uuid-2", Enabled: 0}, // disabled, must be excluded
	}
	body, hash, err := runtime.Build(node, users, "v1")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if !strings.Contains(body, `"realitySettings"`) {
		t.Errorf("expected realitySettings in body:\n%s", body)
	}
	if strings.Contains(body, "uuid-2") {
		t.Errorf("disabled user should not appear")
	}
	if !strings.Contains(body, `"serverNames"`) {
		t.Errorf("xray reality inbound must include serverNames, got body:\n%s", body)
	}
	if strings.Contains(body, `"serverName"`) {
		t.Errorf("xray reality inbound must not use outbound-only serverName, got body:\n%s", body)
	}
	// Hash is content-addressed, so two builds of same input produce same hash.
	body2, hash2, _ := runtime.Build(node, users, "v1")
	if body != body2 || hash != hash2 {
		t.Fatalf("non-deterministic build")
	}
}

func TestBuildSingBoxWithWS(t *testing.T) {
	node := &store.Node{
		ID: 1, Name: "US", Host: "us.example.com", Port: 443,
		Protocol: "vless", Transport: "ws", Security: "tls",
		RuntimeType: "sing-box",
		WSPath:      "/api/v1", WSHost: "cdn.example.com",
		SNI:         "cdn.example.com",
		Fingerprint: "firefox",
	}
	users := []store.NodeUser{{UserID: 1, ClientID: "uuid-1", Enabled: 1}}
	body, _, err := runtime.Build(node, users, "v1")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, body)
	}
	inbounds := doc["inbounds"].([]any)
	if len(inbounds) != 1 {
		t.Fatalf("expected 1 inbound")
	}
	in := inbounds[0].(map[string]any)
	tr, _ := in["transport"].(map[string]any)
	if tr == nil || tr["type"] != "ws" || tr["path"] != "/api/v1" {
		t.Fatalf("transport unexpected: %#v", tr)
	}
	tls, _ := in["tls"].(map[string]any)
	if tls == nil || tls["server_name"] != "cdn.example.com" {
		t.Fatalf("tls unexpected: %#v", tls)
	}
	outs, _ := doc["outbounds"].([]any)
	for _, raw := range outs {
		outbound, _ := raw.(map[string]any)
		if outbound["type"] == "block" {
			t.Fatalf("sing-box config should not emit legacy block outbound: %#v", outbound)
		}
	}
}

func TestXrayConfigExposesStatsAPI(t *testing.T) {
	node := &store.Node{
		ID: 1, Name: "T", Host: "h", Port: 443,
		Protocol: "vless", Transport: "tcp", Security: "tls",
		RuntimeType: "xray",
	}
	users := []store.NodeUser{
		{UserID: 7, ClientID: "uuid-7", Enabled: 1},
		{UserID: 9, ClientID: "uuid-9", Enabled: 1},
	}
	body, _, err := runtime.Build(node, users, "v1")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, body)
	}
	if _, ok := doc["stats"]; !ok {
		t.Errorf("missing top-level 'stats' block")
	}
	api, ok := doc["api"].(map[string]any)
	if !ok || api["tag"] != "api" {
		t.Errorf("expected api block with tag=api, got %#v", doc["api"])
	}
	policy, ok := doc["policy"].(map[string]any)
	if !ok {
		t.Fatalf("expected policy block")
	}
	levels := policy["levels"].(map[string]any)["0"].(map[string]any)
	if levels["statsUserUplink"] != true || levels["statsUserDownlink"] != true {
		t.Errorf("policy.levels.0 missing user stats flags: %#v", levels)
	}
	inbounds := doc["inbounds"].([]any)
	if len(inbounds) != 2 {
		t.Fatalf("expected 2 inbounds (main + api), got %d", len(inbounds))
	}
	apiIn, _ := inbounds[1].(map[string]any)
	if apiIn["tag"] != "api" || apiIn["protocol"] != "dokodemo-door" || apiIn["listen"] != "127.0.0.1" {
		t.Errorf("expected api dokodemo-door inbound on 127.0.0.1, got %#v", apiIn)
	}
	port, _ := apiIn["port"].(float64)
	if int(port) != runtime.StatsAPIPort {
		t.Errorf("api inbound port = %v, want %d", port, runtime.StatsAPIPort)
	}

	// Each client must have an `email` matching the agent-side parser
	// (`u<id>@zboard`).
	main, _ := inbounds[0].(map[string]any)
	settings, _ := main["settings"].(map[string]any)
	clients, _ := settings["clients"].([]any)
	if len(clients) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(clients))
	}
	wantEmails := map[string]bool{"u7@zboard": false, "u9@zboard": false}
	for _, c := range clients {
		m, _ := c.(map[string]any)
		if e, _ := m["email"].(string); e != "" {
			wantEmails[e] = true
		}
	}
	for k, ok := range wantEmails {
		if !ok {
			t.Errorf("missing client email %s", k)
		}
	}
}

func TestSingBoxConfigEnablesV2RayStatsAPI(t *testing.T) {
	node := &store.Node{
		ID: 1, Name: "T", Host: "h", Port: 443,
		Protocol: "vless", Transport: "tcp", Security: "tls",
		RuntimeType: "sing-box",
	}
	users := []store.NodeUser{
		{UserID: 11, ClientID: "uuid-11", Enabled: 1},
		{UserID: 22, ClientID: "uuid-22", Enabled: 1},
	}
	body, _, err := runtime.Build(node, users, "v1")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, body)
	}
	exp, ok := doc["experimental"].(map[string]any)
	if !ok {
		t.Fatalf("sing-box runtime config missing experimental: %#v", doc)
	}
	api, ok := exp["v2ray_api"].(map[string]any)
	if !ok {
		t.Fatalf("sing-box runtime config missing v2ray_api: %#v", exp)
	}
	stats := api["stats"].(map[string]any)
	statUsers := stats["users"].([]any)
	if len(statUsers) != 2 || statUsers[0] != "u11" || statUsers[1] != "u22" {
		t.Fatalf("v2ray_api.stats.users = %#v, want [u11 u22]", statUsers)
	}
}

// Stage 13: VLESS+Vision flow + ALPN must reach the inbound clients and the
// stream's tlsSettings.alpn.
func TestXrayVisionAndALPN(t *testing.T) {
	node := &store.Node{
		ID: 1, Name: "VIS", Host: "vis.example.com", Port: 443,
		Protocol: "vless", Transport: "tcp", Security: "tls",
		RuntimeType: "xray",
		Flow:        "xtls-rprx-vision",
		ALPN:        "h2,http/1.1",
		SNI:         "vis.example.com",
		MuxEnabled:  1,
	}
	users := []store.NodeUser{{UserID: 1, ClientID: "uuid-1", Enabled: 1}}
	body, _, err := runtime.Build(node, users, "v1")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	main := doc["inbounds"].([]any)[0].(map[string]any)
	clients := main["settings"].(map[string]any)["clients"].([]any)
	if got := clients[0].(map[string]any)["flow"]; got != "xtls-rprx-vision" {
		t.Errorf("client flow = %v, want xtls-rprx-vision", got)
	}
	if _, ok := main["multiplexing"].(map[string]any); !ok {
		t.Errorf("expected multiplexing block when MuxEnabled=1")
	}
	stream := main["streamSettings"].(map[string]any)
	tls := stream["tlsSettings"].(map[string]any)
	alpn, _ := tls["alpn"].([]any)
	if len(alpn) != 2 || alpn[0] != "h2" || alpn[1] != "http/1.1" {
		t.Errorf("tlsSettings.alpn = %#v, want [h2 http/1.1]", alpn)
	}
}

func TestXrayOmitsEmptyVlessFlow(t *testing.T) {
	node := &store.Node{
		ID: 1, Name: "NO-FLOW", Host: "noflow.example.com", Port: 443,
		Protocol: "vless", Transport: "tcp", Security: "tls",
		RuntimeType: "xray",
	}
	users := []store.NodeUser{{UserID: 1, ClientID: "uuid-1", Enabled: 1}}
	body, _, err := runtime.Build(node, users, "v1")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	main := doc["inbounds"].([]any)[0].(map[string]any)
	clients := main["settings"].(map[string]any)["clients"].([]any)
	if _, ok := clients[0].(map[string]any)["flow"]; ok {
		t.Fatalf("empty flow must not be emitted into xray clients: %#v", clients[0])
	}
}

func TestXrayModernTransportSettings(t *testing.T) {
	cases := []struct {
		name        string
		transport   string
		wantNetwork string
		wantBlock   string
	}{
		{name: "mkcp", transport: "kcp", wantNetwork: "mkcp", wantBlock: "kcpSettings"},
		{name: "httpupgrade", transport: "httpupgrade", wantNetwork: "httpupgrade", wantBlock: "httpupgradeSettings"},
		{name: "xhttp", transport: "xhttp", wantNetwork: "xhttp", wantBlock: "xhttpSettings"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			node := &store.Node{
				ID: 1, Name: "MODERN", Host: "modern.example.com", Port: 443,
				Protocol: "vless", Transport: tc.transport, Security: "tls",
				RuntimeType: "xray",
				WSPath:      "/edge",
				WSHost:      "cdn.example.com",
			}
			users := []store.NodeUser{{UserID: 1, ClientID: "uuid-1", Enabled: 1}}
			body, _, err := runtime.Build(node, users, "v1")
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			var doc map[string]any
			if err := json.Unmarshal([]byte(body), &doc); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			main := doc["inbounds"].([]any)[0].(map[string]any)
			stream := main["streamSettings"].(map[string]any)
			if stream["network"] != tc.wantNetwork {
				t.Fatalf("network = %v, want %s", stream["network"], tc.wantNetwork)
			}
			block, ok := stream[tc.wantBlock].(map[string]any)
			if !ok {
				t.Fatalf("missing %s in stream settings: %#v", tc.wantBlock, stream)
			}
			switch tc.transport {
			case "httpupgrade", "xhttp":
				if block["path"] != "/edge" || block["host"] != "cdn.example.com" {
					t.Fatalf("%s = %#v, want path/host", tc.wantBlock, block)
				}
			case "kcp":
				if len(block) != 0 {
					t.Fatalf("kcpSettings = %#v, want empty modern settings", block)
				}
			}
		})
	}
}

// Stage 13: Reality must include the *server's* private key + dest in the
// generated runtime config (it must NOT leak through subscription, but it MUST
// be present in runtime config the agent applies).
func TestXrayRealityPrivateKeyAndDest(t *testing.T) {
	node := &store.Node{
		ID: 1, Name: "JP-R", Host: "jp.example.com", Port: 443,
		Protocol: "vless", Transport: "tcp", Security: "reality",
		RuntimeType:       "xray",
		Flow:              "xtls-rprx-vision",
		Fingerprint:       "chrome",
		RealityPublicKey:  "PBK",
		RealityPrivateKey: "PRIVATE-KEY-HEX",
		RealityShortID:    "sid",
		RealityServerName: "www.cloudflare.com",
		RealityDest:       "www.cloudflare.com:443",
	}
	users := []store.NodeUser{{UserID: 1, ClientID: "uuid-1", Enabled: 1}}
	body, _, err := runtime.Build(node, users, "v1")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	main := doc["inbounds"].([]any)[0].(map[string]any)
	stream := main["streamSettings"].(map[string]any)
	r := stream["realitySettings"].(map[string]any)
	if r["privateKey"] != "PRIVATE-KEY-HEX" {
		t.Errorf("realitySettings.privateKey = %v, want PRIVATE-KEY-HEX", r["privateKey"])
	}
	if r["dest"] != "www.cloudflare.com:443" {
		t.Errorf("realitySettings.dest = %v, want www.cloudflare.com:443", r["dest"])
	}
	serverNames, _ := r["serverNames"].([]any)
	if len(serverNames) != 1 || serverNames[0] != "www.cloudflare.com" {
		t.Errorf("realitySettings.serverNames = %#v, want [www.cloudflare.com]", r["serverNames"])
	}
	if _, leak := r["serverName"]; leak {
		t.Errorf("xray reality inbound must not emit outbound-only serverName: %#v", r)
	}
	shortIDs, _ := r["shortIds"].([]any)
	if len(shortIDs) != 1 || shortIDs[0] != "sid" {
		t.Errorf("realitySettings.shortIds = %#v, want [sid]", r["shortIds"])
	}
}

func TestXrayRealityEmptyShortIDStillEmitsShortIds(t *testing.T) {
	node := &store.Node{
		ID: 1, Name: "JP-R", Host: "jp.example.com", Port: 443,
		Protocol: "vless", Transport: "tcp", Security: "reality",
		RuntimeType:       "xray",
		RealityPublicKey:  "PBK",
		RealityPrivateKey: "PRIVATE-KEY-HEX",
		RealityServerName: "www.cloudflare.com",
		RealityDest:       "www.cloudflare.com:443",
	}
	users := []store.NodeUser{{UserID: 1, ClientID: "uuid-1", Enabled: 1}}
	body, _, err := runtime.Build(node, users, "v1")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	main := doc["inbounds"].([]any)[0].(map[string]any)
	stream := main["streamSettings"].(map[string]any)
	r := stream["realitySettings"].(map[string]any)
	shortIDs, _ := r["shortIds"].([]any)
	if len(shortIDs) != 1 || shortIDs[0] != "" {
		t.Fatalf("empty short id must still be emitted as shortIds=[\"\"], got %#v", r["shortIds"])
	}
}

func TestBuildRejectsIncompleteRealityNode(t *testing.T) {
	base := store.Node{
		ID: 1, Name: "JP-R", Host: "jp.example.com", Port: 443,
		Protocol: "vless", Transport: "tcp", Security: "reality",
		RuntimeType:       "xray",
		RealityPublicKey:  "PBK",
		RealityPrivateKey: "PRIVATE-KEY-HEX",
		RealityServerName: "www.cloudflare.com",
	}
	users := []store.NodeUser{{UserID: 1, ClientID: "uuid-1", Enabled: 1}}
	cases := []struct {
		name    string
		mutate  func(*store.Node)
		wantErr string
	}{
		{
			name: "missing server name",
			mutate: func(n *store.Node) {
				n.RealityServerName = ""
			},
			wantErr: "reality_server_name",
		},
		{
			name: "missing public key",
			mutate: func(n *store.Node) {
				n.RealityPublicKey = ""
			},
			wantErr: "reality_public_key",
		},
		{
			name: "missing private key",
			mutate: func(n *store.Node) {
				n.RealityPrivateKey = ""
			},
			wantErr: "reality_private_key",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			node := base
			tc.mutate(&node)
			_, _, err := runtime.Build(&node, users, "v1")
			if err == nil {
				t.Fatalf("expected Build to reject incomplete reality node")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %q, want it to mention %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// Stage 13: SS-2022 — method on inbound, no `decryption` field.
func TestXrayShadowsocks2022(t *testing.T) {
	node := &store.Node{
		ID: 1, Name: "SS22", Host: "ss.example.com", Port: 8443,
		Protocol: "ss", Transport: "tcp", Security: "none",
		RuntimeType: "xray",
		SSMethod:    "2022-blake3-aes-128-gcm",
	}
	users := []store.NodeUser{{UserID: 5, ClientID: "Y2hhcjE2Y2hhcjE2Y2hhcjE2Y2hhcg==", Enabled: 1}}
	body, _, err := runtime.Build(node, users, "v1")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	main := doc["inbounds"].([]any)[0].(map[string]any)
	settings := main["settings"].(map[string]any)
	if settings["method"] != "2022-blake3-aes-128-gcm" {
		t.Errorf("settings.method = %v, want 2022-blake3-aes-128-gcm", settings["method"])
	}
	if _, leak := settings["decryption"]; leak {
		t.Errorf("ss inbound must not have a `decryption` field")
	}
}

// Stage 14: Hysteria2 inbound shape — type, users, tls.alpn, up/down, obfs.
func TestHysteria2Inbound(t *testing.T) {
	node := &store.Node{
		ID: 1, Name: "HY2", Host: "hy.example.com", Port: 443,
		Protocol: "hysteria2", Transport: "udp", Security: "tls",
		RuntimeType:       "sing-box",
		ObfsPassword:      "salty",
		CongestionControl: "bbr",
		UpMbps:            100,
		DownMbps:          200,
		SNI:               "hy.example.com",
	}
	users := []store.NodeUser{
		{UserID: 7, ClientID: "secret-7", Enabled: 1},
		{UserID: 8, ClientID: "secret-8", Enabled: 0}, // disabled excluded
	}
	body, _, err := runtime.Build(node, users, "v1")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	in := doc["inbounds"].([]any)[0].(map[string]any)
	if in["type"] != "hysteria2" {
		t.Errorf("inbound type = %v", in["type"])
	}
	us := in["users"].([]any)
	if len(us) != 1 {
		t.Fatalf("expected 1 user (disabled excluded), got %d", len(us))
	}
	u0 := us[0].(map[string]any)
	if u0["password"] != "secret-7" || u0["name"] != "u7" {
		t.Errorf("user[0] = %#v", u0)
	}
	if in["up_mbps"] != float64(100) || in["down_mbps"] != float64(200) {
		t.Errorf("up/down = %v/%v", in["up_mbps"], in["down_mbps"])
	}
	obfs, _ := in["obfs"].(map[string]any)
	if obfs == nil || obfs["type"] != "salamander" || obfs["password"] != "salty" {
		t.Errorf("obfs = %#v", obfs)
	}
	tls := in["tls"].(map[string]any)
	if tls["server_name"] != "hy.example.com" {
		t.Errorf("tls.server_name = %v", tls["server_name"])
	}
	if tls["certificate_path"] != "/etc/zboard-agent/tls/server.crt" {
		t.Errorf("tls.certificate_path = %v", tls["certificate_path"])
	}
	if tls["key_path"] != "/etc/zboard-agent/tls/server.key" {
		t.Errorf("tls.key_path = %v", tls["key_path"])
	}
	alpn := tls["alpn"].([]any)
	if len(alpn) != 1 || alpn[0] != "h3" {
		t.Errorf("expected default alpn=[h3], got %#v", alpn)
	}
	exp := doc["experimental"].(map[string]any)
	api := exp["v2ray_api"].(map[string]any)
	if api["listen"] != "127.0.0.1:10085" {
		t.Fatalf("v2ray_api.listen = %v", api["listen"])
	}
	stats := api["stats"].(map[string]any)
	if stats["enabled"] != true {
		t.Fatalf("v2ray_api.stats.enabled = %v", stats["enabled"])
	}
	statUsers := stats["users"].([]any)
	if len(statUsers) != 1 || statUsers[0] != "u7" {
		t.Fatalf("v2ray_api.stats.users = %#v, want [u7]", statUsers)
	}
}

func TestHysteria2RejectsInvalidPortRange(t *testing.T) {
	node := &store.Node{
		ID: 1, Name: "HY2", Host: "hy.example.com", Port: 443,
		Protocol: "hysteria2", Transport: "udp", Security: "tls",
		RuntimeType: "sing-box", PortRange: "40000-20000",
	}
	_, _, err := runtime.Build(node, []store.NodeUser{{UserID: 7, ClientID: "secret-7", Enabled: 1}}, "v1")
	if err == nil {
		t.Fatalf("expected invalid port_range to be rejected")
	}
	if !strings.Contains(err.Error(), "port_range") {
		t.Fatalf("error = %q, want port_range", err.Error())
	}
}

// Stage 14: TUIC inbound shape — uuid+password per user, congestion_control,
// QUIC TLS with default ALPN h3.
func TestTUICInbound(t *testing.T) {
	node := &store.Node{
		ID: 1, Name: "TUIC", Host: "tuic.example.com", Port: 443,
		Protocol: "tuic", Transport: "udp", Security: "tls",
		RuntimeType:       "sing-box",
		CongestionControl: "cubic",
		ObfsPassword:      "shared-pwd",
		SNI:               "tuic.example.com",
	}
	users := []store.NodeUser{{UserID: 9, ClientID: "uuid-9", Enabled: 1}}
	body, _, err := runtime.Build(node, users, "v1")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	in := doc["inbounds"].([]any)[0].(map[string]any)
	if in["type"] != "tuic" {
		t.Errorf("inbound type = %v", in["type"])
	}
	if in["congestion_control"] != "cubic" {
		t.Errorf("congestion_control = %v", in["congestion_control"])
	}
	u0 := in["users"].([]any)[0].(map[string]any)
	if u0["uuid"] != "uuid-9" || u0["password"] != "shared-pwd" {
		t.Errorf("user[0] = %#v", u0)
	}
	tls := in["tls"].(map[string]any)
	if tls["certificate_path"] != "/etc/zboard-agent/tls/server.crt" {
		t.Errorf("tls.certificate_path = %v", tls["certificate_path"])
	}
	if tls["key_path"] != "/etc/zboard-agent/tls/server.key" {
		t.Errorf("tls.key_path = %v", tls["key_path"])
	}
	if alpn := tls["alpn"].([]any); len(alpn) != 1 || alpn[0] != "h3" {
		t.Errorf("alpn = %#v", alpn)
	}
}

// Stage 13: sing-box SS-2022 + Reality with private_key + handshake.server.
func TestSingBoxSS2022AndRealityHandshake(t *testing.T) {
	t.Run("ss-2022", func(t *testing.T) {
		node := &store.Node{
			ID: 1, Name: "SS22-SB", Host: "ss.example.com", Port: 8443,
			Protocol: "ss", Transport: "tcp", Security: "none",
			RuntimeType: "sing-box",
			SSMethod:    "2022-blake3-aes-128-gcm",
		}
		users := []store.NodeUser{{UserID: 5, ClientID: "pwd-5", Enabled: 1}}
		body, _, err := runtime.Build(node, users, "v1")
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		var doc map[string]any
		if err := json.Unmarshal([]byte(body), &doc); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		in := doc["inbounds"].([]any)[0].(map[string]any)
		if in["type"] != "shadowsocks" {
			t.Errorf("inbound type = %v, want shadowsocks", in["type"])
		}
		if in["method"] != "2022-blake3-aes-128-gcm" {
			t.Errorf("inbound method = %v", in["method"])
		}
	})
	t.Run("reality-handshake", func(t *testing.T) {
		node := &store.Node{
			ID: 1, Name: "JP-R-SB", Host: "jp.example.com", Port: 443,
			Protocol: "vless", Transport: "tcp", Security: "reality",
			RuntimeType:       "sing-box",
			Flow:              "xtls-rprx-vision",
			RealityPublicKey:  "PBK",
			RealityPrivateKey: "PRIV",
			RealityShortID:    "sid",
			RealityServerName: "www.cloudflare.com",
			RealityDest:       "www.cloudflare.com:443",
		}
		users := []store.NodeUser{{UserID: 1, ClientID: "uuid-1", Enabled: 1}}
		body, _, err := runtime.Build(node, users, "v1")
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		var doc map[string]any
		if err := json.Unmarshal([]byte(body), &doc); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		in := doc["inbounds"].([]any)[0].(map[string]any)
		// vless flow flows down to user entry
		us := in["users"].([]any)[0].(map[string]any)
		if us["flow"] != "xtls-rprx-vision" {
			t.Errorf("user.flow = %v", us["flow"])
		}
		tls := in["tls"].(map[string]any)
		r := tls["reality"].(map[string]any)
		if r["private_key"] != "PRIV" {
			t.Errorf("reality.private_key = %v", r["private_key"])
		}
		hs, _ := r["handshake"].(map[string]any)
		if hs == nil || hs["server"] != "www.cloudflare.com:443" {
			t.Errorf("reality.handshake = %#v", hs)
		}
	})
}

func TestValidateNodeRejectsOutOfRangePort(t *testing.T) {
	base := store.Node{
		Name: "P", Host: "p.example.com", Protocol: "vless",
		Transport: "tcp", Security: "tls", RuntimeType: "xray",
	}
	cases := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"zero", 0, true},
		{"negative", -1, true},
		{"too high", 65536, true},
		{"way too high", 1 << 20, true},
		{"min valid", 1, false},
		{"typical", 443, false},
		{"max valid", 65535, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := base
			n.Port = tc.port
			err := runtime.ValidateNode(&n)
			if tc.wantErr && err == nil {
				t.Fatalf("port %d: expected error, got nil", tc.port)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("port %d: unexpected error: %v", tc.port, err)
			}
		})
	}
}
