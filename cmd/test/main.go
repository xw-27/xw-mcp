package main

import (
	"log"
	"time"

	"xw-mcp/internal/bundle"
)

func main() {
	ctx := bundle.NewBundleContext()
	defer ctx.Close()

	if err := ctx.StartWatcher("./plugins", 3); err != nil {
		log.Fatalf("start watcher failed: %v", err)
	}

	log.Printf("[test] waiting for plugins to load...")
	time.Sleep(2 * time.Second)

	// Test hello tool
	result, err := ctx.DoAction("tool", "test-plugin_hello", map[string]interface{}{
		"name": "World",
	})
	if err != nil {
		log.Fatalf("[test] hello failed: %v", err)
	}
	log.Printf("[test] hello result: %v", result)

	// Test math-plugin add
	result, err = ctx.DoAction("tool", "math-plugin_add", map[string]interface{}{
		"augend": 10, "addend": 20,
	})
	if err != nil {
		log.Fatalf("[test] add failed: %v", err)
	}
	log.Printf("[test] add result: %v", result)
}