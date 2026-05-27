package runtime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInferRuntimeType(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
		ok   bool
	}{
		{
			name: "sing-box",
			body: `{"inbounds":[{"type":"hysteria2","tag":"in"}]}`,
			want: "sing-box",
			ok:   true,
		},
		{
			name: "xray",
			body: `{"inbounds":[{"protocol":"vless","tag":"in"}]}`,
			want: "xray",
			ok:   true,
		},
		{
			name: "unknown",
			body: `{"inbounds":[{"tag":"in"}]}`,
			want: "",
			ok:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := inferRuntimeType([]byte(tt.body))
			if ok != tt.ok || got != tt.want {
				t.Fatalf("inferRuntimeType()=(%q,%t), want (%q,%t)", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestRuntimeBinaryForType(t *testing.T) {
	tests := []struct {
		name        string
		current     string
		runtimeType string
		want        string
	}{
		{
			name:        "xray to sing-box",
			current:     "/usr/local/bin/xray",
			runtimeType: "sing-box",
			want:        "/usr/local/bin/sing-box",
		},
		{
			name:        "sing-box to xray",
			current:     "/usr/local/bin/sing-box",
			runtimeType: "xray",
			want:        "/usr/local/bin/xray",
		},
		{
			name:        "custom binary unchanged",
			current:     "/opt/runtime/custom",
			runtimeType: "sing-box",
			want:        "/opt/runtime/custom",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runtimeBinaryForType(tt.current, tt.runtimeType)
			if got != tt.want {
				t.Fatalf("runtimeBinaryForType()=%q, want %q", got, tt.want)
			}
		})
	}
}

func TestEnsureDefaultTLSCertificate(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "server.crt")
	keyPath := filepath.Join(dir, "server.key")

	configJSON := []byte(`{
		"inbounds": [{
			"type": "hysteria2",
			"tls": {
				"enabled": true,
				"server_name": "hy.example.com",
				"certificate_path": "` + filepath.ToSlash(certPath) + `",
				"key_path": "` + filepath.ToSlash(keyPath) + `"
			}
		}]
	}`)
	if err := ensureRuntimeAssets(configJSON); err != nil {
		t.Fatalf("ensureRuntimeAssets: %v", err)
	}
	if _, err := os.Stat(certPath); err != nil {
		t.Fatalf("certificate was not created: %v", err)
	}
	if _, err := os.Stat(keyPath); err != nil {
		t.Fatalf("private key was not created: %v", err)
	}
}

func TestPrepareRuntimeConfigAddsDefaultQUICCertificatePaths(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "server.crt")
	keyPath := filepath.Join(dir, "server.key")

	t.Setenv("ZBOARD_QUIC_CERT_PATH", certPath)
	t.Setenv("ZBOARD_QUIC_KEY_PATH", keyPath)

	prepared, err := prepareRuntimeConfig([]byte(`{
		"inbounds": [{
			"type": "hysteria2",
			"tls": {
				"enabled": true,
				"server_name": "us1.example.com"
			}
		}]
	}`))
	if err != nil {
		t.Fatalf("prepareRuntimeConfig: %v", err)
	}

	var doc struct {
		Inbounds []struct {
			TLS struct {
				CertificatePath string `json:"certificate_path"`
				KeyPath         string `json:"key_path"`
			} `json:"tls"`
		} `json:"inbounds"`
	}
	if err := json.Unmarshal(prepared, &doc); err != nil {
		t.Fatalf("unmarshal prepared config: %v", err)
	}
	if got := doc.Inbounds[0].TLS.CertificatePath; got != certPath {
		t.Fatalf("certificate_path = %q, want %q", got, certPath)
	}
	if got := doc.Inbounds[0].TLS.KeyPath; got != keyPath {
		t.Fatalf("key_path = %q, want %q", got, keyPath)
	}
	if _, err := os.Stat(certPath); err != nil {
		t.Fatalf("certificate was not created: %v", err)
	}
	if _, err := os.Stat(keyPath); err != nil {
		t.Fatalf("private key was not created: %v", err)
	}
}

func TestPrepareRuntimeConfigStripsUnsupportedSingBoxV2RayAPI(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "server.crt")
	keyPath := filepath.Join(dir, "server.key")

	prepared, err := prepareRuntimeConfig([]byte(`{
		"inbounds": [{
			"type": "hysteria2",
			"tls": {
				"enabled": true,
				"server_name": "us1.example.com",
				"certificate_path": "` + filepath.ToSlash(certPath) + `",
				"key_path": "` + filepath.ToSlash(keyPath) + `"
			}
		}],
		"experimental": {
			"cache_file": {"enabled": true},
			"v2ray_api": {
				"listen": "127.0.0.1:10085",
				"stats": {"enabled": true, "users": ["u1"]}
			}
		}
	}`))
	if err != nil {
		t.Fatalf("prepareRuntimeConfig: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(prepared, &doc); err != nil {
		t.Fatalf("unmarshal prepared config: %v", err)
	}
	exp, _ := doc["experimental"].(map[string]any)
	if _, ok := exp["v2ray_api"]; ok {
		t.Fatalf("v2ray_api should be stripped for bundled sing-box compatibility: %#v", exp)
	}
	if _, ok := exp["cache_file"]; !ok {
		t.Fatalf("non-v2ray experimental settings should be preserved: %#v", exp)
	}
}

func TestPrepareRuntimeConfigStripsLegacySpecialBlockOutbound(t *testing.T) {
	prepared, err := prepareRuntimeConfig([]byte(`{
		"inbounds": [{"type": "hysteria2"}],
		"outbounds": [
			{"type": "direct", "tag": "direct"},
			{"type": "block", "tag": "block"}
		]
	}`))
	if err != nil {
		t.Fatalf("prepareRuntimeConfig: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(prepared, &doc); err != nil {
		t.Fatalf("unmarshal prepared config: %v", err)
	}
	outs, _ := doc["outbounds"].([]any)
	if len(outs) != 1 {
		t.Fatalf("expected only direct outbound after cleanup, got %#v", outs)
	}
	out, _ := outs[0].(map[string]any)
	if out["type"] != "direct" {
		t.Fatalf("expected direct outbound to remain, got %#v", out)
	}
}

func TestTryBootExistingRewritesLegacyQUICConfig(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "tls", "server.crt")
	keyPath := filepath.Join(dir, "tls", "server.key")
	configPath := filepath.Join(dir, "runtime.json")

	t.Setenv("ZBOARD_QUIC_CERT_PATH", certPath)
	t.Setenv("ZBOARD_QUIC_KEY_PATH", keyPath)

	if err := os.WriteFile(configPath, []byte(`{
		"inbounds": [{
			"type": "hysteria2",
			"tls": {
				"enabled": true,
				"server_name": "us1.example.com"
			}
		}]
	}`), 0o600); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}

	s := New(filepath.Join(dir, "missing-sing-box"), "sing-box", configPath, dir)
	if s.TryBootExisting(context.Background()) {
		t.Fatalf("TryBootExisting unexpectedly started missing runtime binary")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read rewritten config: %v", err)
	}
	if !json.Valid(data) {
		t.Fatalf("rewritten config is not valid JSON: %s", string(data))
	}
	var doc struct {
		Inbounds []struct {
			TLS struct {
				CertificatePath string `json:"certificate_path"`
				KeyPath         string `json:"key_path"`
			} `json:"tls"`
		} `json:"inbounds"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal rewritten config: %v", err)
	}
	if got := doc.Inbounds[0].TLS.CertificatePath; got != certPath {
		t.Fatalf("rewritten certificate_path = %q, want %q", got, certPath)
	}
	if got := doc.Inbounds[0].TLS.KeyPath; got != keyPath {
		t.Fatalf("rewritten key_path = %q, want %q", got, keyPath)
	}
}
