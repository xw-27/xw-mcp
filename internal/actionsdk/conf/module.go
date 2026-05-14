package conf

import (
	"github.com/dop251/goja"

	"xw-mcp/internal/config"
)

// Module 配置模块，实现 ActionModule 接口
type Module struct {
	watcher *config.ConfigWatcher
}

// NewModule 创建 Module
func NewModule(watcher *config.ConfigWatcher) *Module {
	return &Module{watcher: watcher}
}

// Register 将 config 模块注册到 VM 全局
//
// JS 用法:
//
//	config.get("server.port").toInt()
//	config.get("database.host").toString()
//	config.get("feature.enabled").toBool()
//	config.get("features").toMap()
//	config.get("items").toSlice()
//	config.get("server.port").exists()
func (m *Module) Register(vm *goja.Runtime) {
	configObj := vm.NewObject()

	configObj.Set("get", func(path string) interface{} {
		val, ok := m.watcher.Get(path)
		if !ok {
			return vm.NewObject()
		}

		obj := vm.NewObject()
		obj.Set("toString", func() string { return val.ToString() })
		obj.Set("toInt", func() int { return val.ToInt() })
		obj.Set("toFloat", func() float64 { return val.ToFloat() })
		obj.Set("toBool", func() bool { return val.ToBool() })
		obj.Set("toMap", func() map[string]interface{} { return val.ToMap() })
		obj.Set("toSlice", func() []interface{} { return val.ToSlice() })
		obj.Set("exists", func() bool { return val.Exists() })
		return obj
	})

	vm.Set("config", configObj)
}

// Destroy 空实现，configWatcher 由 main.go 管理
func (m *Module) Destroy() {
}