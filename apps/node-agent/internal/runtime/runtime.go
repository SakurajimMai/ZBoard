// Package runtime supervises the Xray / sing-box subprocess. Config swaps
// happen by writing the new config to disk, comparing the hash, and restarting
// the process. We deliberately keep this dumb — no SIGHUP reload, no graceful
// drain — because the control plane is the source of truth and a 1-2s restart
// is acceptable for the MVP.
package runtime

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Supervisor struct {
	Binary      string
	RuntimeType string // "xray" | "sing-box"
	ConfigFile  string
	WorkDir     string

	mu        sync.Mutex
	cmd       *exec.Cmd
	configSum string // hex sha256 of the last applied config bytes
}

func New(binary, runtimeType, configFile, workDir string) *Supervisor {
	return &Supervisor{
		Binary:      binary,
		RuntimeType: runtimeType,
		ConfigFile:  configFile,
		WorkDir:     workDir,
	}
}

// TryBootExisting starts the runtime if a config file already exists on disk.
// This handles the case where the agent container restarts but the control plane
// has no new sync_config task (previous task already succeeded or failed to max retries).
// Returns true if the runtime was started.
func (s *Supervisor) TryBootExisting(ctx context.Context) bool {
	data, err := os.ReadFile(s.ConfigFile)
	if err != nil || len(data) == 0 {
		return false
	}
	prepared, err := prepareRuntimeConfig(data)
	if err != nil {
		return false
	}
	data = prepared
	_ = os.WriteFile(s.ConfigFile, data, 0o600)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd != nil {
		return false
	}
	if inferred, ok := inferRuntimeType(data); ok && inferred != s.RuntimeType {
		s.setRuntimeType(inferred)
	}
	sum := sha256.Sum256(data)
	s.configSum = hex.EncodeToString(sum[:])
	if err := s.restartLocked(ctx); err != nil {
		return false
	}
	return true
}

// Apply writes `configJSON` to disk if its hash differs from the last applied
// config and restarts the runtime process. It is a no-op when nothing changed.
// Returns (changed, error).
func (s *Supervisor) Apply(ctx context.Context, configJSON []byte) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	prepared, err := prepareRuntimeConfig(configJSON)
	if err != nil {
		return false, err
	}
	configJSON = prepared
	if inferred, ok := inferRuntimeType(configJSON); ok && inferred != s.RuntimeType {
		s.setRuntimeType(inferred)
	}
	sum := sha256.Sum256(configJSON)
	hash := hex.EncodeToString(sum[:])
	if hash == s.configSum && s.cmd != nil {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(s.ConfigFile), 0o755); err != nil {
		return false, fmt.Errorf("mkdir config dir: %w", err)
	}
	if err := os.WriteFile(s.ConfigFile, configJSON, 0o600); err != nil {
		return false, fmt.Errorf("write config: %w", err)
	}
	if err := s.restartLocked(ctx); err != nil {
		return false, err
	}
	s.configSum = hash
	return true, nil
}

// Stop terminates the runtime process if it is running.
func (s *Supervisor) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stopLocked()
}

// IsRunning reports whether the supervised process is alive (best-effort).
func (s *Supervisor) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd == nil || s.cmd.Process == nil {
		return false
	}
	// Signal 0 = liveness probe on Unix.
	if err := s.cmd.Process.Signal(syscall.Signal(0)); err != nil {
		return false
	}
	return true
}

func (s *Supervisor) restartLocked(ctx context.Context) error {
	_ = s.stopLocked()
	args := s.runArgs()
	cmd := exec.CommandContext(ctx, s.Binary, args...)
	cmd.Dir = s.WorkDir
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start runtime: %w", err)
	}
	s.cmd = cmd
	go func() {
		// Don't block the agent; just reap.
		_ = cmd.Wait()
	}()
	// Tiny grace period for the process to initialize before the agent reports
	// "running" upstream.
	time.Sleep(200 * time.Millisecond)
	return nil
}

func (s *Supervisor) stopLocked() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return nil
	}
	if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil && !errors.Is(err, os.ErrProcessDone) {
		// Fall back to Kill on non-Unix platforms or weird states.
		_ = s.cmd.Process.Kill()
	}
	s.cmd = nil
	return nil
}

func (s *Supervisor) runArgs() []string {
	switch s.RuntimeType {
	case "sing-box", "singbox":
		return []string{"run", "-c", s.ConfigFile}
	default:
		return []string{"run", "-config", s.ConfigFile}
	}
}

func (s *Supervisor) setRuntimeType(runtimeType string) {
	s.RuntimeType = runtimeType
	s.Binary = runtimeBinaryForType(s.Binary, runtimeType)
}

func runtimeBinaryForType(current, runtimeType string) string {
	target := "xray"
	switch runtimeType {
	case "sing-box", "singbox":
		target = "sing-box"
	}
	if current == "" {
		return filepath.Join("/usr/local/bin", target)
	}
	base := filepath.Base(current)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	switch name {
	case "xray", "sing-box":
		return strings.TrimSuffix(current, base) + target + ext
	default:
		return current
	}
}

func inferRuntimeType(configJSON []byte) (string, bool) {
	if len(configJSON) == 0 {
		return "", false
	}
	var doc struct {
		Inbounds []struct {
			Type     string `json:"type"`
			Protocol string `json:"protocol"`
		} `json:"inbounds"`
	}
	if err := json.Unmarshal(configJSON, &doc); err != nil || len(doc.Inbounds) == 0 {
		return "", false
	}
	first := doc.Inbounds[0]
	if first.Type != "" {
		return "sing-box", true
	}
	if first.Protocol != "" {
		return "xray", true
	}
	return "", false
}

func ensureRuntimeAssets(configJSON []byte) error {
	for _, pair := range tlsCertificatePairs(configJSON) {
		if err := ensureSelfSignedCertificate(pair.CertPath, pair.KeyPath, pair.ServerName); err != nil {
			return err
		}
	}
	return nil
}

func prepareRuntimeConfig(configJSON []byte) ([]byte, error) {
	prepared, err := normalizeQUICCertificatePaths(configJSON)
	if err != nil {
		return nil, err
	}
	if err := ensureRuntimeAssets(prepared); err != nil {
		return nil, err
	}
	return prepared, nil
}

func normalizeQUICCertificatePaths(configJSON []byte) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(configJSON, &doc); err != nil {
		return nil, fmt.Errorf("parse runtime config: %w", err)
	}
	inbounds, ok := doc["inbounds"].([]any)
	if !ok {
		return configJSON, nil
	}
	changed := false
	certPath := defaultQUICCertificatePath()
	keyPath := defaultQUICKeyPath()
	for _, item := range inbounds {
		in, ok := item.(map[string]any)
		if !ok {
			continue
		}
		protocol, _ := in["type"].(string)
		if protocol != "hysteria2" && protocol != "tuic" {
			continue
		}
		tls, ok := in["tls"].(map[string]any)
		if !ok {
			tls = map[string]any{}
			in["tls"] = tls
			changed = true
		}
		if tls["enabled"] == nil {
			tls["enabled"] = true
			changed = true
		}
		if strings.TrimSpace(fmt.Sprint(tls["certificate_path"])) == "" || tls["certificate_path"] == nil {
			tls["certificate_path"] = certPath
			changed = true
		}
		if strings.TrimSpace(fmt.Sprint(tls["key_path"])) == "" || tls["key_path"] == nil {
			tls["key_path"] = keyPath
			changed = true
		}
	}
	if !changed {
		return configJSON, nil
	}
	out, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("encode runtime config: %w", err)
	}
	return out, nil
}

func defaultQUICCertificatePath() string {
	if v := strings.TrimSpace(os.Getenv("ZBOARD_QUIC_CERT_PATH")); v != "" {
		return v
	}
	return "/etc/zboard-agent/tls/server.crt"
}

func defaultQUICKeyPath() string {
	if v := strings.TrimSpace(os.Getenv("ZBOARD_QUIC_KEY_PATH")); v != "" {
		return v
	}
	return "/etc/zboard-agent/tls/server.key"
}

type tlsCertPair struct {
	CertPath   string
	KeyPath    string
	ServerName string
}

func tlsCertificatePairs(configJSON []byte) []tlsCertPair {
	var doc struct {
		Inbounds []struct {
			TLS struct {
				Enabled         bool   `json:"enabled"`
				ServerName      string `json:"server_name"`
				CertificatePath string `json:"certificate_path"`
				KeyPath         string `json:"key_path"`
			} `json:"tls"`
		} `json:"inbounds"`
	}
	if err := json.Unmarshal(configJSON, &doc); err != nil {
		return nil
	}
	pairs := make([]tlsCertPair, 0, len(doc.Inbounds))
	seen := map[string]bool{}
	for _, in := range doc.Inbounds {
		if !in.TLS.Enabled || in.TLS.CertificatePath == "" || in.TLS.KeyPath == "" {
			continue
		}
		key := in.TLS.CertificatePath + "\x00" + in.TLS.KeyPath
		if seen[key] {
			continue
		}
		seen[key] = true
		pairs = append(pairs, tlsCertPair{
			CertPath:   in.TLS.CertificatePath,
			KeyPath:    in.TLS.KeyPath,
			ServerName: in.TLS.ServerName,
		})
	}
	return pairs
}

func ensureSelfSignedCertificate(certPath, keyPath, serverName string) error {
	if certPath == "" || keyPath == "" {
		return nil
	}
	if fileExists(certPath) && fileExists(keyPath) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(certPath), 0o755); err != nil {
		return fmt.Errorf("mkdir cert dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		return fmt.Errorf("mkdir key dir: %w", err)
	}
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generate tls key: %w", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("generate tls serial: %w", err)
	}
	if strings.TrimSpace(serverName) == "" {
		serverName = "zboard-agent.local"
	}
	now := time.Now().UTC()
	tmpl := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: serverName,
		},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	if ip := net.ParseIP(serverName); ip != nil {
		tmpl.IPAddresses = []net.IP{ip}
	} else {
		tmpl.DNSNames = []string{serverName}
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		return fmt.Errorf("create tls certificate: %w", err)
	}
	certFile, err := os.OpenFile(certPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("write cert: %w", err)
	}
	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		_ = certFile.Close()
		return fmt.Errorf("encode cert: %w", err)
	}
	if err := certFile.Close(); err != nil {
		return fmt.Errorf("close cert: %w", err)
	}
	keyFile, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("write key: %w", err)
	}
	keyDER := x509.MarshalPKCS1PrivateKey(priv)
	if err := pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyDER}); err != nil {
		_ = keyFile.Close()
		return fmt.Errorf("encode key: %w", err)
	}
	if err := keyFile.Close(); err != nil {
		return fmt.Errorf("close key: %w", err)
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
