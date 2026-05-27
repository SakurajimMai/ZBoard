// Package runtime supervises the Xray / sing-box subprocess. Config swaps
// happen by writing the new config to disk, comparing the hash, and restarting
// the process. We deliberately keep this dumb — no SIGHUP reload, no graceful
// drain — because the control plane is the source of truth and a 1-2s restart
// is acceptable for the MVP.
package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
