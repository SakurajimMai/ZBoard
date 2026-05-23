package nodesvc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
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
	// Build payload: always include version + hash; add port_hopping metadata
	// when the node has a port_range configured (Hysteria2 port jumping).
	payloadMap := map[string]any{
		"version":     version,
		"config_hash": hash,
	}
	if phMeta := runtime.PortHoppingMeta(node); phMeta != nil {
		payloadMap["port_hopping"] = phMeta
	}
	payloadBytes, _ := json.Marshal(payloadMap)
	payload := string(payloadBytes)
	if err := s.Store.CreateNodeTask(ctx, taskID, nodeID, "sync_config", payload); err != nil {
		return "", "", err
	}
	return taskID, version, nil
}

// GenerateSyncTaskAll enqueues a sync_config task for every active node and
// returns per-node results. Failures on individual nodes don't abort the batch.
type SyncResult struct {
	NodeID  int64  `json:"node_id"`
	Name    string `json:"name"`
	TaskID  string `json:"task_id,omitempty"`
	Version string `json:"version,omitempty"`
	Error   string `json:"error,omitempty"`
}

func (s *Service) GenerateSyncTaskAll(ctx context.Context) ([]SyncResult, error) {
	nodes, err := s.Store.ListActiveNodes(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]SyncResult, 0, len(nodes))
	for _, n := range nodes {
		r := SyncResult{NodeID: n.ID, Name: n.Name}
		taskID, version, err := s.GenerateSyncTask(ctx, n.ID)
		if err != nil {
			r.Error = err.Error()
		} else {
			r.TaskID = taskID
			r.Version = version
		}
		results = append(results, r)
	}
	return results, nil
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
