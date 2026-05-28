package subrender_test

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/zboard/api-server/internal/store"
	"github.com/zboard/api-server/internal/subrender"
)

func sptr(s string) *string { return &s }

func sampleNodes() []store.Node {
	return []store.Node{
		{
			ID: 1, NodeCode: "n1", Name: "HK-1", Region: sptr("HK"),
			Host: "hk.example.com", Port: 443,
			Protocol: "vless", Transport: "tcp", Security: "tls",
			SNI: "hk.example.com", Status: "active",
		},
		{
			ID: 2, NodeCode: "n2", Name: "JP-Reality", Region: sptr("JP"),
			Host: "jp.example.com", Port: 443,
			Protocol: "vless", Transport: "tcp", Security: "reality",
			Fingerprint: "chrome", RealityPublicKey: "PBK", RealityShortID: "sid01",
			RealityPrivateKey: "PRIVATE-KEY-MUST-NOT-LEAK",
			RealityServerName: "www.cloudflare.com", Status: "active",
		},
		{
			ID: 3, NodeCode: "n3", Name: "US-WS", Region: sptr("US"),
			Host: "us.example.com", Port: 443,
			Protocol: "vless", Transport: "ws", Security: "tls",
			WSPath: "/api/v1", WSHost: "cdn.example.com",
			Fingerprint: "firefox", SNI: "cdn.example.com", Status: "active",
		},
	}
}

func sampleNodeUsers() []store.NodeUser {
	const cid = "00000000-0000-4000-8000-000000000001"
	return []store.NodeUser{
		{NodeID: 1, UserID: 9, ClientID: cid, Protocol: "vless", Enabled: 1},
		{NodeID: 2, UserID: 9, ClientID: cid, Protocol: "vless", Enabled: 1},
		{NodeID: 3, UserID: 9, ClientID: cid, Protocol: "vless", Enabled: 1},
	}
}

func TestBuildIncludesEnabledOnly(t *testing.T) {
	nu := sampleNodeUsers()
	nu[1].Enabled = 0
	items := subrender.Build(sampleNodes(), nu)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestClashMetaContainsKeyFields(t *testing.T) {
	out := subrender.ClashMeta(subrender.Build(sampleNodes(), sampleNodeUsers()))
	for _, want := range []string{
		"reality-opts:",
		"public-key: PBK",
		"short-id: sid01",
		"client-fingerprint: chrome",
		"path: /api/v1",
		"Host: cdn.example.com",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("clash output missing %q\n---\n%s", want, out)
		}
	}
}

func TestSingBoxJSONShape(t *testing.T) {
	out := subrender.SingBox(subrender.Build(sampleNodes(), sampleNodeUsers()))
	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, out)
	}
	outs, ok := doc["outbounds"].([]any)
	if !ok || len(outs) != 3 {
		t.Fatalf("expected 3 outbounds, got %#v", doc["outbounds"])
	}

	// Find the reality entry by tag suffix.
	var reality map[string]any
	for _, o := range outs {
		m := o.(map[string]any)
		if tag, _ := m["tag"].(string); strings.Contains(tag, "JP-Reality") {
			reality = m
			break
		}
	}
	if reality == nil {
		t.Fatal("reality outbound not found")
	}
	tls, ok := reality["tls"].(map[string]any)
	if !ok {
		t.Fatalf("expected tls map: %#v", reality)
	}
	if r, _ := tls["reality"].(map[string]any); r == nil {
		t.Fatalf("expected tls.reality: %#v", tls)
	}
}

func TestBase64URIRoundTrip(t *testing.T) {
	out := subrender.Base64(subrender.Build(sampleNodes(), sampleNodeUsers()))
	raw, err := base64.StdEncoding.DecodeString(out)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	uri := string(raw)
	for _, want := range []string{
		"vless://", "security=reality", "pbk=PBK", "sid=sid01",
		"path=%2Fapi%2Fv1", "fp=firefox",
	} {
		if !strings.Contains(uri, want) {
			t.Errorf("base64 URIs missing %q\n%s", want, uri)
		}
	}
}

func TestBase64URIIncludesModernTransportOptions(t *testing.T) {
	const cid = "00000000-0000-4000-8000-000000000001"
	nodes := []store.Node{
		{
			ID: 1, NodeCode: "xhttp", Name: "XHTTP", Host: "xhttp.example.com", Port: 443,
			Protocol: "vless", Transport: "xhttp", Security: "tls",
			WSPath: "/edge", WSHost: "cdn.example.com", Status: "active",
		},
		{
			ID: 2, NodeCode: "hup", Name: "HTTPUpgrade", Host: "hup.example.com", Port: 443,
			Protocol: "vless", Transport: "httpupgrade", Security: "tls",
			WSPath: "/upgrade", WSHost: "cdn2.example.com", Status: "active",
		},
	}
	users := []store.NodeUser{
		{NodeID: 1, UserID: 1, ClientID: cid, Protocol: "vless", Enabled: 1},
		{NodeID: 2, UserID: 1, ClientID: cid, Protocol: "vless", Enabled: 1},
	}
	raw, err := base64.StdEncoding.DecodeString(subrender.Base64(subrender.Build(nodes, users)))
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	uri := string(raw)
	for _, want := range []string{
		"type=xhttp", "path=%2Fedge", "host=cdn.example.com",
		"type=httpupgrade", "path=%2Fupgrade", "host=cdn2.example.com",
	} {
		if !strings.Contains(uri, want) {
			t.Errorf("URI list missing %q\n%s", want, uri)
		}
	}
}

func TestBuildSkipsIncompleteRealityNodes(t *testing.T) {
	nodes := []store.Node{
		{
			ID: 1, NodeCode: "bad-reality", Name: "Bad Reality",
			Host: "bad.example.com", Port: 443,
			Protocol: "vless", Transport: "tcp", Security: "reality",
			RealityServerName: "www.cloudflare.com",
			RealityPublicKey:  "",
			RealityPrivateKey: "PRIVATE-KEY-HEX",
			Status:            "active",
		},
		{
			ID: 2, NodeCode: "ok-tls", Name: "OK TLS",
			Host: "ok.example.com", Port: 443,
			Protocol: "vless", Transport: "tcp", Security: "tls",
			Status: "active",
		},
	}
	users := []store.NodeUser{
		{NodeID: 1, UserID: 1, ClientID: "uuid-1", Protocol: "vless", Enabled: 1},
		{NodeID: 2, UserID: 1, ClientID: "uuid-2", Protocol: "vless", Enabled: 1},
	}
	items := subrender.Build(nodes, users)
	if len(items) != 1 {
		t.Fatalf("expected invalid reality node to be skipped, got %d items: %#v", len(items), items)
	}
	if items[0].NodeID != 2 {
		t.Fatalf("unexpected item after skipping invalid reality node: %#v", items[0])
	}
	raw, err := base64.StdEncoding.DecodeString(subrender.Base64(items))
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	if strings.Contains(string(raw), "bad.example.com") {
		t.Fatalf("invalid reality node leaked into subscription:\n%s", raw)
	}
}

func TestDedupNamesAppendsSuffix(t *testing.T) {
	nodes := []store.Node{
		{ID: 1, Name: "Same", Host: "a", Port: 1, Protocol: "vless", Transport: "tcp", Security: "tls"},
		{ID: 2, Name: "Same", Host: "b", Port: 2, Protocol: "vless", Transport: "tcp", Security: "tls"},
	}
	const cid = "00000000-0000-4000-8000-000000000001"
	users := []store.NodeUser{
		{NodeID: 1, UserID: 1, ClientID: cid, Protocol: "vless", Enabled: 1},
		{NodeID: 2, UserID: 1, ClientID: cid, Protocol: "vless", Enabled: 1},
	}
	items := subrender.Build(nodes, users)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Name == items[1].Name {
		t.Fatalf("expected dedup, both still %q", items[0].Name)
	}
}

// Stage 13: VLESS+Vision flow + ALPN must propagate through every renderer.
func stage13Sample() ([]store.Node, []store.NodeUser) {
	nodes := []store.Node{
		{
			ID: 10, NodeCode: "vis", Name: "VIS-Reality", Region: sptr("HK"),
			Host:              "hk.example.com",
			Port:              443,
			Protocol:          "vless",
			Transport:         "tcp",
			Security:          "reality",
			Status:            "active",
			Flow:              "xtls-rprx-vision",
			ALPN:              "h2,http/1.1",
			Fingerprint:       "chrome",
			RealityPublicKey:  "PBK-PUBLIC",
			RealityPrivateKey: "PRIVATE-KEY-MUST-NOT-LEAK", // canary
			RealityShortID:    "sid01",
			RealityServerName: "www.cloudflare.com",
			RealityDest:       "www.cloudflare.com:443",
		},
		{
			ID: 11, NodeCode: "ss22", Name: "SS-2022", Region: sptr("JP"),
			Host: "ss.example.com", Port: 8443,
			Protocol: "ss", Transport: "tcp", Security: "none",
			Status:   "active",
			SSMethod: "2022-blake3-aes-128-gcm",
		},
	}
	users := []store.NodeUser{
		{NodeID: 10, UserID: 7, ClientID: "uuid-7", Protocol: "vless", Enabled: 1},
		{NodeID: 11, UserID: 7, ClientID: "ss-pwd-7", Protocol: "ss", Enabled: 1},
	}
	return nodes, users
}

func TestStage13ClashMetaFlowAlpnSS2022(t *testing.T) {
	nodes, users := stage13Sample()
	out := subrender.ClashMeta(subrender.Build(nodes, users))
	for _, want := range []string{
		"flow: xtls-rprx-vision",
		"alpn:",
		"- h2",
		"- http/1.1",
		"cipher: 2022-blake3-aes-128-gcm",
		"reality-opts:",
		"public-key: PBK-PUBLIC",
		"short-id: sid01",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("clash output missing %q\n---\n%s", want, out)
		}
	}
	if strings.Contains(out, "PRIVATE-KEY-MUST-NOT-LEAK") {
		t.Fatalf("reality private key LEAKED into Clash output:\n%s", out)
	}
}

func TestStage13SingBoxFlowAlpnSS2022(t *testing.T) {
	nodes, users := stage13Sample()
	out := subrender.SingBox(subrender.Build(nodes, users))
	if strings.Contains(out, "PRIVATE-KEY-MUST-NOT-LEAK") {
		t.Fatalf("reality private key LEAKED into sing-box output:\n%s", out)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, out)
	}
	outs := doc["outbounds"].([]any)
	if len(outs) != 2 {
		t.Fatalf("expected 2 outbounds, got %d", len(outs))
	}

	// First outbound = vless reality with vision flow + ALPN
	vis, _ := outs[0].(map[string]any)
	if vis["flow"] != "xtls-rprx-vision" {
		t.Errorf("vless flow = %v, want xtls-rprx-vision", vis["flow"])
	}
	tls, _ := vis["tls"].(map[string]any)
	if tls == nil {
		t.Fatalf("expected tls block")
	}
	alpn, _ := tls["alpn"].([]any)
	if len(alpn) != 2 || alpn[0] != "h2" || alpn[1] != "http/1.1" {
		t.Errorf("alpn = %#v", alpn)
	}

	// Second outbound = SS-2022
	ss, _ := outs[1].(map[string]any)
	if ss["type"] != "shadowsocks" || ss["method"] != "2022-blake3-aes-128-gcm" {
		t.Errorf("ss outbound = %#v", ss)
	}
}

func TestStage13Base64URIIncludesFlowAlpnSS2022(t *testing.T) {
	nodes, users := stage13Sample()
	out := subrender.Base64(subrender.Build(nodes, users))
	if strings.Contains(out, "PRIVATE-KEY-MUST-NOT-LEAK") {
		t.Fatal("reality private key LEAKED into base64 output")
	}
	raw, err := base64.StdEncoding.DecodeString(out)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	uri := string(raw)
	if strings.Contains(uri, "PRIVATE-KEY-MUST-NOT-LEAK") {
		t.Fatalf("reality private key LEAKED into URI list:\n%s", uri)
	}
	for _, want := range []string{
		"vless://", "security=reality", "flow=xtls-rprx-vision",
		"pbk=PBK-PUBLIC", "sid=sid01", "alpn=h2",
		"ss://",
	} {
		if !strings.Contains(uri, want) {
			t.Errorf("URI list missing %q\n%s", want, uri)
		}
	}
	// Decode the SS userinfo and confirm SS-2022 cipher.
	for _, line := range strings.Split(uri, "\n") {
		if !strings.HasPrefix(line, "ss://") {
			continue
		}
		// ss://<userinfo>@host:port#name — userinfo is RawURLEncoded "method:password"
		body := strings.TrimPrefix(line, "ss://")
		at := strings.Index(body, "@")
		if at < 0 {
			t.Fatalf("ss URI missing @: %s", line)
		}
		decoded, err := base64.RawURLEncoding.DecodeString(body[:at])
		if err != nil {
			t.Fatalf("decode ss userinfo: %v", err)
		}
		if !strings.HasPrefix(string(decoded), "2022-blake3-aes-128-gcm:") {
			t.Errorf("ss userinfo missing SS-2022 cipher prefix: %q", decoded)
		}
	}
}

// Stage 14: Hysteria2 + TUIC subscription rendering.
func stage14Sample() ([]store.Node, []store.NodeUser) {
	nodes := []store.Node{
		{
			ID: 20, NodeCode: "hy2", Name: "HY2", Region: sptr("HK"),
			Host: "hy.example.com", Port: 443,
			Protocol: "hysteria2", Transport: "udp", Security: "tls",
			Status:            "active",
			SNI:               "hy.example.com",
			ObfsPassword:      "salty-pwd",
			CongestionControl: "bbr",
			UpMbps:            100,
			DownMbps:          200,
			TLSInsecure:       1,
		},
		{
			ID: 21, NodeCode: "tuic", Name: "TUIC", Region: sptr("JP"),
			Host: "tuic.example.com", Port: 443,
			Protocol: "tuic", Transport: "udp", Security: "tls",
			Status:            "active",
			SNI:               "tuic.example.com",
			CongestionControl: "cubic",
			ObfsPassword:      "shared-tuic-pwd",
			TLSInsecure:       1,
		},
	}
	users := []store.NodeUser{
		{NodeID: 20, UserID: 1, ClientID: "hy2-secret", Protocol: "hysteria2", Enabled: 1},
		{NodeID: 21, UserID: 1, ClientID: "tuic-uuid", Protocol: "tuic", Enabled: 1},
	}
	return nodes, users
}

func TestStage14ClashHysteria2TUIC(t *testing.T) {
	nodes, users := stage14Sample()
	out := subrender.ClashMeta(subrender.Build(nodes, users))
	for _, want := range []string{
		"type: hysteria2",
		"password: hy2-secret",
		"obfs: salamander",
		"obfs-password: salty-pwd",
		"up: 100",
		"down: 200",
		"skip-cert-verify: true",
		"type: tuic",
		"uuid: tuic-uuid",
		"password: shared-tuic-pwd",
		"congestion-controller: cubic",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("clash output missing %q\n---\n%s", want, out)
		}
	}
}

func TestStage14SingBoxHysteria2TUIC(t *testing.T) {
	nodes, users := stage14Sample()
	out := subrender.SingBox(subrender.Build(nodes, users))
	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, out)
	}
	outs := doc["outbounds"].([]any)
	if len(outs) != 2 {
		t.Fatalf("expected 2 outbounds, got %d", len(outs))
	}
	hy2 := outs[0].(map[string]any)
	if hy2["type"] != "hysteria2" {
		t.Errorf("hy2 type = %v", hy2["type"])
	}
	if hy2["up_mbps"] != float64(100) || hy2["down_mbps"] != float64(200) {
		t.Errorf("hy2 up/down = %v/%v", hy2["up_mbps"], hy2["down_mbps"])
	}
	hy2TLS := hy2["tls"].(map[string]any)
	if hy2TLS["insecure"] != true {
		t.Errorf("hy2 tls.insecure = %v, want true", hy2TLS["insecure"])
	}
	obfs, _ := hy2["obfs"].(map[string]any)
	if obfs == nil || obfs["type"] != "salamander" || obfs["password"] != "salty-pwd" {
		t.Errorf("hy2 obfs = %#v", obfs)
	}
	tuic := outs[1].(map[string]any)
	if tuic["type"] != "tuic" || tuic["uuid"] != "tuic-uuid" || tuic["password"] != "shared-tuic-pwd" {
		t.Errorf("tuic = %#v", tuic)
	}
	if tuic["congestion_control"] != "cubic" {
		t.Errorf("tuic congestion_control = %v", tuic["congestion_control"])
	}
	tuicTLS := tuic["tls"].(map[string]any)
	if tuicTLS["insecure"] != true {
		t.Errorf("tuic tls.insecure = %v, want true", tuicTLS["insecure"])
	}
}

func TestStage14Base64Hy2TuicURIs(t *testing.T) {
	nodes, users := stage14Sample()
	out := subrender.Base64(subrender.Build(nodes, users))
	raw, err := base64.StdEncoding.DecodeString(out)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	uri := string(raw)
	for _, want := range []string{
		"hysteria2://",
		"hy2-secret",
		"obfs=salamander",
		"obfs-password=salty-pwd",
		"up=100",
		"down=200",
		"insecure=1",
		"tuic://",
		"tuic-uuid:shared-tuic-pwd",
		"congestion_control=cubic",
		"alpn=h3",
	} {
		if !strings.Contains(uri, want) {
			t.Errorf("URI list missing %q\n%s", want, uri)
		}
	}
}

func TestHysteria2Base64URIKeepsNumericAuthorityPortWhenPortRangeSet(t *testing.T) {
	nodes, users := stage14Sample()
	nodes[0].Port = 20925
	nodes[0].PortRange = "21000-22000"
	nodes[0].Fingerprint = "chrome"
	out := subrender.Base64(subrender.Build(nodes[:1], users[:1]))
	raw, err := base64.StdEncoding.DecodeString(out)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	line := strings.TrimSpace(string(raw))
	u, err := url.Parse(line)
	if err != nil {
		t.Fatalf("hysteria2 URI should be parseable, got err=%v uri=%s", err, line)
	}
	if u.Scheme != "hysteria2" {
		t.Fatalf("hysteria2 URI scheme=%q, want hysteria2: %s", u.Scheme, line)
	}
	if u.Port() != "20925" {
		t.Fatalf("hysteria2 URI authority port=%q, want listen port 20925: %s", u.Port(), line)
	}
	if got := u.Query().Get("mport"); got != "21000-22000" {
		t.Fatalf("hysteria2 URI mport=%q, want 21000-22000: %s", got, line)
	}
	if got := u.Query().Get("fp"); got != "" {
		t.Fatalf("hysteria2 URI should not include uTLS fingerprint, got fp=%q: %s", got, line)
	}
	if got := u.Query().Get("insecure"); got != "1" {
		t.Fatalf("hysteria2 URI insecure=%q, want 1 for self-signed agent certificate: %s", got, line)
	}
}

func TestSingBoxHysteria2DoesNotRenderUTLS(t *testing.T) {
	nodes, users := stage14Sample()
	nodes[0].Fingerprint = "chrome"
	out := subrender.SingBox(subrender.Build(nodes[:1], users[:1]))
	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("sing-box decode: %v\n%s", err, out)
	}
	outbounds := doc["outbounds"].([]any)
	hy2 := outbounds[0].(map[string]any)
	tls, ok := hy2["tls"].(map[string]any)
	if !ok {
		t.Fatalf("hysteria2 outbound missing tls: %#v", hy2)
	}
	if _, ok := tls["utls"]; ok {
		t.Fatalf("hysteria2 outbound should not include tls.utls: %#v", tls)
	}
}
