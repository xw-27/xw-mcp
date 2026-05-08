package main

import (
	"fmt"
	"log"

	"xw-mcp/internal/config"
)

func main() {
	cfg, err := config.New("config.yaml")
	if err != nil {
		log.Fatal(err)
	}
	defer cfg.Close()

	port, ok := cfg.Get("server.port")
	if ok {
		fmt.Printf("port: %d\n", port.Int())
	}

	pluginsDir, ok := cfg.Get("server.plugins-dir")
	if ok {
		fmt.Printf("plugins-dir: %s\n", pluginsDir.String())
	}
}