package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Load 应当在文件存在时读取文件,在文件缺失时降级到纯 env 模式。
// 这是 docker compose `env_file:` 部署的关键路径——容器里没有 /etc/zboard-agent/agent.env
// 这个物理文件,只有注入到环境变量的同名键。

func TestLoad_FileMissing_FallsBackToEnv(t *testing.T) {
	t.Setenv("ZBOARD_AGENT_API_BASE_URL", "https://panel.example.com")
	t.Setenv("ZBOARD_AGENT_NODE_ID", "42")
	t.Setenv("ZBOARD_AGENT_NODE_SECRET", "s3cret")

	cfg, err := Load("/nonexistent/path/agent.env")
	if err != nil {
		t.Fatalf("expected fallback to env, got error: %v", err)
	}
	if cfg.APIBaseURL != "https://panel.example.com" {
		t.Errorf("APIBaseURL = %q", cfg.APIBaseURL)
	}
	if cfg.NodeID != 42 {
		t.Errorf("NodeID = %d", cfg.NodeID)
	}
	if cfg.NodeSecret != "s3cret" {
		t.Errorf("NodeSecret = %q", cfg.NodeSecret)
	}
}

func TestLoad_FileMissing_NoEnv_ReturnsValidationError(t *testing.T) {
	// 防止宿主环境污染
	t.Setenv("ZBOARD_AGENT_API_BASE_URL", "")
	t.Setenv("ZBOARD_AGENT_NODE_ID", "")
	t.Setenv("ZBOARD_AGENT_NODE_SECRET", "")

	if _, err := Load("/nonexistent/path/agent.env"); err == nil {
		t.Fatal("expected validation error when neither file nor env provide config")
	}
}

func TestLoad_FilePresent_OverlayedByEnv(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "agent.env")
	content := "ZBOARD_AGENT_API_BASE_URL=https://from-file\nZBOARD_AGENT_NODE_ID=1\nZBOARD_AGENT_NODE_SECRET=file-secret\n"
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("ZBOARD_AGENT_NODE_SECRET", "env-wins")

	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIBaseURL != "https://from-file" {
		t.Errorf("APIBaseURL = %q", cfg.APIBaseURL)
	}
	if cfg.NodeSecret != "env-wins" {
		t.Errorf("env should overlay file, got NodeSecret = %q", cfg.NodeSecret)
	}
}

func TestLoad_SingBoxDisablesDefaultStatsAPI(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "agent.env")
	content := strings.Join([]string{
		"ZBOARD_AGENT_API_BASE_URL=https://panel.example.com",
		"ZBOARD_AGENT_NODE_ID=1",
		"ZBOARD_AGENT_NODE_SECRET=file-secret",
		"ZBOARD_AGENT_RUNTIME_TYPE=sing-box",
		"ZBOARD_AGENT_RUNTIME_BINARY=/usr/local/bin/sing-box",
	}, "\n")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.StatsAPIAddr != "" {
		t.Fatalf("sing-box should disable default stats API, got %q", cfg.StatsAPIAddr)
	}
}

func TestLoad_SingBoxKeepsExplicitStatsAPI(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "agent.env")
	content := strings.Join([]string{
		"ZBOARD_AGENT_API_BASE_URL=https://panel.example.com",
		"ZBOARD_AGENT_NODE_ID=1",
		"ZBOARD_AGENT_NODE_SECRET=file-secret",
		"ZBOARD_AGENT_RUNTIME_TYPE=sing-box",
		"ZBOARD_AGENT_RUNTIME_BINARY=/usr/local/bin/sing-box",
		"ZBOARD_AGENT_STATS_API_ADDR=127.0.0.1:20085",
	}, "\n")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.StatsAPIAddr != "127.0.0.1:20085" {
		t.Fatalf("explicit sing-box stats API should be preserved, got %q", cfg.StatsAPIAddr)
	}
}
