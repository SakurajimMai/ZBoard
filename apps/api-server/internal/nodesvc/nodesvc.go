package nodesvc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/zboard/api-server/internal/runtime"
	"github.com/zboard/api-server/internal/store"
)

type Service struct {
	Store *store.Store
}

func New(s *store.Store) *Service { return &Service{Store: s} }

// GenerateSyncTask builds a fresh runtime config (with version + hash), persists
// it, and enqueues a `sync_config` task for the agent to pull.
func (s *Service) GenerateSyncTask(ctx context.Context, nodeID int64) (string, string, error) {
	node, err := s.Store.FindNodeByID(ctx, nodeID)
	if err != nil {
		return "", "", err
	}
	users, err := s.Store.ListNodeUsersByNode(ctx, nodeID)
	if err != nil {
		return "", "", err
	}

	version := time.Now().UTC().Format("20060102150405") + "-" + randHex(4)
	cfgJSON, hash, err := runtime.Build(node, users, version)
	if err != nil {
		return "", "", err
	}
	if _, err := s.Store.CreateRuntimeConfig(ctx, nodeID, version, hash, cfgJSON); err != nil {
		return "", "", err
	}
	taskID := "task-" + version
	payload := fmt.Sprintf(`{"version":%q,"config_hash":%q}`, version, hash)
	if err := s.Store.CreateNodeTask(ctx, taskID, nodeID, "sync_config", payload); err != nil {
		return "", "", err
	}
	return taskID, version, nil
}

// Rollback creates a `sync_config` task that points back at an older version.
func (s *Service) Rollback(ctx context.Context, version string) (string, error) {
	rc, err := s.Store.FindRuntimeConfigByVersion(ctx, version)
	if err != nil {
		return "", err
	}
	taskID := "task-rollback-" + version + "-" + randHex(4)
	payload := fmt.Sprintf(`{"version":%q,"config_hash":%q,"rollback":true}`, rc.Version, rc.ConfigHash)
	if err := s.Store.CreateNodeTask(ctx, taskID, rc.NodeID, "sync_config", payload); err != nil {
		return "", err
	}
	return taskID, nil
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
