package runtime

import (
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
