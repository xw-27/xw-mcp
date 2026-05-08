package main

import (
	"flag"
	"log"

	"xw-mcp/internal/config"
	"xw-mcp/internal/mcp"
	"xw-mcp/internal/plugin"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.New(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	defer cfg.Close()

	pm, err := plugin.New(cfg, 3)
	if err != nil {
		log.Fatalf("plugin.New failed: %v", err)
	}
	defer pm.Close()

	server := mcp.New(pm, cfg)
	defer server.Close()

	if err := server.Run(); err != nil {
		log.Fatalf("mcp server run failed: %v", err)
	}
}