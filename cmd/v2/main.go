package main

import (
	"flag"
	"io"
	"log"

	"xw-mcp/internal/bundle"
)

func main() {
	// 解析命令行参数
	pluginsPath := flag.String("plugins", "./plugins", "plugins directory path")
	watchDepth := flag.Int("depth", 3, "watch depth")
	disableLog := flag.Bool("no-log", false, "disable external log output")
	flag.Parse()

	// Stdio模式下禁用外部日志
	if *disableLog {
		log.SetOutput(io.Discard)
	}

	log.Printf("[main] starting bundle context v2...")
	log.Printf("[main] plugins path: %s, depth: %d", *pluginsPath, *watchDepth)

	// 创建BundleContext
	ctx := bundle.NewBundleContext()
	defer ctx.Close()

	// 启动Watcher
	if err := ctx.StartWatcher(*pluginsPath, *watchDepth); err != nil {
		log.Fatalf("[main] start watcher failed: %v", err)
	}

	log.Printf("[main] bundle context started, watching %s", *pluginsPath)

	// 阻塞等待
	select {}
}
