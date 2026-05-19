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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

// Apply writes `configJSON` to disk if its hash differs from the last applied
// config and restarts the runtime process. It is a no-op when nothing changed.
// Returns (changed, error).
func (s *Supervisor) Apply(ctx context.Context, configJSON []byte) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
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
	cmd.Stdout = os.Stdout
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
