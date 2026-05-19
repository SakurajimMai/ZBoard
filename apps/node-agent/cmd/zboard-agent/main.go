package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	"github.com/zboard/node-agent/internal/agent"
	"github.com/zboard/node-agent/internal/config"
)

func main() {
	cfgPath := flag.String("config", "/etc/zboard-agent/agent.env", "path to agent config (key=value)")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	a := agent.New(cfg)
	if err := a.Run(ctx); err != nil {
		log.Fatalf("agent: %v", err)
	}
}
