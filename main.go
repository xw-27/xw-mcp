package main

import (
	"fmt"
	"log"

	"xw-mcp/internal/config"
	"xw-mcp/internal/plugin"
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

	// 测试 plugin 模块 - 通过 New 方法传入回调
	pm, err := plugin.New("./plugins", 3, func(data plugin.PluginEventData) {
		switch data.Event {
		case plugin.EventPluginAdd:
			fmt.Printf("[Event] Plugin Added: %s\n", data.Plugin.Name)
		case plugin.EventPluginUpdate:
			fmt.Printf("[Event] Plugin Updated: %s\n", data.Plugin.Name)
		case plugin.EventPluginDelete:
			fmt.Printf("[Event] Plugin Deleted: %s\n", data.OldPlugin.Name)
		}
	})
	if err != nil {
		log.Fatalf("plugin.New failed: %v", err)
	}
	defer pm.Close()

	plugins := pm.Plugins()
	fmt.Printf("loaded %d plugins:\n", len(plugins))
	for _, p := range plugins {
		fmt.Printf("  - %s: %s\n", p.Name, p.Description)
	}

	// 测试数学工具
	testCases := []struct {
		name   string
		params map[string]interface{}
	}{
		{"add", map[string]interface{}{"a": 10, "b": 5}},
		{"subtract", map[string]interface{}{"a": 10, "b": 5}},
		{"multiply", map[string]interface{}{"a": 10, "b": 5}},
		{"divide", map[string]interface{}{"a": 10, "b": 5}},
		{"power", map[string]interface{}{"base": 2, "exponent": 8}},
		{"sqrt", map[string]interface{}{"value": 16}},
		{"abs", map[string]interface{}{"value": -5}},
		{"max", map[string]interface{}{"values": []interface{}{3, 1, 4, 1, 5, 9, 2, 6}}},
		{"min", map[string]interface{}{"values": []interface{}{3, 1, 4, 1, 5, 9, 2, 6}}},
		{"round", map[string]interface{}{"value": 3.14159, "decimals": 2}},
	}

	for _, tc := range testCases {
		result, err := pm.Call(tc.name, tc.params)
		if err != nil {
			fmt.Printf("%s failed: %v\n", tc.name, err)
		} else {
			fmt.Printf("%s result: %v\n", tc.name, result)
		}
	}
}