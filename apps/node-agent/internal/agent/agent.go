// Package agent ties the apiclient + runtime supervisor together: it registers
// on startup, then runs three concurrent loops (heartbeat, task pull/apply,
// traffic report). The traffic loop scrapes the runtime stats gRPC API and
// reports per-user deltas back to the control plane.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/zboard/node-agent/internal/apiclient"
	"github.com/zboard/node-agent/internal/config"
	"github.com/zboard/node-agent/internal/runtime"
	"github.com/zboard/node-agent/internal/stats"
)

type Agent struct {
	Cfg         *config.Config
	Client      *apiclient.Client
	Supervisor  *runtime.Supervisor
	Stats       *stats.Client // nil when StatsAPIAddr is empty
	lastPayload []byte        // last task payload (for port hopping teardown)
}

func New(cfg *config.Config) *Agent {
	a := &Agent{
		Cfg:        cfg,
		Client:     apiclient.New(cfg.APIBaseURL, cfg.NodeID, cfg.NodeSecret),
		Supervisor: runtime.New(cfg.RuntimeBinary, cfg.RuntimeType, cfg.ConfigFile, cfg.WorkDir),
	}
	if cfg.StatsAPIAddr != "" {
		c, err := stats.Dial(cfg.StatsAPIAddr)
		if err != nil {
			log.Printf("stats: dial %s failed (will retry on first scrape): %v", cfg.StatsAPIAddr, err)
		} else {
			a.Stats = c
		}
	}
	return a
}

// Run blocks until ctx is cancelled. It registers, then starts the three loops.
func (a *Agent) Run(ctx context.Context) error {
	if err := a.register(ctx); err != nil {
		return fmt.Errorf("register: %w", err)
	}
	log.Printf("agent registered with control plane (node_id=%d)", a.Cfg.NodeID)

	// If a runtime config already exists on disk (from a previous sync_config),
	// start the runtime immediately instead of waiting for a new task.
	if a.Supervisor.TryBootExisting(ctx) {
		log.Printf("runtime started from existing config file")
	}

	var wg sync.WaitGroup
	wg.Add(3)
	go func() { defer wg.Done(); a.heartbeatLoop(ctx) }()
	go func() { defer wg.Done(); a.taskLoop(ctx) }()
	go func() { defer wg.Done(); a.trafficLoop(ctx) }()
	wg.Wait()

	_ = a.Supervisor.Stop()
	// Teardown port hopping iptables rules on shutdown.
	if a.lastPayload != nil {
		teardownPortHopping(a.lastPayload)
	}
	if a.Stats != nil {
		_ = a.Stats.Close()
	}
	return nil
}

func (a *Agent) register(ctx context.Context) error {
	body := map[string]any{
		"agent_version": a.Cfg.AgentVersion,
		"os_info":       fmt.Sprintf("%s/%s", goos(), goarch()),
		"runtime_info":  a.Cfg.RuntimeType,
	}
	return a.Client.Do(ctx, "/api/agent/v1/register", body, nil)
}

// ===== Loops =====

func (a *Agent) heartbeatLoop(ctx context.Context) {
	t := time.NewTicker(a.Cfg.HeartbeatInterval)
	defer t.Stop()
	for {
		if err := a.heartbeat(ctx); err != nil {
			log.Printf("heartbeat error: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func (a *Agent) heartbeat(ctx context.Context) error {
	status := "running"
	if !a.Supervisor.IsRunning() {
		status = "stopped"
	}
	body := map[string]any{
		"agent_version":  a.Cfg.AgentVersion,
		"runtime_status": status,
		"runtime_info":   a.Cfg.RuntimeType,
	}
	return a.Client.Do(ctx, "/api/agent/v1/heartbeat", body, nil)
}

func (a *Agent) taskLoop(ctx context.Context) {
	t := time.NewTicker(a.Cfg.PullInterval)
	defer t.Stop()
	for {
		if err := a.pullAndApply(ctx); err != nil {
			if isTimeoutError(err) {
				log.Printf("task loop timeout; will retry next pull: %v", err)
			} else {
				log.Printf("task loop error: %v", err)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

type pullResp struct {
	Tasks []struct {
		TaskID        string          `json:"task_id"`
		TaskType      string          `json:"task_type"`
		Payload       json.RawMessage `json:"payload"`
		RuntimeConfig json.RawMessage `json:"runtime_config"`
	} `json:"tasks"`
}

func (a *Agent) pullAndApply(ctx context.Context) error {
	pullCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var resp pullResp
	if err := a.Client.Do(pullCtx, "/api/agent/v1/tasks/pull", map[string]any{}, &resp); err != nil {
		return err
	}
	for _, t := range resp.Tasks {
		switch t.TaskType {
		case "sync_config":
			if len(t.RuntimeConfig) == 0 {
				log.Printf("task %s: missing runtime_config", t.TaskID)
				a.reportResult(ctx, t.TaskID, "failed", "missing runtime_config in task payload")
				continue
			}
			// Teardown old port hopping rules before applying new config.
			if a.lastPayload != nil {
				teardownPortHopping(a.lastPayload)
			}
			if changed, err := a.Supervisor.Apply(ctx, t.RuntimeConfig); err != nil {
				log.Printf("task %s sync_config FAILED: %v", t.TaskID, err)
				a.reportResult(ctx, t.TaskID, "failed", err.Error())
				continue
			} else {
				log.Printf("task %s sync_config applied (changed=%t)", t.TaskID, changed)
			}
			// Apply port hopping iptables rules from the task PAYLOAD (not
			// runtime config — sing-box rejects unknown fields).
			if err := applyPortHopping(t.Payload); err != nil {
				log.Printf("task %s port-hopping FAILED: %v", t.TaskID, err)
				a.reportResult(ctx, t.TaskID, "failed", "config applied but port-hopping failed: "+err.Error())
				continue
			}
			a.lastPayload = t.Payload
			a.reportResult(ctx, t.TaskID, "success", "")
		case "disable_user":
			log.Printf("task %s disable_user acknowledged", t.TaskID)
			a.reportResult(ctx, t.TaskID, "success", "")
		default:
			log.Printf("task %s unknown type %q", t.TaskID, t.TaskType)
			a.reportResult(ctx, t.TaskID, "failed", "unknown task_type "+t.TaskType)
		}
	}
	return nil
}

func (a *Agent) reportResult(ctx context.Context, taskID, status, reason string) {
	body := map[string]any{"status": status}
	if reason != "" {
		body["failed_reason"] = reason
	}
	if err := a.Client.Do(ctx, "/api/agent/v1/tasks/"+taskID+"/result", body, nil); err != nil {
		log.Printf("report task %s result: %v", taskID, err)
	}
}

func (a *Agent) trafficLoop(ctx context.Context) {
	t := time.NewTicker(a.Cfg.TrafficInterval)
	defer t.Stop()
	for {
		if err := a.reportTraffic(ctx); err != nil {
			log.Printf("traffic report error: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// reportTraffic scrapes per-user uplink/downlink from the runtime stats API
// (if configured) and forwards the deltas to the control plane. The stats API
// resets counters on read, so we never double-count even across config swaps.
func (a *Agent) reportTraffic(ctx context.Context) error {
	type item struct {
		UserID        int64 `json:"user_id"`
		UploadDelta   int64 `json:"upload_delta"`
		DownloadDelta int64 `json:"download_delta"`
	}
	out := []item{}
	if a.Stats != nil && a.Supervisor.IsRunning() && supportsStatsAPI(a.Supervisor.Type()) {
		qctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		deltas, err := a.Stats.QueryAndReset(qctx)
		cancel()
		if err != nil {
			log.Printf("stats scrape failed (sending empty report): %v", err)
		} else {
			for _, d := range deltas {
				if d.Upload <= 0 && d.Download <= 0 {
					continue
				}
				out = append(out, item{
					UserID:        d.UserID,
					UploadDelta:   d.Upload,
					DownloadDelta: d.Download,
				})
			}
		}
	}
	body := map[string]any{"items": out}
	return a.Client.Do(ctx, "/api/agent/v1/traffic/report", body, nil)
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func supportsStatsAPI(runtimeType string) bool {
	switch strings.ToLower(strings.TrimSpace(runtimeType)) {
	case "sing-box", "singbox":
		return false
	default:
		return true
	}
}
