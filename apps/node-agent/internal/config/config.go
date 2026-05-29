// Package config loads node-agent runtime config from a YAML/JSON-ish env file.
// To keep the agent dependency-free, we use a tiny key=value parser instead of
// pulling in a YAML library.
package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	APIBaseURL        string // e.g. https://control.example.com
	NodeID            int64
	NodeSecret        string // plaintext; library only stores sha256
	HeartbeatInterval time.Duration
	PullInterval      time.Duration
	TrafficInterval   time.Duration
	RuntimeBinary     string // path to xray or sing-box
	RuntimeType       string // "xray" | "sing-box"
	ConfigFile        string // where to write the runtime config
	WorkDir           string // working directory for the runtime process
	AgentVersion      string
	StatsAPIAddr      string // host:port of the runtime's stats gRPC API; empty disables stats scraping
	SingBoxV2RayAPI   bool   // true when the sing-box binary supports experimental.v2ray_api
	statsExplicit     bool
}

// Default returns a Config seeded with sensible defaults.
func Default() *Config {
	return &Config{
		HeartbeatInterval: 30 * time.Second,
		PullInterval:      10 * time.Second,
		TrafficInterval:   60 * time.Second,
		RuntimeBinary:     "/usr/local/bin/xray",
		RuntimeType:       "xray",
		ConfigFile:        "/etc/zboard-agent/runtime.json",
		WorkDir:           "/var/lib/zboard-agent",
		AgentVersion:      "0.1.0",
		StatsAPIAddr:      "127.0.0.1:10085",
	}
}

// Load reads a key=value file (lines starting with # are comments) and
// overlays env variables on top. Both layers use the ZBOARD_AGENT_* prefix.
//
// If the config file does not exist, Load gracefully falls back to env-only
// mode. This supports the Docker `env_file:` pattern, where the compose runtime
// injects the variables into the container environment without creating a file
// at the configured --config path.
func Load(path string) (*Config, error) {
	cfg := Default()
	if path != "" {
		if err := readKVFile(path, cfg); err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
			// File missing is OK — rely on env vars below.
		}
	}
	overlayEnv(cfg)
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func readKVFile(path string, cfg *Config) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		applyKV(cfg, k, v)
	}
	return sc.Err()
}

func overlayEnv(cfg *Config) {
	for _, k := range []string{
		"ZBOARD_AGENT_API_BASE_URL",
		"ZBOARD_AGENT_NODE_ID",
		"ZBOARD_AGENT_NODE_SECRET",
		"ZBOARD_AGENT_HEARTBEAT_INTERVAL",
		"ZBOARD_AGENT_PULL_INTERVAL",
		"ZBOARD_AGENT_TRAFFIC_INTERVAL",
		"ZBOARD_AGENT_RUNTIME_BINARY",
		"ZBOARD_AGENT_RUNTIME_TYPE",
		"ZBOARD_AGENT_CONFIG_FILE",
		"ZBOARD_AGENT_WORK_DIR",
		"ZBOARD_AGENT_VERSION",
		"ZBOARD_AGENT_STATS_API_ADDR",
		"ZBOARD_AGENT_SINGBOX_V2RAY_API",
	} {
		if v, ok := os.LookupEnv(k); ok {
			applyKV(cfg, k, v)
		}
	}
}

func applyKV(c *Config, k, v string) {
	switch k {
	case "ZBOARD_AGENT_API_BASE_URL", "API_BASE_URL":
		c.APIBaseURL = v
	case "ZBOARD_AGENT_NODE_ID", "NODE_ID":
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			c.NodeID = n
		}
	case "ZBOARD_AGENT_NODE_SECRET", "NODE_SECRET":
		c.NodeSecret = v
	case "ZBOARD_AGENT_HEARTBEAT_INTERVAL", "HEARTBEAT_INTERVAL":
		if d, err := time.ParseDuration(v); err == nil {
			c.HeartbeatInterval = d
		}
	case "ZBOARD_AGENT_PULL_INTERVAL", "PULL_INTERVAL":
		if d, err := time.ParseDuration(v); err == nil {
			c.PullInterval = d
		}
	case "ZBOARD_AGENT_TRAFFIC_INTERVAL", "TRAFFIC_INTERVAL":
		if d, err := time.ParseDuration(v); err == nil {
			c.TrafficInterval = d
		}
	case "ZBOARD_AGENT_RUNTIME_BINARY", "RUNTIME_BINARY":
		c.RuntimeBinary = v
	case "ZBOARD_AGENT_RUNTIME_TYPE", "RUNTIME_TYPE":
		c.RuntimeType = v
	case "ZBOARD_AGENT_CONFIG_FILE", "CONFIG_FILE":
		c.ConfigFile = v
	case "ZBOARD_AGENT_WORK_DIR", "WORK_DIR":
		c.WorkDir = v
	case "ZBOARD_AGENT_VERSION", "AGENT_VERSION":
		c.AgentVersion = v
	case "ZBOARD_AGENT_STATS_API_ADDR", "STATS_API_ADDR":
		c.StatsAPIAddr = v
		c.statsExplicit = true
	case "ZBOARD_AGENT_SINGBOX_V2RAY_API", "SINGBOX_V2RAY_API":
		c.SingBoxV2RayAPI = parseBool(v)
	}
}

func (c *Config) validate() error {
	if c.APIBaseURL == "" {
		return fmt.Errorf("api_base_url is required")
	}
	if c.NodeID <= 0 {
		return fmt.Errorf("node_id is required")
	}
	if c.NodeSecret == "" {
		return fmt.Errorf("node_secret is required")
	}
	if isSingBoxRuntime(c.RuntimeType, c.RuntimeBinary) {
		if !c.SingBoxV2RayAPI {
			c.StatsAPIAddr = ""
		}
	}
	c.APIBaseURL = strings.TrimRight(c.APIBaseURL, "/")
	return nil
}

func parseBool(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func isSingBoxRuntime(runtimeType, runtimeBinary string) bool {
	t := strings.ToLower(strings.TrimSpace(runtimeType))
	if t == "sing-box" || t == "singbox" {
		return true
	}
	b := strings.ToLower(filepathBase(runtimeBinary))
	return b == "sing-box" || b == "singbox"
}

func filepathBase(path string) string {
	path = strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
	if path == "" {
		return ""
	}
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}
