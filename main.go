package main

import (
	"flag"
	"log"

	"xw-mcp/internal/bundle"
	"xw-mcp/internal/config"
	"xw-mcp/internal/mcp"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.New(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	defer cfg.Close()

	bundleCtx := bundle.NewBundleContext()
	defer bundleCtx.Close()

	bundlesPath := "bundles"
	if pathVal, ok := cfg.Get("bundles.path"); ok {
		bundlesPath = pathVal.String()
	}

	if err := bundleCtx.StartWatcher(bundlesPath, 2); err != nil {
		log.Fatalf("bundleCtx.StartWatcher failed: %v", err)
	}

	server := mcp.New(bundleCtx, cfg)
	defer server.Close()

	if err := server.Run(); err != nil {
		log.Fatalf("mcp server run failed: %v", err)
	}
}