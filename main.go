package main

import (
	"flag"
	"log"

	"xw-mcp/internal/bundle"
	"xw-mcp/internal/config"
	"xw-mcp/internal/mcp"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "config.yaml", "config file path")
	flag.Parse()

	// 加载配置文件
	cfg, err := config.NewWatcher(*configPath)
	if err != nil {
		log.Fatalf("[main] load config failed: %v", err)
	}
	defer cfg.Close()

	// 获取配置
	bundleDir := "bundles"
	if val, ok := cfg.Get("server.bundle-dir"); ok {
		bundleDir = val.ToString()
	}

	watchDepth := 3
	if val, ok := cfg.Get("server.watch-depth"); ok {
		watchDepth = val.ToInt()
	}

	log.Printf("[main] starting...")
	log.Printf("[main] bundle dir: %s, watch depth: %d", bundleDir, watchDepth)

	// 创建 BundleContext
	ctx := bundle.NewBundleContext(cfg)
	defer ctx.Close()

	// 创建 MCP Server
	mcpServer := mcp.New(ctx, cfg)
	defer mcpServer.Close()

	// 启动文件监控
	if err := ctx.StartWatcher(bundleDir, watchDepth); err != nil {
		log.Fatalf("[main] start watcher failed: %v", err)
	}

	log.Printf("[main] started, watching %s", bundleDir)

	// 启动 MCP Server（阻塞）
	if err := mcpServer.Run(); err != nil {
		log.Fatalf("[main] mcp server failed: %v", err)
	}
}
